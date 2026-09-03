package services

import (
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestBuildPensionRecordsCreatedEmail(t *testing.T) {
	salaryPayments := []models.SalaryPayment{{Employer: "Empresa XYZ", Amount: 15000, Currency: "NIO"}}
	records := []models.SupportRecord{{ChildName: "Juan Pérez", CategoryName: "Colegio", Amount: 1500, Currency: "NIO"}}

	title, content := buildPensionRecordsCreatedEmail(salaryPayments, records, 2026, 8)
	if !strings.Contains(title, "Agosto 2026") {
		t.Fatalf("título debería contener el período: %s", title)
	}
	if !strings.Contains(content, "Empresa XYZ") || !strings.Contains(content, "Juan Pérez") {
		t.Fatalf("contenido debería listar salarios y registros: %s", content)
	}
}

func TestBuildPensionRecordPaidEmail(t *testing.T) {
	paidAt := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	method := "bank_transfer"
	ref := "REF-1"
	record := &models.SupportRecord{
		ChildName: "Juan Pérez", CategoryName: "Colegio",
		Amount: 1500, Currency: "NIO", Year: 2026, Month: 8,
		PaidAt: &paidAt, PaymentMethod: &method, PaymentReference: &ref,
	}
	title, content := buildPensionRecordPaidEmail(record)
	if !strings.Contains(title, "Pago registrado") {
		t.Fatalf("título incorrecto: %s", title)
	}
	if !strings.Contains(content, "Juan Pérez") || !strings.Contains(content, "REF-1") {
		t.Fatalf("contenido incorrecto: %s", content)
	}
}

func TestBuildPensionSalaryReceivedEmail(t *testing.T) {
	receivedAt := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	received := 14500.0
	payment := &models.SalaryPayment{
		Employer: "Empresa XYZ", Amount: 15000, Currency: "NIO",
		Year: 2026, Month: 8, ReceivedAt: &receivedAt, ReceivedAmount: &received,
	}
	title, content := buildPensionSalaryReceivedEmail(payment)
	if !strings.Contains(title, "Salario recibido") {
		t.Fatalf("título incorrecto: %s", title)
	}
	if !strings.Contains(content, "Empresa XYZ") || !strings.Contains(content, "14500.00") {
		t.Fatalf("contenido incorrecto: %s", content)
	}
}

func TestBuildPensionRecordRejectedEmail(t *testing.T) {
	record := &models.SupportRecord{
		ChildName: "Juan Pérez", CategoryName: "Colegio",
		Amount: 1500, Currency: "NIO", Year: 2026, Month: 8,
	}
	title, content := buildPensionRecordRejectedEmail(record, "Comprobante inválido")
	if !strings.Contains(title, "Registro rechazado") {
		t.Fatalf("título incorrecto: %s", title)
	}
	if !strings.Contains(content, "Comprobante inválido") {
		t.Fatalf("el motivo debería estar en el contenido: %s", content)
	}
}

func TestBuildPensionMonthClosingEmail(t *testing.T) {
	records := []models.SupportRecord{
		{Status: "paid", Amount: 1500, Currency: "NIO", ChildName: "Juan", CategoryName: "Colegio"},
		{Status: "pending", Amount: 900, Currency: "NIO", ChildName: "Ana", CategoryName: "Transporte"},
	}
	salaryPayments := []models.SalaryPayment{{Employer: "Empresa XYZ", Amount: 15000, Currency: "NIO", Status: "received"}}

	title, content := buildPensionMonthClosingEmail(records, salaryPayments, 2026, 8)
	if !strings.Contains(title, "Cierre de mes") {
		t.Fatalf("título incorrecto: %s", title)
	}
	if !strings.Contains(content, "Pagados:") || !strings.Contains(content, "Pendientes:") || !strings.Contains(content, "Empresa XYZ") {
		t.Fatalf("el resumen debería incluir pagos, pendientes y salarios: %s", content)
	}
}
