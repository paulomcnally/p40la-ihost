#!/usr/bin/env bash
#
# Tests de detección de cambios de scripts/release.sh (SPEC-160).
#
# Levanta repositorios git desechables en un directorio temporal (con un origen
# bare local) y verifica la matriz de detección por artefacto sin tocar Docker
# Hub, sin red y sin publicar nada.
#
# Uso:
#   ./scripts/test-release-detection.sh
#
# Requisitos: bash, git. Opcionales: python3 + PyYAML (validación de YAML).
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

WORK_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/release-detection.XXXXXX")"
cleanup() {
  rm -rf "${WORK_ROOT}"
}
trap cleanup EXIT

PASSED=0
FAILED=0

pass() {
  PASSED=$((PASSED + 1))
  printf 'ok   %s\n' "$1"
}

fail() {
  FAILED=$((FAILED + 1))
  printf 'FAIL %s\n' "$1"
  if [[ -n "${2:-}" && -f "${2:-}" ]]; then
    printf '     --- salida ---\n'
    sed 's/^/     /' "$2" | tail -30
    printf '     --------------\n'
  fi
}

# Crea un fixture: repo con allowlists cubiertas, tags base y origen bare local.
new_fixture() {
  local name="$1"
  local dir="${WORK_ROOT}/${name}"
  local repo="${dir}/repo"

  mkdir -p "${repo}/scripts" "${repo}/frontend" "${repo}/cmd/server" "${repo}/internal/mcp" \
    "${repo}/deploy/mcp/tailscale" "${repo}/docs/specs" "${repo}/public" "${repo}/migrations" \
    "${repo}/.github/workflows" "${dir}/bin" "${dir}/nohooks"

  cp "${REPO_ROOT}/scripts/release.sh" "${repo}/scripts/release.sh"
  chmod +x "${repo}/scripts/release.sh"

  # Stub de gh: el modo real solo necesita `gh auth status` antes de no publicar.
  cat > "${dir}/bin/gh" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
  chmod +x "${dir}/bin/gh"

  # Evita abrir un navegador real durante los tests.
  for opener in xdg-open google-chrome open; do
    cat > "${dir}/bin/${opener}" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "${dir}/bin/${opener}"
  done

  git init --bare -q "${dir}/origin.git"
  git --git-dir="${dir}/origin.git" symbolic-ref HEAD refs/heads/main

  (
    cd "${repo}"
    git init -q
    git symbolic-ref HEAD refs/heads/main
    git config user.name "Demo Developer"
    git config user.email "demo@example.com"
    git config core.hooksPath "${dir}/nohooks"
    git remote add origin "${dir}/origin.git"

    cat > docker-compose.yml <<'YAML'
services:
  app:
    image: paulomcnally/p40la-ihost:latest
    environment:
      - VERSION=0.5.21
YAML
    cat > frontend/package.json <<'JSON'
{
  "name": "demo-frontend",
  "version": "0.5.21"
}
JSON
    printf 'package main\n\nfunc main() {}\n' > cmd/server/main.go
    printf 'package mcp\n' > internal/mcp/client.go
    printf 'FROM tailscale/tailscale:latest\n' > deploy/mcp/tailscale/Dockerfile
    printf '{"TCP": {"443": {"HTTPS": true}}}\n' > deploy/mcp/tailscale/serve.json
    printf '# SPEC demo\n' > docs/specs/SPEC-000-demo.md
    printf '<html></html>\n' > public/index.html
    printf 'CREATE TABLE demo (id INTEGER);\n' > migrations/0001_demo.up.sql
    printf 'module demo\n\ngo 1.23\n' > go.mod
    printf 'FROM scratch\n' > Dockerfile
    printf 'FROM scratch\n' > Dockerfile.mcp
    printf 'name: demo\n' > .github/workflows/docker-publish.yml
    printf 'name: demo\n' > .github/workflows/mcp-publish.yml
    printf 'name: demo\n' > .github/workflows/tailscale-publish.yml

    git add -A
    git commit -q -m "estado inicial"
    git tag -a v0.5.21 -m "Release v0.5.21"
    git tag -a mcp-v0.6.1 -m "Release MCP v0.6.1"
    git tag -a tailscale-v0.1.0 -m "Release Tailscale wrapper v0.1.0"
    git push -q origin main --tags
  )

  printf '%s' "${dir}"
}

