package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/crypto/bcrypt"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

const (
	testPassword = "demo-password"
	testRedirect = "https://claude.ai/api/mcp/auth_callback"
)

type echoInput struct {
	Text string `json:"text" jsonschema:"texto a repetir"`
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

func newTestServer(t *testing.T, rateLimit int) (*Server, *httptest.Server) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Transport:         config.TransportHTTP,
		Listen:            "127.0.0.1:0",
		AdminListen:       "127.0.0.1:0",
		PublicURL:         "http://127.0.0.1:9999",
		OAuthSubject:      "owner",
		OAuthPasswordHash: string(hash),
		StateDir:          t.TempDir(),
		AllowWrites:       true,
		RemoteWrites:      false,
		RateLimit:         rateLimit,
		AuditMaxMB:        1,
		Timeout:           time.Second,
		Location:          time.UTC,
	}
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "echo", Description: "eco"}, func(_ context.Context, _ *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: in.Text}}}, in, nil
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv, err := New(cfg, mcpServer, logger, "test")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		_ = srv.audit.Close()
	})
	return srv, ts
}

func postJSON(t *testing.T, endpoint, token string, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func registerClient(t *testing.T, base string) string {
	t.Helper()
	body := fmt.Sprintf(`{"client_name":"Claude","redirect_uris":[%q],"token_endpoint_auth_method":"none"}`, testRedirect)
	resp, err := http.Post(base+"/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("register status=%d body=%s", resp.StatusCode, raw)
	}
	var out struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.ClientID
}

func obtainToken(t *testing.T, base string) string {
	t.Helper()
	clientID := registerClient(t, base)
	verifier := base64.RawURLEncoding.EncodeToString([]byte("demo-verifier-0123456789-0123456789-0123456789"))
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {testRedirect},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"demo-state"},
		"password":              {testPassword},
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.PostForm(base+"/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("authorize status=%d body=%s", resp.StatusCode, raw)
	}
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatal("sin code")
	}

	tokenResp, err := http.PostForm(base+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {verifier},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.AccessToken == "" {
		t.Fatal("access_token vacío")
	}
	return out.AccessToken
}

func TestMCPRequiresToken(t *testing.T) {
	_, ts := newTestServer(t, 10)
	resp := postJSON(t, ts.URL+"/mcp", "", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, "resource_metadata=") {
		t.Fatalf("WWW-Authenticate=%q", got)
	}
}

func TestMCPProtocolAndAudit(t *testing.T) {
	srv, ts := newTestServer(t, 10)
	token := obtainToken(t, ts.URL)

	transport := &mcp.StreamableClientTransport{
		Endpoint: ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{
			token: token,
			base:  http.DefaultTransport,
		}},
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "echo" {
		t.Fatalf("tools=%v", tools.Tools)
	}
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "echo", Arguments: map[string]any{"text": "hola"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool error: %v", result.Content)
	}

	raw, err := os.ReadFile(filepath.Join(srv.cfg.StateDir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	line := string(raw)
	if !strings.Contains(line, `"tool":"echo"`) {
		t.Fatalf("auditoría sin tool: %s", line)
	}
	if !strings.Contains(line, `"subject":"owner"`) {
		t.Fatalf("auditoría sin subject: %s", line)
	}
	if strings.Contains(line, "hola") {
		t.Fatalf("la auditoría registró argumentos: %s", line)
	}
}

func TestRateLimit(t *testing.T) {
	_, ts := newTestServer(t, 2)
	token := obtainToken(t, ts.URL)
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	for i := 0; i < 2; i++ {
		resp := postJSON(t, ts.URL+"/mcp", token, body)
		resp.Body.Close()
	}
	resp := postJSON(t, ts.URL+"/mcp", token, body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Fatal("sin Retry-After")
	}
}

func TestBodyLimit(t *testing.T) {
	_, ts := newTestServer(t, 10)
	token := obtainToken(t, ts.URL)
	big := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"x":"` + strings.Repeat("a", maxMCPBody) + `"}}`
	resp := postJSON(t, ts.URL+"/mcp", token, big)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestAdminHealthAndRevoke(t *testing.T) {
	srv, ts := newTestServer(t, 10)
	token := obtainToken(t, ts.URL)

	admin := httptest.NewServer(srv.adminHandler())
	defer admin.Close()

	resp, err := http.Get(admin.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	revoke, err := http.Post(admin.URL+"/admin/revoke", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	if revoke.StatusCode != http.StatusOK {
		t.Fatalf("revoke status=%d", revoke.StatusCode)
	}
	revoke.Body.Close()

	after := postJSON(t, ts.URL+"/mcp", token, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	defer after.Body.Close()
	if after.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token revocado status=%d", after.StatusCode)
	}
}
