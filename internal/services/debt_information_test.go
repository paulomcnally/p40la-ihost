package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func setupDebtInformationTest(t *testing.T) (*DebtInformationService, *sql.DB) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO institutions (name) VALUES ('Banco Demo')`); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO debts (institution_id, identifier, description, total, principal,
		                   currency_id, installments_total, installment_amount,
		                   interest_rate, payment_day, start_date, status)
		VALUES (1, 'demo-account-001', 'Lote Demo 1', 100, 100, 1, 1, 100, 0, 5, '2026-09-01', 'activa')
	`); err != nil {
		t.Fatalf("insertar deuda: %v", err)
	}

	infoStorage := storage.NewDebtInformationStorage(database)
	debtStorage := storage.NewDebtStorage(database)
	return NewDebtInformationService(infoStorage, debtStorage), database
}

func validDebtInformationTemplate(name string) *models.DebtInformationTemplate {
	return &models.DebtInformationTemplate{
		Name:        name,
		Description: "Datos de prueba",
		Fields: []models.DebtInformationField{
			{Key: "monto", Label: "Monto", Type: models.DebtInformationFieldComputed, ValueTemplate: "{{cuota.monto_formateado}}"},
			{Key: "descripcion", Label: "Descripción", Type: models.DebtInformationFieldComputed, ValueTemplate: "{{deuda.descripcion}} - Cuota {{cuota.numero}}"},
			{Key: "banco", Label: "Banco", Type: models.DebtInformationFieldText, Required: true},
			{Key: "cuenta", Label: "# Cuenta", Type: models.DebtInformationFieldText, Required: true},
			{Key: "nombre", Label: "Nombre", Type: models.DebtInformationFieldText, Required: true},
		},
	}
}

func TestDebtInformationTemplateValidation(t *testing.T) {
	service, _ := setupDebtInformationTest(t)

	t.Run("plantilla válida", func(t *testing.T) {
		if err := service.prepareTemplate(validDebtInformationTemplate("Transferencia Demo")); err != nil {
			t.Fatalf("prepareTemplate: %v", err)
		}
	})

	t.Run("nombre obligatorio", func(t *testing.T) {
		input := validDebtInformationTemplate("   ")
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por nombre vacío")
		}
	})

	t.Run("al menos un campo", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields = nil
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por plantilla sin campos")
		}
	})

	t.Run("clave repetida", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[3].Key = "banco"
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por clave repetida")
		}
	})

	t.Run("clave inválida", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[2].Key = "Banco Demo"
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por clave inválida")
		}
	})

	t.Run("calculado sin expresión", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[0].ValueTemplate = ""
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por campo calculado sin expresión")
		}
	})

	t.Run("token desconocido", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[0].ValueTemplate = "{{cuota.total}}"
		err := service.prepareTemplate(input)
		var validation *DebtInformationValidationError
		if !errors.As(err, &validation) || validation.Code != "invalid_token" {
			t.Fatalf("se esperaba invalid_token, got %v", err)
		}
	})

	t.Run("token mal formado", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[0].ValueTemplate = "{{cuota.monto"
		err := service.prepareTemplate(input)
		var validation *DebtInformationValidationError
		if !errors.As(err, &validation) || validation.Code != "invalid_token" {
			t.Fatalf("se esperaba invalid_token, got %v", err)
		}
	})

	t.Run("select sin opciones", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[2] = models.DebtInformationField{Key: "metodo", Label: "Método", Type: models.DebtInformationFieldSelect}
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por select sin opciones")
		}
	})

	t.Run("select con default fuera de opciones", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[2] = models.DebtInformationField{
			Key: "metodo", Label: "Método", Type: models.DebtInformationFieldSelect,
			Options: []string{"Transferencia", "Efectivo"}, DefaultValue: "Tarjeta",
		}
		if err := service.prepareTemplate(input); err == nil {
			t.Fatal("se esperaba error por default fuera de opciones")
		}
	})

	t.Run("normaliza campos según el tipo", func(t *testing.T) {
		input := validDebtInformationTemplate("Transferencia Demo")
		input.Fields[2].ValueTemplate = "{{cuota.monto}}"
		input.Fields[0].Required = true
		if err := service.prepareTemplate(input); err != nil {
			t.Fatalf("prepareTemplate: %v", err)
		}
		if input.Fields[2].ValueTemplate != "" {
			t.Fatal("un campo de texto no debería conservar value_template")
		}
		if input.Fields[0].Required {
			t.Fatal("un campo calculado no debería ser requerido")
		}
	})
}

