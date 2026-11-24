package oauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const maxAuthBody = 64 << 10

type registerRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

func (p *Provider) register(w http.ResponseWriter, r *http.Request) {
	if allowed, retry := p.registerLimit.Allow(clientIP(r)); !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())+1))
		writeOAuthError(w, http.StatusTooManyRequests, "too_many_requests", "demasiados registros; intentá más tarde")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "JSON inválido")
		return
	}
	if len(req.RedirectURIs) == 0 || len(req.RedirectURIs) > 5 {
		writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris debe tener entre 1 y 5 entradas")
		return
	}
	for _, uri := range req.RedirectURIs {
		if !allowedRedirect(uri) {
			writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uri no permitido")
			return
		}
	}
	if method := strings.TrimSpace(req.TokenEndpointAuthMethod); method != "" && method != "none" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "solo se admiten clientes públicos (none)")
		return
	}

	c, err := p.store.createClient(strings.TrimSpace(req.ClientName), req.RedirectURIs, p.now())
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "no se pudo registrar el cliente")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  c.ID,
		"client_name":                c.Name,
		"redirect_uris":              c.RedirectURIs,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"client_id_issued_at":        c.CreatedAt.Unix(),
	})
}

type authorizeData struct {
	ClientID     string
	RedirectURI  string
	Challenge    string
	State        string
	Scope        string
	ClientName   string
	RedirectHost string
	FormAction   string
	Error        string
}

var authorizeTemplate = template.Must(template.New("authorize").Parse(`<!doctype html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Autorizar p40la MCP</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; display: grid; place-items: center; min-height: 100vh; background: #f2f2f7; color: #111; }
  main { background: #fff; padding: 2rem; border-radius: 1rem; box-shadow: 0 10px 30px rgba(0,0,0,.08); max-width: 26rem; width: calc(100% - 2rem); }
  h1 { font-size: 1.25rem; margin: 0 0 1rem; }
  p { line-height: 1.5; }
  code { background: #f2f2f7; padding: .1rem .3rem; border-radius: .25rem; }
  label { display: block; margin: 1.25rem 0 .5rem; font-weight: 600; }
  input[type=password] { width: 100%; box-sizing: border-box; padding: .75rem; border: 1px solid #c7c7cc; border-radius: .6rem; font-size: 1rem; }
  button { margin-top: 1.25rem; width: 100%; padding: .85rem; border: 0; border-radius: .6rem; background: #007aff; color: #fff; font-size: 1rem; font-weight: 600; }
  .error { color: #b00020; font-weight: 600; }
  .hint { color: #6e6e73; font-size: .85rem; }
  @media (prefers-color-scheme: dark) {
    body { background: #000; color: #f2f2f7; }
    main { background: #1c1c1e; box-shadow: none; }
    code { background: #2c2c2e; }
    input[type=password] { background: #2c2c2e; color: #f2f2f7; border-color: #3a3a3c; }
  }
</style>
</head>
<body>
<main>
  <h1>Autorizar p40la MCP</h1>
  <p><strong>{{.ClientName}}</strong> solicita acceso a tu instancia de p40la-ihost.</p>
  <p>Al continuar serás redirigido a <code>{{.RedirectHost}}</code>.</p>
  {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
  <form method="post" action="/authorize">
    <input type="hidden" name="response_type" value="code">
    <input type="hidden" name="code_challenge_method" value="S256">
    <input type="hidden" name="client_id" value="{{.ClientID}}">
    <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
    <input type="hidden" name="code_challenge" value="{{.Challenge}}">
    <input type="hidden" name="state" value="{{.State}}">
    <input type="hidden" name="scope" value="{{.Scope}}">
    <label for="password">Contraseña de administración</label>
    <input id="password" type="password" name="password" autocomplete="current-password" autofocus required>
    <button type="submit">Autorizar</button>
  </form>
  <p class="hint">Acceso solicitado: {{.Scope}}</p>
</main>
</body>
</html>`))

func (p *Provider) authorizeGet(w http.ResponseWriter, r *http.Request) {
	data, err := p.parseAuthorize(r.URL.Query())
	if err != nil {
		p.renderAuthorize(w, http.StatusBadRequest, authorizeData{Error: err.Error()})
		return
	}
	p.renderAuthorize(w, http.StatusOK, data)
}