# Agrega un commit que toca una ruta del fixture.
touch_commit() {
  local dir="$1"
  local path="$2"
  local repo="${dir}/repo"
  mkdir -p "$(dirname "${repo}/${path}")"
  printf '# cambio sintetico\n' >> "${repo}/${path}"
  (
    cd "${repo}"
    git add -A
    git commit -q -m "cambio en ${path}"
  )
}

# Ejecuta release.sh en el fixture y deja la salida en <dir>/out.log.
# Devuelve el código de salida.
run_release() {
  local dir="$1"
  shift
  local rc=0
  (
    cd "${dir}/repo"
    PATH="${dir}/bin:${PATH}" ./scripts/release.sh "$@"
  ) >"${dir}/out.log" 2>&1 || rc=$?
  printf '%s' "${rc}"
}

contains() {
  grep -qF -- "$2" "$1"
}

contains_count() {
  grep -cF -- "$2" "$1" || true
}

not_contains() {
  ! grep -qF -- "$2" "$1"
}

# -----------------------------------------------------------------------------
# Tests estáticos
# -----------------------------------------------------------------------------

if bash -n "${REPO_ROOT}/scripts/release.sh"; then
  pass "bash -n scripts/release.sh"
else
  fail "bash -n scripts/release.sh"
fi

if bash -n "${REPO_ROOT}/scripts/test-release-detection.sh"; then
  pass "bash -n scripts/test-release-detection.sh"
else
  fail "bash -n scripts/test-release-detection.sh"
fi

if not_contains "${REPO_ROOT}/.github/workflows/mcp-publish.yml" "build-tailscale"; then
  pass "mcp-publish.yml no tiene el job build-tailscale"
else
  fail "mcp-publish.yml no tiene el job build-tailscale"
fi

if contains "${REPO_ROOT}/.github/workflows/tailscale-publish.yml" 'tags: ["tailscale-v*"]'; then
  pass "tailscale-publish.yml se dispara con tailscale-v*"
else
  fail "tailscale-publish.yml se dispara con tailscale-v*"
fi

if contains "${REPO_ROOT}/.github/workflows/tailscale-publish.yml" "linux/arm/v7" \
  && contains "${REPO_ROOT}/.github/workflows/tailscale-publish.yml" "linux/arm64" \
  && contains "${REPO_ROOT}/.github/workflows/tailscale-publish.yml" "linux/amd64"; then
  pass "tailscale-publish.yml construye las 3 plataformas (DELIV-DOCKER)"
else
  fail "tailscale-publish.yml construye las 3 plataformas (DELIV-DOCKER)"
fi

if command -v python3 >/dev/null 2>&1 && python3 -c 'import yaml' >/dev/null 2>&1; then
  if python3 - "${REPO_ROOT}" <<'PY'
import sys
import yaml

root = sys.argv[1]
for name in ("docker-publish.yml", "mcp-publish.yml", "tailscale-publish.yml"):
    with open(f"{root}/.github/workflows/{name}") as fh:
        yaml.safe_load(fh)
PY
  then
    pass "YAML de los tres workflows válido"
  else
    fail "YAML de los tres workflows válido"
  fi
else
  printf 'skip validación YAML (python3 + PyYAML no disponible)\n'
fi

# -----------------------------------------------------------------------------
# CA-REQ-001: solo docs -> nada para liberar
# -----------------------------------------------------------------------------

