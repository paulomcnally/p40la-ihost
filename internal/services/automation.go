package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ErrAutomationAccountNotFound indica que la cuenta no existe.
var ErrAutomationAccountNotFound = errors.New("cuenta de automation no encontrada")

// AutomationService contiene la lógica de lectura del módulo de automation
// unificado (SPEC-110 fase 1). Las operaciones de escritura, plugins y
// scheduler se agregan en las fases siguientes.
type AutomationService struct {
	storage *storage.AutomationStorage
}

func NewAutomationService(st *storage.AutomationStorage) *AutomationService {
	return &AutomationService{storage: st}
}

func (s *AutomationService) ListAccounts(ctx context.Context) ([]models.AutomationAccount, error) {
	accounts, err := s.storage.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if accounts == nil {
		accounts = []models.AutomationAccount{}
	}
	return accounts, nil
}

// ListAccountOptions devuelve las opciones del selector de servicios
// (SPEC-111): institución · identificador · plugin.
func (s *AutomationService) ListAccountOptions(ctx context.Context) ([]models.AutomationAccountOption, error) {
	accounts, err := s.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]models.AutomationAccountOption, 0, len(accounts))
	for _, account := range accounts {
		institution := strings.TrimSpace(account.InstitutionName)
		if institution == "" {
			institution = fmt.Sprintf("Institución %d", account.InstitutionID)
		}
		label := institution + " · " + account.Identifier
		if account.PluginName != nil && *account.PluginName != "" {
			label += " (" + *account.PluginName + ")"
		}
		options = append(options, models.AutomationAccountOption{ID: account.ID, Label: label})
	}
	return options, nil
}

func (s *AutomationService) GetAccount(ctx context.Context, id int64) (*models.AutomationAccount, error) {
	account, err := s.storage.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAutomationAccountNotFound
	}
	return account, nil
}

func (s *AutomationService) ListPlugins(ctx context.Context) ([]models.AutomationPlugin, error) {
	plugins, err := s.storage.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}
	if plugins == nil {
		plugins = []models.AutomationPlugin{}
	}
	return plugins, nil
}

func (s *AutomationService) ListBills(ctx context.Context, accountID int64) ([]models.AutomationBill, error) {
	if _, err := s.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}
	bills, err := s.storage.ListBills(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if bills == nil {
		bills = []models.AutomationBill{}
	}
	return bills, nil
}

func (s *AutomationService) ListWebhookLogs(ctx context.Context, accountID int64, limit, offset int) ([]models.AutomationWebhookLog, error) {
	if _, err := s.GetAccount(ctx, accountID); err != nil {
		return nil, err
	}
	logs, err := s.storage.ListWebhookLogs(ctx, accountID, limit, offset)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []models.AutomationWebhookLog{}
	}
	return logs, nil
}

// Errores de dominio del CRUD de cuentas (SPEC-110).
var (
	ErrAutomationAccountInUse  = errors.New("la cuenta está vinculada a servicios activos")
	ErrAutomationAccountExists = errors.New("ya existe una cuenta con ese identificador en la institución")
)

// AutomationAccountInput son los campos editables de una cuenta.
type AutomationAccountInput struct {
	InstitutionID     int64
	Identifier        string
	Label             string
	Credentials       string
	PluginName        *string
	WebhookURL        *string
	ScheduleTime      *string
	ScheduleEnabled   bool
	ScheduleFrequency string
	ScheduleInterval  *int
	ScheduleDays      *string
}

// AutomationWebhookInput son los campos de webhook/schedule de una cuenta.
type AutomationWebhookInput struct {
	WebhookURL        *string
	ScheduleTime      *string
	ScheduleEnabled   bool
	ScheduleFrequency string
	ScheduleInterval  *int
	ScheduleDays      *string
}

func validateAutomationAccountInput(in AutomationAccountInput) error {
	if in.InstitutionID <= 0 {
		return errors.New("institución requerida")
	}
	if strings.TrimSpace(in.Identifier) == "" {
		return errors.New("identificador requerido")
	}
	return validateAutomationSchedule(in.ScheduleTime, in.ScheduleFrequency, in.ScheduleInterval)
}

func validateAutomationSchedule(scheduleTime *string, frequency string, interval *int) error {
	switch frequency {
	case "", "daily", "every_n_days", "weekly", "monthly":
	default:
		return fmt.Errorf("frecuencia inválida: %s", frequency)
	}
	if scheduleTime != nil && strings.TrimSpace(*scheduleTime) != "" {
		if _, err := time.Parse("15:04", *scheduleTime); err != nil {
			return errors.New("hora de ejecución inválida (formato HH:MM)")
		}
	}
	if frequency == "every_n_days" && (interval == nil || *interval <= 0) {
		return errors.New("para 'every_n_days' el intervalo debe ser mayor a 0")
	}
	return nil
}

func validateWebhookURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("URL de webhook inválida")
	}
	return nil
}

func (s *AutomationService) validateAccountRefs(ctx context.Context, in AutomationAccountInput) error {
	ok, err := s.storage.InstitutionExists(ctx, in.InstitutionID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("la institución %d no existe", in.InstitutionID)
	}
	if in.PluginName != nil && strings.TrimSpace(*in.PluginName) != "" {
		exists, err := s.storage.PluginExists(ctx, *in.PluginName)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("el plugin %q no existe", *in.PluginName)
		}
	}
	return nil
}

// CreateAccount crea una cuenta validando institución y plugin.
func (s *AutomationService) CreateAccount(ctx context.Context, in AutomationAccountInput) (*models.AutomationAccount, error) {
	if err := validateAutomationAccountInput(in); err != nil {
		return nil, err
	}
	if err := s.validateAccountRefs(ctx, in); err != nil {
		return nil, err
	}
	frequency := in.ScheduleFrequency
	if frequency == "" {
		frequency = "daily"
	}
	account := &models.AutomationAccount{
		InstitutionID:     in.InstitutionID,
		Identifier:        strings.TrimSpace(in.Identifier),
		Label:             in.Label,
		PluginName:        in.PluginName,
		WebhookURL:        in.WebhookURL,
		ScheduleTime:      in.ScheduleTime,
		ScheduleEnabled:   in.ScheduleEnabled,
		ScheduleFrequency: frequency,
		ScheduleInterval:  in.ScheduleInterval,
		ScheduleDays:      in.ScheduleDays,
	}
	created, err := s.storage.CreateAccount(ctx, account, in.Credentials)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrAutomationAccountExists
		}
		return nil, err
	}
	return created, nil
}

// UpdateAccount actualiza una cuenta existente.
func (s *AutomationService) UpdateAccount(ctx context.Context, id int64, in AutomationAccountInput) (*models.AutomationAccount, error) {
	current, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validateAutomationAccountInput(in); err != nil {
		return nil, err
	}
	if err := s.validateAccountRefs(ctx, in); err != nil {
		return nil, err
	}
	frequency := in.ScheduleFrequency
	if frequency == "" {
		frequency = "daily"
	}
	current.InstitutionID = in.InstitutionID
	current.Identifier = strings.TrimSpace(in.Identifier)
	current.Label = in.Label
	current.PluginName = in.PluginName
	current.WebhookURL = in.WebhookURL
	current.ScheduleTime = in.ScheduleTime
	current.ScheduleEnabled = in.ScheduleEnabled
	current.ScheduleFrequency = frequency
	current.ScheduleInterval = in.ScheduleInterval
	current.ScheduleDays = in.ScheduleDays

	updated, err := s.storage.UpdateAccount(ctx, current, in.Credentials)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrAutomationAccountExists
		}
		return nil, err
	}
	return updated, nil
}

// DeleteAccount elimina una cuenta si no está vinculada a servicios activos.
func (s *AutomationService) DeleteAccount(ctx context.Context, id int64) error {
	if _, err := s.GetAccount(ctx, id); err != nil {
		return err
	}
	count, err := s.storage.CountServicesUsingAccount(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrAutomationAccountInUse
	}
	return s.storage.DeleteAccount(ctx, id)
}

// CredentialsForAccount devuelve las credenciales JSON (solo para el reveal
// autenticado con contraseña).
func (s *AutomationService) CredentialsForAccount(ctx context.Context, id int64) (string, error) {
	if _, err := s.GetAccount(ctx, id); err != nil {
		return "", err
	}
	return s.storage.CredentialsForAccount(ctx, id)
}

// UpdateAccountWebhook actualiza webhook_url y schedule de una cuenta.
func (s *AutomationService) UpdateAccountWebhook(ctx context.Context, id int64, in AutomationWebhookInput) (*models.AutomationAccount, error) {
	current, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.WebhookURL != nil {
		if err := validateWebhookURL(*in.WebhookURL); err != nil {
			return nil, err
		}
	}
	if err := validateAutomationSchedule(in.ScheduleTime, in.ScheduleFrequency, in.ScheduleInterval); err != nil {
		return nil, err
	}
	frequency := in.ScheduleFrequency
	if frequency == "" {
		frequency = "daily"
	}
	current.WebhookURL = in.WebhookURL
	current.ScheduleTime = in.ScheduleTime
	current.ScheduleEnabled = in.ScheduleEnabled
	current.ScheduleFrequency = frequency
	current.ScheduleInterval = in.ScheduleInterval
	current.ScheduleDays = in.ScheduleDays
	return s.storage.UpdateAccount(ctx, current, "")
}
