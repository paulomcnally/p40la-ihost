package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	testPassword = "demo-password"
	testRedirect = "https://claude.ai/api/mcp/auth_callback"
)

type testProvider struct {
	provider *Provider
	server   *httptest.Server
	clientID string
}

func newTestProvider(t *testing.T) *testProvider {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := New(Config{
		Issuer:       "http://127.0.0.1:9999",
		Subject:      "owner",
		PasswordHash: string(hash),
		StateDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(provider.Handler())
	t.Cleanup(server.Close)

	tp := &testProvider{provider: provider, server: server}
	tp.clientID = tp.register(t, testRedirect)
	return tp
}

func (tp *testProvider) register(t *testing.T, redirect string) string {
	t.Helper()
	body := fmt.Sprintf(`{"client_name":"Claude","redirect_uris":[%q],"token_endpoint_auth_method":"none"}`, redirect)
	resp, err := http.Post(tp.server.URL+"/register", "application/json", strings.NewReader(body))
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
	if out.ClientID == "" {
		t.Fatal("client_id vacío")
	}
	return out.ClientID
}

func pkcePair(t *testing.T) (verifier, challenge string) {
	t.Helper()
	verifier = base64.RawURLEncoding.EncodeToString([]byte("demo-verifier-0123456789-0123456789-0123456789"))
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func (tp *testProvider) authorize(t *testing.T, verifier, challenge, password string) *http.Response {
	t.Helper()
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {tp.clientID},
		"redirect_uri":          {testRedirect},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"demo-state"},
		"scope":                 {"mcp:read mcp:write"},
		"password":              {password},
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.PostForm(tp.server.URL+"/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func (tp *testProvider) token(t *testing.T, form url.Values) (int, map[string]any) {
	t.Helper()
	resp, err := http.PostForm(tp.server.URL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, out
}

func codeFromRedirect(t *testing.T, resp *http.Response) string {
	t.Helper()
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
		t.Fatalf("sin code en %s", location)
	}
	if got := location.Query().Get("state"); got != "demo-state" {
		t.Fatalf("state=%q", got)
	}
	return code
}

func TestAuthorizationCodeFlow(t *testing.T) {
	tp := newTestProvider(t)
	verifier, challenge := pkcePair(t)

	code := codeFromRedirect(t, tp.authorize(t, verifier, challenge, testPassword))
	status, body := tp.token(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tp.clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {verifier},
	})
	if status != http.StatusOK {
		t.Fatalf("token status=%d body=%v", status, body)
	}
	access, _ := body["access_token"].(string)
	refresh, _ := body["refresh_token"].(string)
	if access == "" || refresh == "" {
		t.Fatalf("tokens vacíos: %v", body)
	}
	subject, ok := tp.provider.ValidateToken(t.Context(), access)
	if !ok || subject != "owner" {
		t.Fatalf("validate: ok=%v subject=%q", ok, subject)
	}

	// El código es de un solo uso.
	status, _ = tp.token(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tp.clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {verifier},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("código reutilizado status=%d", status)
	}

	// Refresh rotativo.
	status, rotated := tp.token(t, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {tp.clientID},
	})
	if status != http.StatusOK {
		t.Fatalf("refresh status=%d body=%v", status, rotated)
	}
	newAccess, _ := rotated["access_token"].(string)
	if newAccess == "" || newAccess == access {
		t.Fatal("el access token no rotó")
	}
	status, _ = tp.token(t, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
		"client_id":     {tp.clientID},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("refresh reutilizado status=%d", status)
	}
}

func TestPKCERejectsWrongVerifier(t *testing.T) {
	tp := newTestProvider(t)
	verifier, challenge := pkcePair(t)
	code := codeFromRedirect(t, tp.authorize(t, verifier, challenge, testPassword))
	status, body := tp.token(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tp.clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {strings.Repeat("x", 43)},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%v", status, body)
	}
}

func TestWrongPasswordIsRejected(t *testing.T) {
	tp := newTestProvider(t)
	_, challenge := pkcePair(t)
	resp := tp.authorize(t, strings.Repeat("v", 43), challenge, "incorrecta")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestRegisterRejectsUnknownRedirect(t *testing.T) {
	tp := newTestProvider(t)
	body := `{"client_name":"X","redirect_uris":["https://evil.example/callback"]}`
	resp, err := http.Post(tp.server.URL+"/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestMetadata(t *testing.T) {
	tp := newTestProvider(t)
	for _, path := range []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
	} {
		resp, err := http.Get(tp.server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status=%d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestExpiredTokenIsRejected(t *testing.T) {
	tp := newTestProvider(t)
	verifier, challenge := pkcePair(t)
	code := codeFromRedirect(t, tp.authorize(t, verifier, challenge, testPassword))
	status, body := tp.token(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tp.clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {verifier},
	})
	if status != http.StatusOK {
		t.Fatal(status)
	}
	access := body["access_token"].(string)
	tp.provider.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, ok := tp.provider.ValidateToken(t.Context(), access); ok {
		t.Fatal("token expirado aceptado")
	}
}

func TestRefreshWithoutClientID(t *testing.T) {
	tp := newTestProvider(t)
	verifier, challenge := pkcePair(t)
	code := codeFromRedirect(t, tp.authorize(t, verifier, challenge, testPassword))
	status, body := tp.token(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {tp.clientID},
		"redirect_uri":  {testRedirect},
		"code_verifier": {verifier},
	})
	if status != http.StatusOK {
		t.Fatal(status)
	}
	refresh := body["refresh_token"].(string)
	status, rotated := tp.token(t, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	})
	if status != http.StatusOK {
		t.Fatalf("refresh sin client_id status=%d body=%v", status, rotated)
	}
}

// TestAuthorizeFormIncludesRequiredFields reproduce el envío del navegador:
// toma solo los inputs del formulario, agrega la contraseña y espera el code.
// Regresión: el formulario no enviaba response_type ni code_challenge_method.
func TestAuthorizeFormIncludesRequiredFields(t *testing.T) {
	tp := newTestProvider(t)
	_, challenge := pkcePair(t)
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {tp.clientID},
		"redirect_uri":          {testRedirect},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"demo-state"},
		"scope":                 {"mcp:read"},
	}
	resp, err := http.Get(tp.server.URL + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d body=%s", resp.StatusCode, raw)
	}

	form := url.Values{}
	for _, m := range regexp.MustCompile(`<input type="hidden" name="([^"]+)" value="([^"]*)">`).FindAllStringSubmatch(string(raw), -1) {
		form.Set(m[1], m[2])
	}
	form.Set("password", testPassword)

	if got := form.Get("response_type"); got != "code" {
		t.Fatalf("response_type=%q", got)
	}
	if got := form.Get("code_challenge_method"); got != "S256" {
		t.Fatalf("code_challenge_method=%q", got)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	posted, err := client.PostForm(tp.server.URL+"/authorize", form)
	if err != nil {
		t.Fatal(err)
	}
	codeFromRedirect(t, posted)
}
