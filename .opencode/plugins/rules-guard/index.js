import fs from "node:fs"
import path from "node:path"
import { Plugin } from "@opencode/plugin"

/**
 * rules-guard — Prevención en tiempo de ejecución de las reglas del proyecto.
 *
 * Fuente de verdad: docs/rules/00-index.md
 * Fail-safe: si la lógica del guard falla, la operación continúa.
 */

const MARK = "[rules-guard]"

const RULES_BANNER = `${MARK} Reglas del proyecto activas.
- Índice: docs/rules/00-index.md (leelo antes de crear specs o código).
- Prohibido: borrar data/app.db; pkill; git push --force; git reset --hard.
- Prohibido: editar public/i18n/ (usar frontend/public/i18n/).
- Prohibido: <select> nativo y checkbox nativo fuera de Toggle.tsx.
- Cada spec debe incluir la tabla "Cumplimiento de reglas" con IDs de docs/rules/.`

function block(message, rule) {
  throw new Error(
    `${MARK} ⛔ ${message}\n` +
      `Regla: ${rule}\n` +
      `Referencia: docs/rules/00-index.md\n` +
      `Si la acción está autorizada, pedí confirmación explícita al usuario y usá la alternativa indicada.`
  )
}

function isOwnError(error) {
  return error instanceof Error && error.message.startsWith(MARK)
}

function isMainCheckout(directory) {
  try {
    return fs.statSync(path.join(directory, ".git")).isDirectory()
  } catch {
    return false
  }
}

function guardCommand(command, directory) {
  const cmd = String(command ?? "")
  if (!cmd.trim()) return

  // DATA-NO-DROP: nunca borrar la base del usuario
  if (/\brm\b[^\n]*\bdata\b/i.test(cmd)) {
    block("No se puede borrar ni reemplazar la base de datos del usuario.", "DATA-NO-DROP")
  }

  // TEST-KILLALL: pkill falla silenciosamente en este entorno
  if (/(^|[;&|]\s*)pkill\b/.test(cmd)) {
    block("Usá killall, nunca pkill. Ej: killall p40la-ihost", "TEST-KILLALL")
  }

  // DELIV-CONFIRM: nada de force push
  if (/\bgit\s+push\b[^\n]*(--force\b|--force-with-lease|\s-f(\s|$))/.test(cmd)) {
    block("Prohibido git push --force / --force-with-lease.", "DELIV-CONFIRM")
  }

  // DELIV-WORKTREE: reset --hard solo con orden explícita del usuario
  if (/\bgit\s+reset\s+--hard\b/.test(cmd)) {
    block(
      "git reset --hard descarta trabajo sin commitear. Requiere orden explícita del usuario.",
      "DELIV-WORKTREE"
    )
  }

  // DELIV-WORKTREE: el checkout principal es solo lectura
  if (
    isMainCheckout(directory) &&
    /\bgit\s+(checkout|switch|reset|stash|clean)\b/.test(cmd)
  ) {
    block(
      "El checkout principal (main) es solo lectura. Trabajá en un worktree de spec.",
      "DELIV-WORKTREE"
    )
  }
}

function guardWrite(filePath, content) {
  const p = String(filePath ?? "").replaceAll("\\", "/")
  if (!p) return

  // UI-I18N: public/i18n es salida del build
  const isPublicI18n =
    /(^|\/)public\/i18n\//.test(p) && !/(^|\/)frontend\/public\/i18n\//.test(p)
  if (isPublicI18n) {
    block(
      "public/i18n/ es salida del build y se sobrescribe. Editá frontend/public/i18n/{es,en}.json.",
      "UI-I18N"
    )
  }

  if (!/\.(tsx|jsx)$/.test(p)) return
  const text = String(content ?? "")

  // UI-SELECT: dropdown nativo prohibido
  if (/<select[\s>]/.test(text)) {
    block("Prohibido <select> nativo. Usá el componente Select del proyecto.", "UI-SELECT")
  }

  // UI-TOGGLE: checkbox nativo prohibido fuera de Toggle.tsx
  if (/type\s*=\s*["']checkbox["']/.test(text) && !/components\/Toggle\.tsx$/.test(p)) {
    block("Prohibido <input type=\"checkbox\">. Usá el componente Toggle.", "UI-TOGGLE")
  }
}

export default Plugin.define({
  id: "rules-guard",

  async setup(ctx) {
    const injected = new Set()

    // Recordatorio de reglas una vez por sesión (refuerzo del Paso 0).
    try {
      await ctx.session.hook("context", (event) => {
        try {
          const sessionID = event?.sessionID
          if (sessionID && injected.has(sessionID)) return
          if (sessionID) injected.add(sessionID)
          if (Array.isArray(event?.system)) {
            event.system.push({ type: "text", text: RULES_BANNER })
          }
        } catch {
          // fail-safe
        }
      })
    } catch {
      // si la API no expone el hook, seguimos con tool.execute.before
    }

    await ctx.tool.hook("execute.before", (event) => {
      try {
        const tool = String(event?.tool ?? "")
        const input = event?.input ?? {}

        if (tool === "bash" || tool === "shell") {
          guardCommand(input?.command, ctx.location?.directory)
          return
        }

        if (tool === "write" || tool === "edit" || tool === "patch") {
          const filePath = input?.filePath ?? input?.path ?? input?.file
          const content = [input?.content, input?.newString, input?.oldString]
            .filter((v) => typeof v === "string")
            .join("\n")
          guardWrite(filePath, content)
        }
      } catch (error) {
        if (isOwnError(error)) throw error
        // fail-safe: un fallo del guard no debe romper la sesión
      }
    })
  },
})