fx=$(new_fixture 'docs-only')
touch_commit "${fx}" 'docs/specs/SPEC-999-solo-docs.md'

rc=$(run_release "${fx}" --check)
if [[ "${rc}" -eq 0 ]] && [[ "$(contains_count "${fx}/out.log" 'decisión: omitir (sin cambios)')" -eq 3 ]] \
  && not_contains "${fx}/out.log" 'publicar ('; then
  pass "docs-only: --check omite los tres artefactos"
else
  fail "docs-only: --check omite los tres artefactos" "${fx}/out.log"
fi

if contains "${fx}/out.log" 'Tags que se crearían: ninguno (nada para liberar)'; then
  pass "docs-only: --check no crea tags"
else
  fail "docs-only: --check no crea tags" "${fx}/out.log"
fi

# CA-REQ-006 (modo real): nada para liberar, sin commits ni tags.
head_before=$(git -C "${fx}/repo" rev-parse HEAD)
tags_before=$(git -C "${fx}/repo" tag --list | wc -l | tr -d ' ')
remote_before=$(git -C "${fx}/repo" ls-remote --tags origin | wc -l | tr -d ' ')

rc=$(run_release "${fx}")
head_after=$(git -C "${fx}/repo" rev-parse HEAD)
tags_after=$(git -C "${fx}/repo" tag --list | wc -l | tr -d ' ')
remote_after=$(git -C "${fx}/repo" ls-remote --tags origin | wc -l | tr -d ' ')

if [[ "${rc}" -eq 0 ]] && contains "${fx}/out.log" 'Nada para liberar' \
  && [[ "${head_before}" == "${head_after}" && "${tags_before}" == "${tags_after}" && "${remote_before}" == "${remote_after}" ]]; then
  pass "docs-only: release real imprime «Nada para liberar» sin commits, tags ni push"
else
  fail "docs-only: release real imprime «Nada para liberar» sin commits, tags ni push" "${fx}/out.log"
fi

# CA-REQ-007: --check no muta refs ni hace git fetch.
fx=$(new_fixture 'check-readonly')
touch_commit "${fx}" 'cmd/server/main.go'

refs_before=$(git -C "${fx}/repo" show-ref)
rc=0
(
  cd "${fx}/repo"
  PATH="${fx}/bin:${PATH}" GIT_TRACE=1 ./scripts/release.sh --check
) >"${fx}/out.log" 2>&1 || rc=$?
refs_after=$(git -C "${fx}/repo" show-ref)

if [[ "${rc}" -eq 0 && "${refs_before}" == "${refs_after}" ]]; then
  pass "--check no cambia refs locales (git show-ref idéntico)"
else
  fail "--check no cambia refs locales (git show-ref idéntico)" "${fx}/out.log"
fi

if [[ "$(grep -c 'built-in: git fetch' "${fx}/out.log" || true)" -eq 0 ]] \
  && [[ "$(grep -c 'built-in: git ls-remote' "${fx}/out.log" || true)" -ge 1 ]]; then
  pass "--check usa git ls-remote y nunca git fetch"
else
  fail "--check usa git ls-remote y nunca git fetch" "${fx}/out.log"
fi

# CA-REQ-001: un cambio real de la app sí versiona la app.
if contains "${fx}/out.log" '[app] versión a crear: v0.5.22' \
  && contains "${fx}/out.log" '[app] decisión: publicar (cambios detectados)' \
  && contains "${fx}/out.log" '[MCP] decisión: omitir (sin cambios)' \
  && contains "${fx}/out.log" '[Tailscale] decisión: omitir (sin cambios)'; then
  pass "app-only (cmd/server): publica la app y omite MCP y Tailscale"
else
  fail "app-only (cmd/server): publica la app y omite MCP y Tailscale" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-002: solo internal/mcp -> publica MCP, no la app
# -----------------------------------------------------------------------------

