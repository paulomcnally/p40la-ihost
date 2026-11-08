package services

import (
	"encoding/json"
	"errors"

	"github.com/paulomcnally/p40la-ihost-automation-plugins/plugins"
)

// Códigos estables de error de automation (SPEC-121). Viajan en el campo
// error_code de las facturas consultadas; no cambian el contrato HTTP.
const (
	AutomationErrorPortalMigrated  = "portal_migrated"
	AutomationErrorCaptchaRequired = "captcha_required"
	AutomationErrorAuthExpired     = "auth_expired"
)

// AutomationErrorMessage traduce los errores de autenticación del plugin a un
// mensaje accionable. El texto se persiste en automation_bills.error y se
// devuelve en la API; nunca incluye credenciales, cookies ni query strings.
func AutomationErrorMessage(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, plugins.ErrPortalMigrated):
		return "El proveedor migró su portal de login y el acceso automático ya no es válido. Desactivá la consulta automática para evitar errores repetidos."
	case errors.Is(err, plugins.ErrCaptchaRequired):
		return "El proveedor requiere CAPTCHA para iniciar sesión y el acceso automático no es posible. Desactivá la consulta automática."
	case errors.Is(err, plugins.ErrAuthExpired):
		return "La sesión del proveedor expiró. Volvé a configurar la sesión."
	default:
		return err.Error()
	}
}

// AutomationErrorCode devuelve el código estable del fallo, o "" si es un
// error genérico.
func AutomationErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, plugins.ErrPortalMigrated):
		return AutomationErrorPortalMigrated
	case errors.Is(err, plugins.ErrCaptchaRequired):
		return AutomationErrorCaptchaRequired
	case errors.Is(err, plugins.ErrAuthExpired):
		return AutomationErrorAuthExpired
	default:
		return ""
	}
}

// automationErrorRaw arma la metadata que acompaña al error persistido. Se
// guarda en la columna raw (JSON) para no cambiar el esquema SQLite.
func automationErrorRaw(err error) string {
	code := AutomationErrorCode(err)
	if code == "" {
		return "{}"
	}
	raw, marshalErr := json.Marshal(map[string]string{"error_code": code})
	if marshalErr != nil {
		return "{}"
	}
	return string(raw)
}

// automationErrorCodeFromRaw recupera el código persistido en raw.
func automationErrorCodeFromRaw(raw string) string {
	if raw == "" {
		return ""
	}
	var meta struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return ""
	}
	return meta.ErrorCode
}
