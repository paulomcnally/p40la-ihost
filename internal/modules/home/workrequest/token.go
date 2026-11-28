package workrequest

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const uploadTokenVersion = "v1"

type uploadTokenPayload struct {
	HomeID        int64  `json:"hid"`
	WorkRequestID int64  `json:"wid"`
	Exp           int64  `json:"exp"`
	JTI           string `json:"jti"`
	MaxFiles      int    `json:"max_files"`
}

// NewUploadJTI genera un identificador aleatorio de 16 bytes para el token.
func NewUploadJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// MintUploadToken firma un token de carga con HMAC-SHA256.
func MintUploadToken(secret string, claims UploadClaims) (string, error) {
	if secret == "" {
		return "", invalid("upload_link_disabled", "La carga por link no está configurada")
	}
	if claims.MaxFiles < 1 || claims.MaxFiles > MaxUploadFilesPerLink {
		return "", invalid("invalid_request", "max_files debe estar entre 1 y 10")
	}
	payload := uploadTokenPayload{
		HomeID:        claims.HomeID,
		WorkRequestID: claims.WorkRequestID,
		Exp:           claims.ExpiresAt.UTC().Unix(),
		JTI:           claims.JTI,
		MaxFiles:      claims.MaxFiles,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(encoded)
	signed := uploadTokenVersion + "." + payloadPart
	return signed + "." + signUploadToken(secret, signed), nil
}

// ParseUploadToken valida versión, firma, expiración y forma del payload.
func ParseUploadToken(secret, raw string, now time.Time) (*UploadClaims, error) {
	if secret == "" {
		return nil, invalid("upload_link_disabled", "La carga por link no está configurada")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 || parts[0] != uploadTokenVersion || parts[1] == "" || parts[2] == "" {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	signed := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(signUploadToken(secret, signed))) {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	var payload uploadTokenPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	if payload.HomeID <= 0 || payload.WorkRequestID <= 0 || payload.JTI == "" ||
		payload.MaxFiles < 1 || payload.MaxFiles > MaxUploadFilesPerLink {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	expires := time.Unix(payload.Exp, 0).UTC()
	if payload.Exp <= 0 {
		return nil, invalid("invalid_token", "El token de carga no es válido")
	}
	if !expires.After(now) {
		return nil, invalid("expired_token", "El token de carga expiró")
	}
	return &UploadClaims{
		HomeID:        payload.HomeID,
		WorkRequestID: payload.WorkRequestID,
		ExpiresAt:     expires,
		JTI:           payload.JTI,
		MaxFiles:      payload.MaxFiles,
	}, nil
}

func signUploadToken(secret, signed string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signed))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