fx=$(new_fixture 'mcp-only')
touch_commit "${fx}" 'internal/mcp/client.go'

rc=$(run_release "${fx}" --check)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[MCP] último tag: mcp-v0.6.1' \
  && contains "${fx}/out.log" '[MCP] versión a crear: mcp-v0.6.2' \
  && contains "${fx}/out.log" '[MCP] decisión: publicar (cambios detectados)' \
  && contains "${fx}/out.log" '[app] decisión: omitir (sin cambios)'; then
  pass "mcp-only: publica mcp-v0.6.2 y no versiona la app"
else
  fail "mcp-only: publica mcp-v0.6.2 y no versiona la app" "${fx}/out.log"
fi

if contains "${fx}/out.log" 'Archivos que se actualizarían: ninguno (solo tags)'; then
  pass "mcp-only: no se actualizan docker-compose.yml ni frontend/package.json"
else
  fail "mcp-only: no se actualizan docker-compose.yml ni frontend/package.json" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-003: solo deploy/mcp/tailscale -> publica el wrapper, no el MCP
# -----------------------------------------------------------------------------

fx=$(new_fixture 'tailscale-only')
touch_commit "${fx}" 'deploy/mcp/tailscale/serve.json'

rc=$(run_release "${fx}" --check)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[Tailscale] último tag: tailscale-v0.1.0' \
  && contains "${fx}/out.log" '[Tailscale] versión a crear: tailscale-v0.1.1' \
  && contains "${fx}/out.log" '[Tailscale] decisión: publicar (cambios detectados)' \
  && contains "${fx}/out.log" '[MCP] decisión: omitir (sin cambios)' \
  && contains "${fx}/out.log" '[app] decisión: omitir (sin cambios)'; then
  pass "tailscale-only: publica tailscale-v0.1.1 sin tocar MCP ni app"
else
  fail "tailscale-only: publica tailscale-v0.1.1 sin tocar MCP ni app" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-004: MCP + tailscale con tags independientes
# -----------------------------------------------------------------------------

fx=$(new_fixture 'mcp-and-tailscale')
touch_commit "${fx}" 'internal/mcp/client.go'
touch_commit "${fx}" 'deploy/mcp/tailscale/Dockerfile'

rc=$(run_release "${fx}" --check)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[MCP] versión a crear: mcp-v0.6.2' \
  && contains "${fx}/out.log" '[Tailscale] versión a crear: tailscale-v0.1.1' \
  && contains "${fx}/out.log" 'Tags que se crearían: mcp-v0.6.2, tailscale-v0.1.1'; then
  pass "mcp+tailscale: dos tags independientes y la app omitida"
else
  fail "mcp+tailscale: dos tags independientes y la app omitida" "${fx}/out.log"
fi

# CA-REQ-011: --no-tailscale omite el wrapper aunque haya cambios.
rc=$(run_release "${fx}" --check --no-tailscale)
if [[ "${rc}" -eq 0 ]] && contains "${fx}/out.log" '[Tailscale] decisión: omitir (--no-tailscale)'; then
  pass "--no-tailscale omite el wrapper aunque haya cambios"
else
  fail "--no-tailscale omite el wrapper aunque haya cambios" "${fx}/out.log"
fi

# CA-REQ-011: MCP_VERSION=skip omite el MCP.
rc=0
(
  cd "${fx}/repo"
  PATH="${fx}/bin:${PATH}" MCP_VERSION=skip ./scripts/release.sh --check
) >"${fx}/out.log" 2>&1 || rc=$?
if [[ "${rc}" -eq 0 ]] && contains "${fx}/out.log" '[MCP] decisión: omitir (--no-mcp)'; then
  pass "MCP_VERSION=skip omite el MCP"
else
  fail "MCP_VERSION=skip omite el MCP" "${fx}/out.log"
fi

