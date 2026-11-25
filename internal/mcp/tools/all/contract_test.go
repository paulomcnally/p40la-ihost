// Package all testea el contrato del catálogo completo de tools MCP.
package all

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/registry"
)

var updateContracts = flag.Bool("update-contracts", false, "actualiza docs/mcp/tools/contracts.json")

type toolContract struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	InputSchema  any                  `json:"input_schema"`
	OutputSchema any                  `json:"output_schema,omitempty"`
	Annotations  *mcp.ToolAnnotations `json:"annotations,omitempty"`
}

type contractFile struct {
	Version     int            `json:"version"`
	GeneratedBy string         `json:"generated_by"`
	Tools       []toolContract `json:"tools"`
}

// contractAPI satisface api.API para registrar las tools sin llamar al backend.
type contractAPI struct{}

// ReadJSON satisface api.ReadAPI; el test de contrato solo lista tools.
func (contractAPI) ReadJSON(context.Context, string, any) error { return nil }

func (contractAPI) GetService(_ context.Context, id int64) (*api.Service, error) {
	return &api.Service{ID: id, Name: "Servicio Demo"}, nil
}

func (contractAPI) ListBills(context.Context, int64) ([]api.Bill, error) { return nil, nil }

func (contractAPI) GenerateBills(context.Context, int64, api.GenerateRequest) (*api.GenerateResult, error) {
	return &api.GenerateResult{}, nil
}

func (contractAPI) PayBills(context.Context, int64, api.PayRequest) (*api.PayResult, error) {
	return &api.PayResult{}, nil
}

func (contractAPI) ListHomes(context.Context) ([]api.Home, error) { return nil, nil }

func (contractAPI) CreateHome(context.Context, api.HomeCreateRequest) (*api.Home, error) {
	return &api.Home{}, nil
}

func (contractAPI) UpdateHome(context.Context, int64, api.HomeUpdateRequest) (*api.Home, error) {
	return &api.Home{}, nil
}

func (contractAPI) ListInstitutions(context.Context) ([]api.Institution, error) { return nil, nil }

func (contractAPI) CreateInstitution(context.Context, api.InstitutionCreateRequest) (*api.Institution, error) {
	return &api.Institution{}, nil
}

func (contractAPI) ListBanks(context.Context) ([]api.Bank, error) { return nil, nil }

func (contractAPI) CreateBank(context.Context, api.BankCreateRequest) (*api.Bank, error) {
	return &api.Bank{}, nil
}

func (contractAPI) ListCurrencies(context.Context) ([]api.Currency, error) { return nil, nil }

func (contractAPI) ListInstitutionCategories(context.Context) ([]api.InstitutionCategory, error) {
	return nil, nil
}

func (contractAPI) ListServices(context.Context, *int64) ([]api.Service, error) { return nil, nil }

func (contractAPI) CreateService(context.Context, api.ServiceCreateRequest) (*api.Service, error) {
	return &api.Service{}, nil
}

func (contractAPI) ListBankRefs(context.Context) ([]api.BankRef, error) { return nil, nil }

func (contractAPI) ListCurrencyRefs(context.Context) ([]api.CurrencyRef, error) { return nil, nil }

func (contractAPI) ListBankAccounts(context.Context, int64) ([]api.BankAccount, error) {
	return nil, nil
}

func (contractAPI) GetBankAccount(context.Context, int64) (*api.BankAccount, error) {
	return &api.BankAccount{}, nil
}

func (contractAPI) CreateBankAccount(context.Context, int64, api.BankAccountInput) (*api.BankAccount, error) {
	return &api.BankAccount{}, nil
}

func (contractAPI) UpdateBankAccount(context.Context, int64, api.BankAccountInput) (*api.BankAccount, error) {
	return &api.BankAccount{}, nil
}

func (contractAPI) ListCreditCards(context.Context, int64) ([]api.CreditCard, error) {
	return nil, nil
}

func (contractAPI) GetCreditCard(context.Context, int64) (*api.CreditCard, error) {
	return &api.CreditCard{}, nil
}

func (contractAPI) CreateCreditCard(context.Context, int64, api.CreditCardInput) (*api.CreditCard, error) {
	return &api.CreditCard{}, nil
}

func (contractAPI) UpdateCreditCard(context.Context, int64, api.CreditCardInput) (*api.CreditCard, error) {
	return &api.CreditCard{}, nil
}

func (contractAPI) ListAutos(context.Context) ([]api.Auto, error) { return nil, nil }

func (contractAPI) GetAuto(context.Context, int64) (*api.Auto, error) { return &api.Auto{}, nil }

func (contractAPI) CreateAuto(context.Context, api.AutoRequest) (*api.Auto, error) {
	return &api.Auto{}, nil
}

func (contractAPI) UpdateAuto(context.Context, int64, api.AutoRequest) (*api.Auto, error) {
	return &api.Auto{}, nil
}

