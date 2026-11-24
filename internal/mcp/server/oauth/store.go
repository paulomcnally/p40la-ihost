package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	storeFile     = "oauth.json"
	accessTTL     = time.Hour
	refreshTTL    = 30 * 24 * time.Hour
	codeTTL       = 5 * time.Minute
	clientIDBytes = 16
	tokenBytes    = 32
)

var errNotFound = errors.New("no encontrado")

type client struct {
	ID           string    `json:"client_id"`
	Name         string    `json:"client_name,omitempty"`
	RedirectURIs []string  `json:"redirect_uris"`
	CreatedAt    time.Time `json:"created_at"`
}

type code struct {
	ClientID    string    `json:"client_id"`
	RedirectURI string    `json:"redirect_uri"`
	Challenge   string    `json:"challenge"`
	Scope       string    `json:"scope,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type token struct {
	RefreshHash string    `json:"refresh_hash"`
	ClientID    string    `json:"client_id"`
	Subject     string    `json:"subject"`
	Scope       string    `json:"scope,omitempty"`
	AccessUntil time.Time `json:"access_until"`
	RefreshEnd  time.Time `json:"refresh_end"`
}

type storeData struct {
	Clients map[string]client `json:"clients"`
	Codes   map[string]code   `json:"codes"`
	Tokens  map[string]token  `json:"tokens"`
}

// store persiste clientes, códigos y tokens en un JSON 0600.
type store struct {
	path string

	mu   sync.Mutex
	data storeData
}

func newStore(dir string) (*store, error) {
	if dir == "" {
		return nil, fmt.Errorf("directorio de estado vacío")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("crear directorio de estado: %w", err)
	}
	s := &store{
		path: filepath.Join(dir, storeFile),
		data: storeData{
			Clients: make(map[string]client),
			Codes:   make(map[string]code),
			Tokens:  make(map[string]token),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(s.path); err == nil {
		if err := os.Chmod(s.path, 0o600); err != nil {
			return nil, fmt.Errorf("permisos de estado OAuth: %w", err)
		}
	}
	return s, nil
}

func (s *store) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("leer estado OAuth: %w", err)
	}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("decodificar estado OAuth: %w", err)
	}
	if s.data.Clients == nil {
		s.data.Clients = make(map[string]client)
	}
	if s.data.Codes == nil {
		s.data.Codes = make(map[string]code)
	}
	if s.data.Tokens == nil {
		s.data.Tokens = make(map[string]token)
	}
	s.cleanup(time.Now())
	return nil
}

func (s *store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("codificar estado OAuth: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("escribir estado OAuth: %w", err)
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		return fmt.Errorf("permisos de estado OAuth: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("guardar estado OAuth: %w", err)
	}
	return nil
}

func (s *store) cleanup(now time.Time) {
	for hash, c := range s.data.Codes {
		if now.After(c.ExpiresAt) {
			delete(s.data.Codes, hash)
		}
	}
	for hash, t := range s.data.Tokens {
		if now.After(t.RefreshEnd) {
			delete(s.data.Tokens, hash)
		}
	}
}

func (s *store) createClient(name string, redirects []string, now time.Time) (client, error) {
	id, err := randomID(clientIDBytes)
	if err != nil {
		return client{}, err
	}
	c := client{ID: id, Name: name, RedirectURIs: redirects, CreatedAt: now}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup(now)
	s.data.Clients[id] = c
	if err := s.save(); err != nil {
		return client{}, err
	}
	return c, nil
}

func (s *store) getClient(id string) (client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.data.Clients[id]
	return c, ok
}

func (s *store) createCode(c client, redirectURI, challenge, scope string, now time.Time) (string, error) {
	raw, err := randomID(tokenBytes)
	if err != nil {
		return "", err
	}
	hash := hashToken(raw)
	entry := code{
		ClientID:    c.ID,
		RedirectURI: redirectURI,
		Challenge:   challenge,
		Scope:       scope,
		ExpiresAt:   now.Add(codeTTL),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup(now)
	s.data.Codes[hash] = entry
	if err := s.save(); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *store) consumeCode(raw string, now time.Time) (code, bool) {
	hash := hashToken(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data.Codes[hash]
	if !ok || now.After(entry.ExpiresAt) {
		delete(s.data.Codes, hash)
		return code{}, false
	}
	delete(s.data.Codes, hash)
	if err := s.save(); err != nil {
		return code{}, false
	}
	return entry, true
}

type issuedToken struct {
	Access     string
	Refresh    string
	Scope      string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func (s *store) issueToken(clientID, subject, scope string, now time.Time) (issuedToken, error) {
	access, err := randomID(tokenBytes)
	if err != nil {
		return issuedToken{}, err
	}
	refresh, err := randomID(tokenBytes)
	if err != nil {
		return issuedToken{}, err
	}
	entry := token{
		RefreshHash: hashToken(refresh),
		ClientID:    clientID,
		Subject:     subject,
		Scope:       scope,
		AccessUntil: now.Add(accessTTL),
		RefreshEnd:  now.Add(refreshTTL),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanup(now)
	s.data.Tokens[hashToken(access)] = entry
	if err := s.save(); err != nil {
		return issuedToken{}, err
	}
	return issuedToken{Access: access, Refresh: refresh, Scope: scope, AccessTTL: accessTTL, RefreshTTL: refreshTTL}, nil
}

func (s *store) validateAccess(raw string, now time.Time) (token, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data.Tokens[hashToken(raw)]
	if !ok || now.After(entry.AccessUntil) {
		return token{}, false
	}
	return entry, true
}

func (s *store) rotateRefresh(raw, clientID, subject string, now time.Time) (issuedToken, bool) {
	refreshHash := hashToken(raw)

	s.mu.Lock()
	defer s.mu.Unlock()

	var accessHash string
	var found token
	for ah, entry := range s.data.Tokens {
		if entry.RefreshHash == refreshHash {
			accessHash, found = ah, entry
			break
		}
	}
	if accessHash == "" || now.After(found.RefreshEnd) || (clientID != "" && found.ClientID != clientID) {
		return issuedToken{}, false
	}
	delete(s.data.Tokens, accessHash)

	access, err := randomID(tokenBytes)
	if err != nil {
		return issuedToken{}, false
	}
	refresh, err := randomID(tokenBytes)
	if err != nil {
		return issuedToken{}, false
	}
	s.data.Tokens[hashToken(access)] = token{
		RefreshHash: hashToken(refresh),
		ClientID:    found.ClientID,
		Subject:     subject,
		Scope:       found.Scope,
		AccessUntil: now.Add(accessTTL),
		RefreshEnd:  now.Add(refreshTTL),
	}
	if err := s.save(); err != nil {
		return issuedToken{}, false
	}
	return issuedToken{Access: access, Refresh: refresh, Scope: found.Scope, AccessTTL: accessTTL, RefreshTTL: refreshTTL}, true
}

func (s *store) revokeAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Tokens = make(map[string]token)
	s.data.Codes = make(map[string]code)
	return s.save()
}

func randomID(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