# CA-REQ-014: --list-tags sin plan de release.
fx=$(new_fixture 'list-tags')
touch_commit "${fx}" 'internal/mcp/client.go'

rc=$(run_release "${fx}" --list-tags)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[app] último tag: v0.5.21' \
  && contains "${fx}/out.log" '[app] último tag: v0.5.21 — cambios pendientes: no' \
  && contains "${fx}/out.log" '[MCP] último tag: mcp-v0.6.1 — cambios pendientes: sí (1 archivo(s))' \
  && contains "${fx}/out.log" '[Tailscale] último tag: tailscale-v0.1.0 — cambios pendientes: no'; then
  pass "--list-tags muestra último tag y cambios pendientes por artefacto"
else
  fail "--list-tags muestra último tag y cambios pendientes por artefacto" "${fx}/out.log"
fi

if not_contains "${fx}/out.log" 'Tags que se crearían'; then
  pass "--list-tags no arma plan de release"
else
  fail "--list-tags no arma plan de release" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-005 / CA-REQ-010: versiones explícitas (argumentos y entorno)
# -----------------------------------------------------------------------------

fx=$(new_fixture 'forced-version')
touch_commit "${fx}" 'docs/specs/SPEC-999-solo-docs.md'

rc=$(run_release "${fx}" 0.9.9 --check)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[app] versión a crear: v0.9.9' \
  && contains "${fx}/out.log" '[app] decisión: publicar (forzado por versión explícita)' \
  && contains "${fx}/out.log" 'AVISO: app: no se detectaron cambios'; then
  pass "release.sh 0.9.9 fuerza la app sin cambios y avisa"
else
  fail "release.sh 0.9.9 fuerza la app sin cambios y avisa" "${fx}/out.log"
fi

rc=0
(
  cd "${fx}/repo"
  PATH="${fx}/bin:${PATH}" TAILSCALE_VERSION=0.2.0 ./scripts/release.sh --check
) >"${fx}/out.log" 2>&1 || rc=$?
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[Tailscale] versión a crear: tailscale-v0.2.0' \
  && contains "${fx}/out.log" '[Tailscale] decisión: publicar (forzado por versión explícita)' \
  && contains "${fx}/out.log" '[app] decisión: omitir (sin cambios)'; then
  pass "TAILSCALE_VERSION=0.2.0 fuerza el wrapper y no la app"
else
  fail "TAILSCALE_VERSION=0.2.0 fuerza el wrapper y no la app" "${fx}/out.log"
fi

# CA-REQ-010: tercer argumento posicional = versión del wrapper.
rc=$(run_release "${fx}" 0.5.22 0.7.0 0.2.0 --check)
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" '[app] versión a crear: v0.5.22' \
  && contains "${fx}/out.log" '[MCP] versión a crear: mcp-v0.7.0' \
  && contains "${fx}/out.log" '[Tailscale] versión a crear: tailscale-v0.2.0' \
  && contains "${fx}/out.log" 'Tags que se crearían: v0.5.22, mcp-v0.7.0, tailscale-v0.2.0'; then
  pass "tres argumentos posicionales: app, MCP y wrapper"
else
  fail "tres argumentos posicionales: app, MCP y wrapper" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-013: tag existente aborta antes de crear nada
# -----------------------------------------------------------------------------

fx=$(new_fixture 'tag-collision')
touch_commit "${fx}" 'cmd/server/main.go'
git -C "${fx}/repo" tag -a v0.5.22 -m "tag local duplicado"

rc=$(run_release "${fx}" --check)
if [[ "${rc}" -ne 0 ]] && contains "${fx}/out.log" 'El tag v0.5.22 ya existe localmente'; then
  pass "tag local existente aborta la ejecución"
else
  fail "tag local existente aborta la ejecución" "${fx}/out.log"
fi

