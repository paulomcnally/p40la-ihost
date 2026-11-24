package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func TestAllowedHost(t *testing.T) {
	s := &Server{cfg: config.Config{PublicURL: "https://demo.example.ts.net"}}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := s.allowedHost(next)

	cases := []struct {
		host string
		want int
	}{
		{"demo.example.ts.net", http.StatusNoContent},
		{"demo.example.ts.net:443", http.StatusNoContent},
		{"127.0.0.1:8090", http.StatusNoContent},
		{"[::1]:8090", http.StatusNoContent},
		{"localhost:8090", http.StatusNoContent},
		{"evil.example.com", http.StatusForbidden},
		{"", http.StatusForbidden},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Host = tc.host
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("host %q: got %d, want %d", tc.host, rec.Code, tc.want)
		}
	}
}
