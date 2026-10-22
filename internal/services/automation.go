package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