fx=$(new_fixture 'tag-collision-remote')
touch_commit "${fx}" 'cmd/server/main.go'
# Crea el tag v0.5.22 solo en el remoto y fuerza esa versión explícita.
(
  cd "${fx}/repo"
  git tag -a v0.5.22 -m "tag remoto duplicado"
  git push -q origin v0.5.22
  git tag -d v0.5.22 >/dev/null
)

rc=$(run_release "${fx}" 0.5.22 --check)
if [[ "${rc}" -ne 0 ]] && contains "${fx}/out.log" 'El tag v0.5.22 ya existe en origin'; then
  pass "tag remoto existente aborta la ejecución"
else
  fail "tag remoto existente aborta la ejecución" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# CA-REQ-009: propagación de fallos entre workflows (sin tocar GitHub)
# -----------------------------------------------------------------------------

write_gh_stub() {
  local dir="$1"
  local conclusion="$2"

  cat > "${dir}/bin/gh" <<EOF
#!/usr/bin/env bash
if [[ "\$1" == "auth" ]]; then exit 0; fi
if [[ "\$1" == "run" && "\$2" == "list" ]]; then
  sha=\$(git rev-parse HEAD)
  printf '[{"url":"https://github.com/paulomcnally/p40la-ihost/actions/runs/123456","headBranch":"mock","headSha":"%s"}]\\n' "\$sha"
  exit 0
fi
if [[ "\$1" == "run" && "\$2" == "view" ]]; then
  case "\$*" in
    *"--json status"*) echo "completed" ;;
    *"--json conclusion"*) echo "${conclusion}" ;;
    *) echo "" ;;
  esac
  exit 0
fi
exit 0
EOF
  chmod +x "${dir}/bin/gh"

  # curl falla a propósito: la guarda de Docker Hub se saltea con aviso (sin red).
  cat > "${dir}/bin/curl" <<'EOF'
#!/usr/bin/env bash
exit 22
EOF
  chmod +x "${dir}/bin/curl"
}

fx=$(new_fixture 'workflow-failure')
touch_commit "${fx}" 'cmd/server/main.go'
write_gh_stub "${fx}" "failure"

rc=$(run_release "${fx}")
if [[ "${rc}" -ne 0 ]] \
  && contains "${fx}/out.log" "ERROR: App v0.5.22 terminó con conclusión 'failure'" \
  && contains "${fx}/out.log" 'Uno o más workflows fallaron' \
  && git -C "${fx}/repo" rev-parse -q --verify refs/tags/v0.5.22 >/dev/null \
  && git -C "${fx}/repo" ls-remote --tags origin | grep -q 'refs/tags/v0.5.22'; then
  pass "workflow fallido: release.sh sale con error y deja el tag para reintentar"
else
  fail "workflow fallido: release.sh sale con error y deja el tag para reintentar" "${fx}/out.log"
fi

fx=$(new_fixture 'workflow-success')
touch_commit "${fx}" 'cmd/server/main.go'
write_gh_stub "${fx}" "success"

rc=$(run_release "${fx}")
if [[ "${rc}" -eq 0 ]] \
  && contains "${fx}/out.log" 'App v0.5.22: workflow completado con éxito.' \
  && contains "${fx}/out.log" 'App v0.5.22 procesado' \
  && contains "${fx}/out.log" 'MCP: sin cambios, no se publicó.' \
  && grep -q 'VERSION=0.5.22' "${fx}/repo/docker-compose.yml" \
  && git -C "${fx}/repo" ls-remote --tags origin | grep -q 'refs/tags/v0.5.22'; then
  pass "release real de la app: bump, tag pusheado y workflow exitoso"
else
  fail "release real de la app: bump, tag pusheado y workflow exitoso" "${fx}/out.log"
fi

# -----------------------------------------------------------------------------
# Resumen
# -----------------------------------------------------------------------------

printf '\n%d ok, %d fallidos\n' "${PASSED}" "${FAILED}"
if [[ "${FAILED}" -ne 0 ]]; then
  exit 1
fi
