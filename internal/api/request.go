package api

import (
	"net"
	"net/http"
	"strings"
)

// isSecureRequest indica si la request llegó por HTTPS. Detrás de Tailscale
// Serve el backend recibe HTTP desde 127.0.0.1, pero el proxy agrega
// X-Forwarded-Proto: https. Solo se confía en ese header cuando la conexión
// viene de loopback, para que un cliente de la LAN no pueda falsificarlo.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host != "127.0.0.1" && host != "::1" {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

// setSessionCookie escribe la cookie de sesión marcándola Secure cuando la
// request original fue HTTPS. En HTTP LAN se mantiene no-Secure para no romper
// el acceso de respaldo (SPEC-129, REQ-011).
func setSessionCookie(w http.ResponseWriter, r *http.Request, cookie *http.Cookie) {
	if cookie == nil {
		return
	}
	if isSecureRequest(r) {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)
}
