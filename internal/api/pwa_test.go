package api

import (
	"crypto/tls"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", name, err)
	}
}

func TestWebmanifestMIMEType(t *testing.T) {
	got := mime.TypeByExtension(".webmanifest")
	if !strings.HasPrefix(got, "application/manifest+json") {
		t.Fatalf("MIME de .webmanifest = %q, se esperaba application/manifest+json", got)
	}
}

func TestPWAManifestHandler(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "manifest.webmanifest", `{"name":"p40la"}`)

	rr := httptest.NewRecorder()
	pwaManifestHandler(dir).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/manifest+json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, se esperaba no-cache", got)
	}
	if !strings.Contains(rr.Body.String(), `"name":"p40la"`) {
		t.Fatalf("body inesperado: %q", rr.Body.String())
	}
}

func TestPWAManifestHandlerMissing(t *testing.T) {
	rr := httptest.NewRecorder()
	pwaManifestHandler(t.TempDir()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, se esperaba 404 (no index.html)", rr.Code)
	}
}

func TestPWAServiceWorkerHandler(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "sw.js", "self.addEventListener('fetch', () => {})")

	rr := httptest.NewRecorder()
	pwaServiceWorkerHandler(dir).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sw.js", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, se esperaba no-cache", got)
	}
	if got := rr.Header().Get("Service-Worker-Allowed"); got != "/" {
		t.Fatalf("Service-Worker-Allowed = %q", got)
	}
}

func TestPWAServiceWorkerHandlerMissing(t *testing.T) {
	rr := httptest.NewRecorder()
	pwaServiceWorkerHandler(t.TempDir()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sw.js", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, se esperaba 404 (no index.html)", rr.Code)
	}
}

func TestSPAHandlerDoesNotCacheIndex(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "<html>p40la</html>")

	rr := httptest.NewRecorder()
	spaHandler(dir).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/settings", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, se esperaba no-cache", got)
	}
	if !strings.Contains(rr.Body.String(), "p40la") {
		t.Fatalf("body inesperado: %q", rr.Body.String())
	}
}

func TestSPAHandlerServesExistingStaticFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "index")
	writeTestFile(t, dir, "favicon.svg", "svg")

	rr := httptest.NewRecorder()
	spaHandler(dir).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/favicon.svg", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, se esperaba no-cache", got)
	}
	if rr.Body.String() != "svg" {
		t.Fatalf("body = %q", rr.Body.String())
	}
}

func TestWithCacheControl(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	rr := httptest.NewRecorder()
	withCacheControl("public, max-age=31536000, immutable", next).
		ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app-123.js", nil))

	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestIsSecureRequest(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		withTLS    bool
		proto      string
		want       bool
	}{
		{name: "TLS directo", remoteAddr: "192.0.2.10:1234", withTLS: true, want: true},
		{name: "loopback con X-Forwarded-Proto https", remoteAddr: "127.0.0.1:4567", proto: "https", want: true},
		{name: "loopback con X-Forwarded-Proto HTTPS", remoteAddr: "127.0.0.1:4567", proto: "HTTPS", want: true},
		{name: "loopback HTTP", remoteAddr: "127.0.0.1:4567", proto: "http", want: false},
		{name: "IPv6 loopback https", remoteAddr: "[::1]:4567", proto: "https", want: true},
		{name: "LAN con header falsificado", remoteAddr: "192.0.2.10:4567", proto: "https", want: false},
		{name: "sin address", remoteAddr: "", proto: "https", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.withTLS {
				req.TLS = &tls.ConnectionState{}
			}
			if tc.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.proto)
			}
			if got := isSecureRequest(req); got != tc.want {
				t.Fatalf("isSecureRequest() = %v, se esperaba %v", got, tc.want)
			}
		})
	}
}

func TestSetSessionCookieSecureOnHTTPS(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = "127.0.0.1:4567"
	req.Header.Set("X-Forwarded-Proto", "https")

	setSessionCookie(rr, req, &http.Cookie{Name: "session", Value: "demo"})

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, se esperaba 1", len(cookies))
	}
	if !cookies[0].Secure {
		t.Fatal("la cookie debería ser Secure sobre HTTPS")
	}
}

func TestSetSessionCookieNotSecureOnLAN(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	req.RemoteAddr = "192.0.2.10:4567"
	req.Header.Set("X-Forwarded-Proto", "https")

	setSessionCookie(rr, req, &http.Cookie{Name: "session", Value: "demo"})

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, se esperaba 1", len(cookies))
	}
	if cookies[0].Secure {
		t.Fatal("la cookie no debería ser Secure en HTTP LAN")
	}
}

func TestBuildRouterPWARoutes(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "index")
	writeTestFile(t, dir, "manifest.webmanifest", `{"name":"p40la"}`)
	writeTestFile(t, dir, "sw.js", "self.addEventListener('fetch', () => {})")

	handler := &Handler{webhooks: &WebhookHandlers{}}
	router := BuildRouter(handler, nil, dir)

	t.Run("manifest", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/manifest+json" {
			t.Fatalf("Content-Type = %q", got)
		}
	})

	t.Run("service worker", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sw.js", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("Cache-Control = %q", got)
		}
	})

	t.Run("fallback SPA", func(t *testing.T) {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/settings", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "index") {
			t.Fatalf("body = %q", rr.Body.String())
		}
	})
}

func TestBuildRouterMissingServiceWorkerIs404(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", "index")

	handler := &Handler{webhooks: &WebhookHandlers{}}
	router := BuildRouter(handler, nil, dir)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/sw.js", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, se esperaba 404 (no index.html)", rr.Code)
	}
}