func TestDebtInformationTemplateCRUDAndUsage(t *testing.T) {
	service, database := setupDebtInformationTest(t)
	ctx := context.Background()

	created, err := service.CreateTemplate(ctx, validDebtInformationTemplate("Transferencia Demo"))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if created.ID == 0 || created.FieldCount != 5 || created.UsageCount != 0 {
		t.Fatalf("plantilla creada inesperada: %+v", created)
	}

	list, err := service.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID || list[0].FieldCount != 5 || list[0].UsageCount != 0 {
		t.Fatalf("lista inesperada: %+v", list)
	}
	if len(list[0].Fields) != 0 {
		t.Fatal("el listado no debería incluir los campos completos")
	}

	duplicate := validDebtInformationTemplate("transferencia demo")
	if _, err := service.CreateTemplate(ctx, duplicate); !errors.Is(err, ErrDebtInformationTemplateNameExists) {
		t.Fatalf("se esperaba ErrDebtInformationTemplateNameExists, got %v", err)
	}

	update := validDebtInformationTemplate("Transferencia Demo actualizada")
	update.ID = created.ID
	update.Fields = []models.DebtInformationField{
		{Key: "banco", Label: "Banco", Type: models.DebtInformationFieldText, Required: true},
		{Key: "cuenta", Label: "# Cuenta", Type: models.DebtInformationFieldText, Required: true},
	}
	updated, err := service.UpdateTemplate(ctx, update)
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Name != "Transferencia Demo actualizada" || updated.FieldCount != 2 {
		t.Fatalf("plantilla actualizada inesperada: %+v", updated)
	}

	info, err := service.SaveForDebt(ctx, 1, created.ID, map[string]any{
		"banco":  "Banco Demo",
		"cuenta": "demo-account-001",
		"nombre": "Titular Demo",
	})
	if err != nil {
		t.Fatalf("SaveForDebt: %v", err)
	}
	if info.TemplateID == nil || *info.TemplateID != created.ID {
		t.Fatalf("template_id inesperado: %+v", info.TemplateID)
	}
	if info.Values["banco"] != "Banco Demo" {
		t.Fatalf("valores inesperados: %+v", info.Values)
	}

	got, err := service.GetForDebt(ctx, 1)
	if err != nil {
		t.Fatalf("GetForDebt: %v", err)
	}
	if got.Template == nil || got.Template.ID != created.ID || got.Values["cuenta"] != "demo-account-001" {
		t.Fatalf("forma de pago inesperada: %+v", got)
	}

	list, err = service.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if list[0].UsageCount != 1 {
		t.Fatalf("usage_count esperado 1, got %d", list[0].UsageCount)
	}

	if err := service.DeleteTemplate(ctx, created.ID); !errors.Is(err, ErrDebtInformationTemplateInUse) {
		t.Fatalf("se esperaba ErrDebtInformationTemplateInUse, got %v", err)
	}

	if err := service.ClearForDebt(ctx, 1); err != nil {
		t.Fatalf("ClearForDebt: %v", err)
	}
	empty, err := service.GetForDebt(ctx, 1)
	if err != nil {
		t.Fatalf("GetForDebt después de limpiar: %v", err)
	}
	if empty.TemplateID != nil || empty.Template != nil || len(empty.Values) != 0 {
		t.Fatalf("se esperaba forma de pago vacía, got %+v", empty)
	}

	if err := service.DeleteTemplate(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	if deleted, err := service.GetTemplate(ctx, created.ID); err != nil || deleted != nil {
		t.Fatalf("la plantilla debería estar eliminada, got %+v, err %v", deleted, err)
	}

	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM debt_information_templates`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("quedaron plantillas: %d", count)
	}
}

func TestDebtInformationValuesValidation(t *testing.T) {
	fields := []models.DebtInformationField{
		{Key: "banco", Label: "Banco", Type: models.DebtInformationFieldText, Required: true},
		{Key: "nota", Label: "Nota", Type: models.DebtInformationFieldTextarea},
		{Key: "monto", Label: "Monto", Type: models.DebtInformationFieldNumber},
		{Key: "fecha", Label: "Fecha", Type: models.DebtInformationFieldDate},
		{Key: "metodo", Label: "Método", Type: models.DebtInformationFieldSelect, Options: []string{"Transferencia", "Efectivo"}},
		{Key: "acepta", Label: "Acepta", Type: models.DebtInformationFieldCheckbox, Required: true},
		{Key: "monto_calculado", Label: "Monto calculado", Type: models.DebtInformationFieldComputed, ValueTemplate: "{{cuota.monto_formateado}}"},
	}

	t.Run("requerido faltante", func(t *testing.T) {
		if _, err := validateDebtInformationValues(fields, map[string]any{}); err == nil {
			t.Fatal("se esperaba error por campo requerido")
		}
	})

	t.Run("valores válidos", func(t *testing.T) {
		values, err := validateDebtInformationValues(fields, map[string]any{
			"banco":  "  Banco Demo  ",
			"nota":   "Línea 1\nLínea 2",
			"monto":  float64(100),
			"fecha":  "2026-12-10",
			"metodo": "Transferencia",
			"acepta": true,
			"extra":  "ignorado",
		})
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if values["banco"] != "Banco Demo" {
			t.Fatalf("se esperaba texto recortado, got %#v", values["banco"])
		}
		if values["monto"] != float64(100) || values["acepta"] != true {
			t.Fatalf("valores inesperados: %#v", values)
		}
		if _, exists := values["extra"]; exists {
			t.Fatal("las claves desconocidas deberían ignorarse")
		}
		if _, exists := values["monto_calculado"]; exists {
			t.Fatal("los campos calculados no deberían persistirse")
		}
	})

	t.Run("número inválido", func(t *testing.T) {
		if _, err := validateDebtInformationValues(fields, map[string]any{
			"banco": "Banco Demo", "monto": "no-es-numero", "acepta": true,
		}); err == nil {
			t.Fatal("se esperaba error de número")
		}
	})

	t.Run("fecha inválida", func(t *testing.T) {
		if _, err := validateDebtInformationValues(fields, map[string]any{
			"banco": "Banco Demo", "fecha": "2026-02-31", "acepta": true,
		}); err == nil {
			t.Fatal("se esperaba error de fecha")
		}
	})

	t.Run("opción inválida", func(t *testing.T) {
		if _, err := validateDebtInformationValues(fields, map[string]any{
			"banco": "Banco Demo", "metodo": "Tarjeta", "acepta": true,
		}); err == nil {
			t.Fatal("se esperaba error de opción")
		}
	})

	t.Run("checkbox requerido en false", func(t *testing.T) {
		if _, err := validateDebtInformationValues(fields, map[string]any{
			"banco": "Banco Demo", "acepta": false,
		}); err == nil {
			t.Fatal("se esperaba error de checkbox requerido")
		}
	})

	t.Run("defaults aplicados", func(t *testing.T) {
		withDefaults := []models.DebtInformationField{
			{Key: "metodo", Label: "Método", Type: models.DebtInformationFieldSelect, Options: []string{"Transferencia", "Efectivo"}, DefaultValue: "Transferencia"},
			{Key: "monto", Label: "Monto", Type: models.DebtInformationFieldNumber, DefaultValue: "100"},
			{Key: "acepta", Label: "Acepta", Type: models.DebtInformationFieldCheckbox},
		}
		values, err := validateDebtInformationValues(withDefaults, map[string]any{})
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if values["metodo"] != "Transferencia" || values["monto"] != float64(100) || values["acepta"] != false {
			t.Fatalf("defaults inesperados: %#v", values)
		}
	})
}

func TestDebtInformationTokens(t *testing.T) {
	service, _ := setupDebtInformationTest(t)
	tokens := service.Tokens()
	if len(tokens) != 15 {
		t.Fatalf("tokens esperados 15, got %d", len(tokens))
	}
	if err := validateDebtInformationValueTemplate("{{cuota.monto_formateado}} - {{deuda.descripcion}}"); err != nil {
		t.Fatalf("template válido: %v", err)
	}
	if err := validateDebtInformationValueTemplate("{{ cuota.numero }}"); err != nil {
		t.Fatalf("template con espacios: %v", err)
	}
	if err := validateDebtInformationValueTemplate("{{cuota.total}}"); err == nil {
		t.Fatal("se esperaba error por token desconocido")
	}
	if err := validateDebtInformationValueTemplate("{{cuota.monto"); err == nil {
		t.Fatal("se esperaba error por token mal formado")
	}
}

func TestDebtInformationDebtNotFound(t *testing.T) {
	service, _ := setupDebtInformationTest(t)
	if _, err := service.GetForDebt(context.Background(), 999); !errors.Is(err, ErrDebtInformationDebtNotFound) {
		t.Fatalf("se esperaba ErrDebtInformationDebtNotFound, got %v", err)
	}
}

func TestDebtInformationSoftDeleteCleansPaymentInformation(t *testing.T) {
	service, database := setupDebtInformationTest(t)
	ctx := context.Background()

	created, err := service.CreateTemplate(ctx, validDebtInformationTemplate("Transferencia Demo"))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if _, err := service.SaveForDebt(ctx, 1, created.ID, map[string]any{
		"banco": "Banco Demo", "cuenta": "demo-account-001", "nombre": "Titular Demo",
	}); err != nil {
		t.Fatalf("SaveForDebt: %v", err)
	}

	if err := storage.NewDebtStorage(database).SoftDelete(ctx, 1); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM debt_payment_information`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("la forma de pago debería limpiarse al borrar la deuda, count=%d", count)
	}

	list, err := service.ListTemplates(ctx)
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(list) != 1 || list[0].UsageCount != 0 {
		t.Fatalf("usage_count debería quedar en 0: %+v", list)
	}
	if err := service.DeleteTemplate(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTemplate después del borrado lógico: %v", err)
	}
}