func (p *Provider) authorizePost(w http.ResponseWriter, r *http.Request) {
	if allowed, retry := p.loginLimit.Allow(clientIP(r)); !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())+1))
		p.renderAuthorize(w, http.StatusTooManyRequests, authorizeData{Error: "demasiados intentos; esperá un minuto"})
		return
	}
	if err := r.ParseForm(); err != nil {
		p.renderAuthorize(w, http.StatusBadRequest, authorizeData{Error: "formulario inválido"})
		return
	}
	data, err := p.parseAuthorize(r.PostForm)
	if err != nil {
		p.renderAuthorize(w, http.StatusBadRequest, authorizeData{Error: err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(p.passwordHash), []byte(r.PostForm.Get("password"))); err != nil {
		data.Error = "contraseña incorrecta"
		p.renderAuthorize(w, http.StatusUnauthorized, data)
		return
	}

	c, ok := p.store.getClient(data.ClientID)
	if !ok {
		p.renderAuthorize(w, http.StatusBadRequest, authorizeData{Error: "cliente desconocido"})
		return
	}
	codeRaw, err := p.store.createCode(c, data.RedirectURI, data.Challenge, data.Scope, p.now())
	if err != nil {
		p.renderAuthorize(w, http.StatusInternalServerError, authorizeData{Error: "no se pudo emitir el código"})
		return
	}

	redirect, err := url.Parse(data.RedirectURI)
	if err != nil {
		p.renderAuthorize(w, http.StatusBadRequest, authorizeData{Error: "redirect_uri inválido"})
		return
	}
	query := redirect.Query()
	query.Set("code", codeRaw)
	if data.State != "" {
		query.Set("state", data.State)
	}
	redirect.RawQuery = query.Encode()
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
}

func (p *Provider) parseAuthorize(values url.Values) (authorizeData, error) {
	data := authorizeData{
		ClientID:    strings.TrimSpace(values.Get("client_id")),
		RedirectURI: strings.TrimSpace(values.Get("redirect_uri")),
		Challenge:   strings.TrimSpace(values.Get("code_challenge")),
		State:       values.Get("state"),
		Scope:       strings.TrimSpace(values.Get("scope")),
	}
	if values.Get("response_type") != "code" {
		return data, fmt.Errorf("response_type debe ser code")
	}
	if values.Get("code_challenge_method") != "S256" {
		return data, fmt.Errorf("code_challenge_method debe ser S256")
	}
	if len(data.Challenge) < 43 || len(data.Challenge) > 128 {
		return data, fmt.Errorf("code_challenge inválido")
	}
	if len(data.State) > 512 {
		return data, fmt.Errorf("state demasiado largo")
	}
	c, ok := p.store.getClient(data.ClientID)
	if !ok {
		return data, fmt.Errorf("cliente desconocido")
	}
	if !contains(c.RedirectURIs, data.RedirectURI) {
		return data, fmt.Errorf("redirect_uri no registrado para el cliente")
	}
	if data.Scope == "" {
		data.Scope = scopeRead + " " + scopeWrite
	}
	data.ClientName = c.Name
	if data.ClientName == "" {
		data.ClientName = "Aplicación"
	}
	host := data.RedirectURI
	if u, err := url.Parse(data.RedirectURI); err == nil {
		host = u.Host
	}
	data.RedirectHost = host
	data.FormAction = originOf(data.RedirectURI)
	return data, nil
}

func (p *Provider) renderAuthorize(w http.ResponseWriter, status int, data authorizeData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	csp := "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'"
	if data.FormAction != "" {
		// El POST responde 303 al redirect_uri registrado; form-action también
		// valida los redirects, así que debe incluir ese origen.
		csp += " " + data.FormAction
	}
	w.Header().Set("Content-Security-Policy", csp)
	w.WriteHeader(status)
	_ = authorizeTemplate.Execute(w, data)
}

func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	if allowed, retry := p.tokenLimit.Allow(clientIP(r)); !allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())+1))
		writeOAuthError(w, http.StatusTooManyRequests, "too_many_requests", "demasiadas solicitudes; intentá más tarde")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "formulario inválido")
		return
	}

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		p.exchangeCode(w, r)
	case "refresh_token":
		p.exchangeRefresh(w, r)
	default:
		writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type no soportado")
	}
}

func (p *Provider) exchangeCode(w http.ResponseWriter, r *http.Request) {
	rawCode := r.PostForm.Get("code")
	clientID := r.PostForm.Get("client_id")
	redirectURI := r.PostForm.Get("redirect_uri")
	verifier := r.PostForm.Get("code_verifier")
	if rawCode == "" || clientID == "" || redirectURI == "" || verifier == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
		return
	}
	entry, ok := p.store.consumeCode(rawCode, p.now())
	if !ok || entry.ClientID != clientID || entry.RedirectURI != redirectURI {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "código inválido o expirado")
		return
	}
	if !verifyPKCE(verifier, entry.Challenge) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "PKCE inválido")
		return
	}
	p.issue(w, entry.ClientID, entry.Scope)
}

func (p *Provider) exchangeRefresh(w http.ResponseWriter, r *http.Request) {
	rawRefresh := r.PostForm.Get("refresh_token")
	clientID := r.PostForm.Get("client_id")
	if rawRefresh == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "faltan parámetros")
		return
	}
	issued, ok := p.store.rotateRefresh(rawRefresh, clientID, p.subject, p.now())
	if !ok {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token inválido o expirado")
		return
	}
	writeToken(w, issued)
}

func (p *Provider) issue(w http.ResponseWriter, clientID, scope string) {
	issued, err := p.store.issueToken(clientID, p.subject, scope, p.now())
	if err != nil {
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "no se pudo emitir el token")
		return
	}
	writeToken(w, issued)
}

func writeToken(w http.ResponseWriter, issued issuedToken) {
	body := map[string]any{
		"access_token":  issued.Access,
		"token_type":    "Bearer",
		"expires_in":    int(issued.AccessTTL.Seconds()),
		"refresh_token": issued.Refresh,
	}
	if issued.Scope != "" {
		body["scope"] = issued.Scope
	}
	writeJSON(w, http.StatusOK, body)
}

func writeOAuthError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": description,
	})
}

func verifyPKCE(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(challenge)) == 1
}

func allowedRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.User != nil {
		return false
	}
	if u.Scheme == "https" && strings.EqualFold(u.Hostname(), "claude.ai") && u.Port() == "" && u.Path == "/api/mcp/auth_callback" {
		return true
	}
	if u.Scheme == "http" && isLoopback(u.Hostname()) && u.Path == "/callback" {
		return true
	}
	return false
}

func isLoopback(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