func (contractAPI) ListAvailableServices(context.Context, int64) ([]api.AvailableService, error) {
	return nil, nil
}

func (contractAPI) CreateMaintenance(context.Context, int64, api.MaintenanceRequest) (*api.Maintenance, error) {
	return &api.Maintenance{}, nil
}

func (contractAPI) GetMaintenance(context.Context, int64, int64) (*api.Maintenance, error) {
	return &api.Maintenance{}, nil
}

func (contractAPI) CreateMaintenanceItem(context.Context, int64, int64, api.MaintenanceItemRequest) (*api.MaintenanceItem, error) {
	return &api.MaintenanceItem{}, nil
}

func (contractAPI) CreateInspection(context.Context, int64, api.InspectionRequest) (*api.Inspection, error) {
	return &api.Inspection{}, nil
}

func (contractAPI) ListInspections(context.Context, int64) (*api.InspectionList, error) {
	return &api.InspectionList{}, nil
}

func (contractAPI) ListInsurances(context.Context, int64) ([]api.InsuranceDetail, error) {
	return nil, nil
}

func (contractAPI) CreateInsurance(context.Context, int64, api.InsuranceRequest) (*api.Insurance, error) {
	return &api.Insurance{}, nil
}

func (contractAPI) ListHomeMaintenances(context.Context, int64, api.HomeMaintenanceFilter) (*api.HomeMaintenanceList, error) {
	return &api.HomeMaintenanceList{}, nil
}

func (contractAPI) GetHomeMaintenance(context.Context, int64, int64) (*api.HomeMaintenance, error) {
	return &api.HomeMaintenance{}, nil
}

func (contractAPI) CreateHomeMaintenance(context.Context, int64, api.HomeMaintenanceRequest) (*api.HomeMaintenance, error) {
	return &api.HomeMaintenance{}, nil
}

func (contractAPI) UpdateHomeMaintenance(context.Context, int64, int64, api.HomeMaintenanceUpdateRequest) (*api.HomeMaintenance, error) {
	return &api.HomeMaintenance{}, nil
}

func (contractAPI) CreateHomeMaintenanceItem(context.Context, int64, int64, api.HomeMaintenanceItemRequest) (*api.HomeMaintenanceItem, error) {
	return &api.HomeMaintenanceItem{}, nil
}

func (contractAPI) UpdateHomeMaintenanceItem(context.Context, int64, int64, int64, api.HomeMaintenanceItemRequest) (*api.HomeMaintenanceItem, error) {
	return &api.HomeMaintenanceItem{}, nil
}

func (contractAPI) DeleteHomeMaintenanceItem(context.Context, int64, int64, int64) error {
	return nil
}

func (contractAPI) MoveHomeMaintenanceItems(context.Context, int64, int64, api.HomeMaintenanceMoveRequest) (*api.HomeMaintenanceMoveResult, error) {
	return &api.HomeMaintenanceMoveResult{}, nil
}

// TestToolContracts congela el contrato visible de tools/list para todo el
// catálogo: nombres, descripciones, esquemas y anotaciones. Un cambio
// intencional se acepta con:
//
//	go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts
func TestToolContracts(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-contract", Version: "test"}, nil)
	reg := &registry.Registry{}
	Register(reg, contractAPI{}, config.Config{Timezone: "UTC", Location: time.UTC})
	reg.Mount(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("conectar server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "p40la-mcp-contract-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("conectar cliente: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}

	contracts := make([]toolContract, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		contract := toolContract{
			Name:        tool.Name,
			Description: tool.Description,
			Annotations: tool.Annotations,
		}
		if tool.InputSchema != nil {
			contract.InputSchema = tool.InputSchema
		}
		if tool.OutputSchema != nil {
			contract.OutputSchema = tool.OutputSchema
		}
		contracts = append(contracts, contract)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].Name < contracts[j].Name })

	got := contractFile{
		Version:     1,
		GeneratedBy: "go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts",
		Tools:       contracts,
	}
	path := filepath.Join("..", "..", "..", "..", "docs", "mcp", "tools", "contracts.json")

	if *updateContracts {
		encoded, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatalf("codificar contrato: %v", err)
		}
		encoded = append(encoded, '\n')
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatalf("escribir contrato: %v", err)
		}
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer contrato %s: %v (generalo con -update-contracts)", path, err)
	}
	var want contractFile
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("decodificar contrato: %v", err)
	}

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("codificar contrato generado: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("codificar contrato esperado: %v", err)
	}
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("el contrato de tools/list cambió.\n"+
			"Revisá el diff y, si es intencional, regenerá con:\n"+
			"  go test ./internal/mcp/tools/all -run TestToolContracts -update-contracts\n\n"+
			"--- generado ---\n%s\n--- esperado ---\n%s", gotJSON, wantJSON)
	}
}
