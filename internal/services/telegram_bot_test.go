package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func testTelegramSettings(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestFormatServiciosPendientes(t *testing.T) {
	format := DefaultCurrencyFormat()
	// Fecha fija de referencia: 2026-09-19 (zona UTC, ADR-004).
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	t.Run("sin pendientes", func(t *testing.T) {
		got := formatServiciosPendientes(nil, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 1 || got[0] != "✅ No hay facturas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("un mensaje por servicio + totales, ordenado por monto", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Year: 2026, Month: 8, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Year: 2026, Month: 9, Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{ServiceID: 2, ServiceName: "ENATREL", HomeName: "Casa B", Year: 2026, Month: 8, Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		// 2 servicios + 1 totales.
		if len(got) != 3 {
			t.Fatalf("esperados 3 mensajes, got %d: %q", len(got), got)
		}
		// ENATREL (500) antes que Claro (150).
		enatrelPos := indexOf(got[0], "ENATREL")
		claroPos := indexOf(got[1], "Claro")
		if enatrelPos == -1 || claroPos == -1 {
			t.Fatalf("mensajes por servicio incorrectos:\n%s\n%s", got[0], got[1])
		}
		if !contains(got[1], "*Facturas*: 2") || !contains(got[1], "C$150.00") {
			t.Errorf("conteo/monto de Claro incorrecto:\n%s", got[1])
		}
		if !contains(got[0], "C$500.00") {
			t.Errorf("monto de ENATREL incorrecto:\n%s", got[0])
		}
		// Último mensaje: totales por moneda.
		if !contains(got[2], "Totales") || !contains(got[2], "NIO: C$650.00") {
			t.Errorf("mensaje de totales incorrecto:\n%s", got[2])
		}
	})

	t.Run("itemiza cada factura con semáforo y fecha legible", func(t *testing.T) {
		due1 := "2026-09-07" // vencida hace 12 días → 🔴
		due2 := "2026-09-23" // vence en 4 días → 🟡
		due3 := "2026-09-30" // vence en 11 días → 🟢 (último día del mes)
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Year: 2026, Month: 8, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due1},
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Year: 2026, Month: 9, Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due2},
			{ServiceID: 1, ServiceName: "Claro", HomeName: "Casa A", Year: 2026, Month: 10, Amount: 30, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due3},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		// 1 servicio + 1 totales.
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		// Encabezado con casa en línea.
		if !contains(got[0], "⚡ *Claro* — Casa A") {
			t.Errorf("mensaje sin encabezado del servicio:\n%s", got[0])
		}
		// Fechas legibles y semáforos por regla SPEC-081 REQ-008.
		if !contains(got[0], "🔴 Vencida hace 12 días") || !contains(got[0], "📅 07 sep 2026 — C$100.00") {
			t.Errorf("mensaje sin vencida resaltada:\n%s", got[0])
		}
		if !contains(got[0], "🟡 Vence en 4 días") || !contains(got[0], "📅 23 sep 2026 — C$50.00") {
			t.Errorf("mensaje sin próxima amarilla:\n%s", got[0])
		}
		if !contains(got[0], "🟢 Vence en 11 días") || !contains(got[0], "📅 30 sep 2026 — C$30.00") {
			t.Errorf("mensaje sin lejana verde:\n%s", got[0])
		}
		// Resumen con separador ASCII al final.
		if !contains(got[0], "--------------------") || !contains(got[0], "*Facturas*: 3") || !contains(got[0], "*Pendiente*: C$180.00") {
			t.Errorf("mensaje sin resumen final:\n%s", got[0])
		}
	})

	t.Run("filtra facturas de meses futuros", func(t *testing.T) {
		dueVencida := "2026-09-01"  // vencida → se mantiene
		dueActual := "2026-09-30"   // último día del mes → se mantiene
		dueFutura := "2026-10-01"   // mes siguiente → se excluye
		dueFutura2 := "2026-11-15"  // meses siguientes → se excluye
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueVencida},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueActual},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 10, Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueFutura},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 11, Amount: 700, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueFutura2},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		// Las futuras no aparecen ni en detalle ni en resumen/totales.
		if strings.Contains(got[0], "500.00") || strings.Contains(got[0], "700.00") {
			t.Errorf("factura futura aparece en el detalle:\n%s", got[0])
		}
		if !contains(got[0], "*Facturas*: 2") || !contains(got[0], "*Pendiente*: C$150.00") {
			t.Errorf("conteo/total debería excluir futuras:\n%s", got[0])
		}
		if !contains(got[1], "NIO: C$150.00") {
			t.Errorf("totales debería excluir futuras:\n%s", got[1])
		}
	})

t.Run("sin facturas del periodo actual muestra vacío", func(t *testing.T) {
		dueFutura := "2026-10-05"
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 10, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueFutura},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 1 || got[0] != "✅ No hay facturas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("showMonths=2 incluye el próximo mes pero no el siguiente", func(t *testing.T) {
		dueVencida := "2026-07-01" // vencida hace mucho → siempre
		dueProximo := "2026-10-20" // mes siguiente → con N=2 se muestra
		dueSiguiente := "2026-11-15" // dos meses después → con N=2 se excluye
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 7, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueVencida},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 10, Amount: 200, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueProximo},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 11, Amount: 400, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueSiguiente},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, 2)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "C$100.00") || !contains(got[0], "C$200.00") {
			t.Errorf("vencida o próxima mes ausente:\n%s", got[0])
		}
		if strings.Contains(got[0], "C$400.00") {
			t.Errorf("factura de +2 meses no debería aparecer con N=2:\n%s", got[0])
		}
		if !contains(got[0], "*Facturas*: 2") || !contains(got[0], "*Pendiente*: C$300.00") {
			t.Errorf("conteo/total incorrecto:\n%s", got[0])
		}
	})

	t.Run("showMonths=3 incluye dos meses futuros", func(t *testing.T) {
		dueProximo := "2026-10-20"
		dueSiguiente := "2026-11-15"
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 10, Amount: 200, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueProximo},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 11, Amount: 300, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueSiguiente},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, 3)
		if !contains(got[0], "C$200.00") || !contains(got[0], "C$300.00") {
			t.Errorf("meses futuros ausentes con N=3:\n%s", got[0])
		}
		if !contains(got[0], "*Facturas*: 2") {
			t.Errorf("conteo incorrecto con N=3:\n%s", got[0])
		}
	})

	t.Run("vencidas antiguas siempre se muestran aunque N=1", func(t *testing.T) {
		dueVencida := "2025-01-15" // vencida hace más de un año
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2025, Month: 1, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueVencida},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 2 || !contains(got[0], "C$100.00") {
			t.Errorf("vencida antigua debería mostrarse siempre:\n%q", got)
		}
	})

	t.Run("separador configurable", func(t *testing.T) {
		due := "2026-09-25"
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due},
		}
		// 30 guiones.
		got30 := formatServiciosPendientes(pending, format, now, 30, DefaultTelegramBotShowMonths)
		if !contains(got30[0], "  ------------------------------") {
			t.Errorf("separador de 30 guiones incorrecto:\n%s", got30[0])
		}
		// 0 = sin separador.
		got0 := formatServiciosPendientes(pending, format, now, 0, DefaultTelegramBotShowMonths)
		if strings.Contains(got0[0], "----") || !contains(got0[0], "*Facturas*: 1") {
			t.Errorf("separador 0 debería omitirse:\n%s", got0[0])
		}
	})

	t.Run("vence hoy es rojo", func(t *testing.T) {
		due := "2026-09-19"
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if !contains(got[0], "🔴 Vence hoy") {
			t.Errorf("vence hoy debería ser rojo:\n%s", got[0])
		}
	})

	t.Run("singular en días", func(t *testing.T) {
		due := "2026-09-18"
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if !contains(got[0], "🔴 Vencida hace 1 día") {
			t.Errorf("singular incorrecto:\n%s", got[0])
		}
	})

	t.Run("factura sin due_date usa periodo con semáforo verde", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "Luz", HomeName: "Casa A", Year: 2026, Month: 9, Amount: 620, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "🟢 Sin fecha") || !contains(got[0], "📅 sep 2026 — C$620.00") {
			t.Errorf("factura sin due_date debería mostrar periodo con 🟢:\n%s", got[0])
		}
	})

	t.Run("ordena por urgencia: vencidas primero, sin fecha al final", func(t *testing.T) {
		dueLejana := "2026-09-30" // último día del mes → 🟢
		dueVencida := "2026-09-01" // vencida hace 18 días → 🔴
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 9, Amount: 10, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueLejana},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 8, Amount: 20, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Year: 2026, Month: 7, Amount: 30, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &dueVencida},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		msg := got[0]
		vencidaPos := indexOf(msg, "Vencida hace 18 días")
		lejanaPos := indexOf(msg, "Vence en 11 días")
		sinFechaPos := indexOf(msg, "Sin fecha")
		if vencidaPos == -1 || lejanaPos == -1 || sinFechaPos == -1 {
			t.Fatalf("no se encontraron los estados:\n%s", msg)
		}
		if !(vencidaPos < lejanaPos && lejanaPos < sinFechaPos) {
			t.Errorf("orden incorrecto (esperado vencida < lejana < sin fecha):\n%s", msg)
		}
	})

	t.Run("particiona servicios con más de 25 facturas", func(t *testing.T) {
		var pending []appmodels.PendingBillDetail
		for i := 1; i <= 30; i++ {
			due := "2026-09-20"
			pending = append(pending, appmodels.PendingBillDetail{
				ServiceID: 1, ServiceName: "Internet", HomeName: "Casa A", Year: 2026, Month: 9,
				Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO", DueDate: &due,
			})
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		// 30 facturas → 2 bloques (25+5) + 1 totales.
		if len(got) != 3 {
			t.Fatalf("esperados 3 mensajes (2 bloques + totales), got %d", len(got))
		}
		// Encabezado solo en el primer bloque.
		if !contains(got[0], "⚡ *Internet*") || contains(got[1], "⚡") {
			t.Errorf("encabezado repetido en bloques:\n%s\n---\n%s", got[0], got[1])
		}
		// Resumen solo al final del último bloque.
		if contains(got[0], "Pendiente:") || !contains(got[1], "*Facturas*: 30") || !contains(got[1], "*Pendiente*: C$3,000.00") {
			t.Errorf("resumen mal ubicado:\n%s\n---\n%s", got[0], got[1])
		}
		// Primer bloque 25 facturas, segundo 5.
		if strings.Count(got[0], "📅") != 25 || strings.Count(got[1], "📅") != 5 {
			t.Errorf("cantidad de facturas por bloque incorrecta: %d/%d",
				strings.Count(got[0], "📅"), strings.Count(got[1], "📅"))
		}
		// Ningún mensaje de servicio supera el límite de Telegram.
		for i, m := range got[:2] {
			if len(m) > 4096 {
				t.Errorf("bloque %d supera 4096 chars: %d", i, len(m))
			}
		}
	})

	t.Run("formato de moneda personalizado", func(t *testing.T) {
		format := CurrencyFormat{ThousandsSeparator: ",", DecimalSeparator: ".", DecimalDigits: 2}
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 1250.5, CurrencySymbol: "$", CurrencyCode: "USD"},
		}
		got := formatServiciosPendientes(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "$1,250.50") {
			t.Errorf("formato personalizado incorrecto:\n%s", got[0])
		}
		if !contains(got[1], "USD: $1,250.50") {
			t.Errorf("totales con formato personalizado incorrecto:\n%s", got[1])
		}
	})

	t.Run("totales multi-moneda", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 1000, CurrencySymbol: "$", CurrencyCode: "USD"},
			{ServiceID: 2, ServiceName: "S2", HomeName: "H", Amount: 5000, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatServiciosTotales(pending, format)
		if !contains(got, "USD: $1,000.00") || !contains(got, "NIO: C$5,000.00") {
			t.Errorf("totales multi-moneda incorrectos:\n%s", got)
		}
	})

	t.Run("totales con código vacío usa símbolo", func(t *testing.T) {
		pending := []appmodels.PendingBillDetail{
			{ServiceID: 1, ServiceName: "S1", HomeName: "H", Amount: 100, CurrencySymbol: "C$"},
		}
		got := formatServiciosTotales(pending, format)
		if !contains(got, "C$: C$100.00") {
			t.Errorf("totales con fallback a símbolo incorrectos:\n%s", got)
		}
	})
}

func TestFormatDeudasPendientes(t *testing.T) {
	format := DefaultCurrencyFormat()
	// Fecha fija de referencia: 2026-09-19 (zona UTC, ADR-004).
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	sepLen := DefaultTelegramBotSeparatorLength
	showMonths := DefaultTelegramBotShowMonths

	t.Run("sin pendientes", func(t *testing.T) {
		got := formatDeudasPendientes(nil, format, now, sepLen, showMonths)
		if len(got) != 1 || got[0] != "✅ No hay deudas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("itemiza cuotas con semáforo, fecha legible, espaciado y negritas", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "Préstamo LAFISE", InstitutionName: "Banco LAFISE", DueDate: "2026-09-07", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "Préstamo LAFISE", InstitutionName: "Banco LAFISE", DueDate: "2026-09-23", Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 2, DebtDescription: "Tarjeta BAC", InstitutionName: "BAC Credomatic", DueDate: "2026-09-10", Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format, now, sepLen, showMonths)
		// 2 deudas + 1 totales.
		if len(got) != 3 {
			t.Fatalf("esperados 3 mensajes, got %d: %q", len(got), got)
		}
		// BAC (500) primero: encabezado + cuota con semáforo + resumen.
		if !contains(got[0], "💳 *Tarjeta BAC*") || !contains(got[0], "*Institución*: BAC Credomatic") {
			t.Errorf("mensaje de BAC sin encabezado:\n%s", got[0])
		}
		if !contains(got[0], "🔴 Vencida hace 9 días") || !contains(got[0], "📅 10 sep 2026 — C$500.00") {
			t.Errorf("mensaje de BAC sin semáforo/fecha legible:\n%s", got[0])
		}
		if !contains(got[0], "*Cuotas*: 1") || !contains(got[0], "*Pendiente*: C$500.00") {
			t.Errorf("mensaje de BAC sin resumen en negrita:\n%s", got[0])
		}
		// LAFISE: cuotas con semáforo y fecha legible, línea en blanco entre ellas.
		if !contains(got[1], "🔴 Vencida hace 12 días") || !contains(got[1], "📅 07 sep 2026 — C$100.00") {
			t.Errorf("mensaje de LAFISE sin vencida:\n%s", got[1])
		}
		if !contains(got[1], "🟡 Vence en 4 días") || !contains(got[1], "📅 23 sep 2026 — C$50.00") {
			t.Errorf("mensaje de LAFISE sin próxima amarilla:\n%s", got[1])
		}
		// Espaciado: cada cuota termina en línea en blanco.
		if !strings.Contains(got[1], "📅 07 sep 2026 — C$100.00\n\n  🟡") {
			t.Errorf("sin línea en blanco entre cuotas:\n%s", got[1])
		}
		if !contains(got[1], "*Cuotas*: 2") || !contains(got[1], "*Pendiente*: C$150.00") {
			t.Errorf("mensaje de LAFISE sin resumen:\n%s", got[1])
		}
		if !contains(got[2], "Totales") || !contains(got[2], "NIO: C$650.00") {
			t.Errorf("mensaje de totales incorrecto:\n%s", got[2])
		}
	})

	t.Run("filtra cuotas de meses futuros", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-09-01", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-09-30", Amount: 50, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-10-01", Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2027-03-10", Amount: 700, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format, now, sepLen, showMonths)
		// 1 deuda + 1 totales.
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		// Futuras excluidas del detalle, conteo y totales.
		if strings.Contains(got[0], "500.00") || strings.Contains(got[0], "700.00") {
			t.Errorf("cuota futura aparece en el detalle:\n%s", got[0])
		}
		if !contains(got[0], "*Cuotas*: 2") || !contains(got[0], "*Pendiente*: C$150.00") {
			t.Errorf("conteo/total debería excluir futuras:\n%s", got[0])
		}
		if !contains(got[1], "NIO: C$150.00") {
			t.Errorf("totales debería excluir futuras:\n%s", got[1])
		}
	})

	t.Run("sin cuotas del periodo actual muestra vacío", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-10-05", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format, now, sepLen, showMonths)
		if len(got) != 1 || got[0] != "✅ No hay deudas pendientes." {
			t.Errorf("esperado único mensaje vacío, got %q", got)
		}
	})

	t.Run("showMonths=2 incluye el próximo mes pero no el siguiente", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-07-01", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-10-20", Amount: 200, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-11-15", Amount: 400, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format, now, sepLen, 2)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes, got %d: %q", len(got), got)
		}
		if !contains(got[0], "C$100.00") || !contains(got[0], "C$200.00") {
			t.Errorf("vencida o próximo mes ausente:\n%s", got[0])
		}
		if strings.Contains(got[0], "C$400.00") {
			t.Errorf("cuota de +2 meses no debería aparecer con N=2:\n%s", got[0])
		}
		if !contains(got[0], "*Cuotas*: 2") || !contains(got[0], "*Pendiente*: C$300.00") {
			t.Errorf("conteo/total incorrecto:\n%s", got[0])
		}
	})

	t.Run("separador configurable", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", DueDate: "2026-09-05", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasPendientes(pending, format, now, 30, showMonths)
		if !strings.Contains(got[0], strings.Repeat("-", 30)) {
			t.Errorf("separador de 30 guiones no aplicado:\n%s", got[0])
		}
		got0 := formatDeudasPendientes(pending, format, now, 0, showMonths)
		if strings.Contains(got0[0], strings.Repeat("-", 20)) {
			t.Errorf("separador 0 debería omitirse:\n%s", got0[0])
		}
	})

	t.Run("particiona deudas con más de 25 cuotas", func(t *testing.T) {
		var pending []appmodels.PendingDebtDetail
		for i := 1; i <= 60; i++ {
			pending = append(pending, appmodels.PendingDebtDetail{
				DebtID: 1, DebtDescription: "Hipoteca", InstitutionName: "Banco",
				DueDate: "2026-09-12", Amount: 620, CurrencySymbol: "C$", CurrencyCode: "NIO",
			})
		}
		got := formatDeudasPendientes(pending, format, now, sepLen, showMonths)
		// 60 cuotas → 3 bloques (25+25+10) + 1 totales.
		if len(got) != 4 {
			t.Fatalf("esperados 4 mensajes (3 bloques + totales), got %d", len(got))
		}
		// Encabezado solo en el primer bloque.
		if !contains(got[0], "💳 *Hipoteca*") || contains(got[1], "💳") {
			t.Errorf("encabezado repetido en bloques:\n%s\n---\n%s", got[0], got[1])
		}
		// Resumen solo al final del último bloque.
		if contains(got[0], "Pendiente:") || !contains(got[2], "*Cuotas*: 60") || !contains(got[2], "*Pendiente*: C$37,200.00") {
			t.Errorf("resumen mal ubicado:\n%s\n---\n%s", got[0], got[2])
		}
		// Primer y segundo bloque tienen 25 cuotas; el tercero 10.
		if strings.Count(got[0], "📅") != 25 || strings.Count(got[1], "📅") != 25 || strings.Count(got[2], "📅") != 10 {
			t.Errorf("cantidad de cuotas por bloque incorrecta: %d/%d/%d",
				strings.Count(got[0], "📅"), strings.Count(got[1], "📅"), strings.Count(got[2], "📅"))
		}
		// Ningún mensaje de deuda supera el límite de Telegram.
		for i, m := range got[:3] {
			if len(m) > 4096 {
				t.Errorf("bloque %d supera 4096 chars: %d", i, len(m))
			}
		}
	})

	t.Run("totales multi-moneda", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", Amount: 200, CurrencySymbol: "$", CurrencyCode: "USD"},
			{DebtID: 2, DebtDescription: "D2", InstitutionName: "I2", Amount: 300, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDeudasTotales(pending, format)
		if !contains(got, "USD: $200.00") || !contains(got, "NIO: C$300.00") {
			t.Errorf("totales multi-moneda incorrectos:\n%s", got)
		}
	})

	t.Run("totales con código vacío usa símbolo", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "D1", InstitutionName: "I1", Amount: 75, CurrencySymbol: "C$"},
		}
		got := formatDeudasTotales(pending, format)
		if !contains(got, "C$: C$75.00") {
			t.Errorf("totales con fallback a símbolo incorrectos:\n%s", got)
		}
	})
}

func TestTelegramBotServiceLifecycle(t *testing.T) {
	settings := testTelegramSettings(t)
	bills := storage.NewBillStorage(nil)
	svc := NewTelegramBotService(settings, bills, storage.NewDebtBillStorage(nil))

	// Sin config: Start no debe panicear y el supervisor queda inactivo.
	svc.Start()
	svc.Stop()

	// Notify sin polling activo no debe bloquearse.
	svc.NotifyConfigChanged()
}

func TestTelegramBotConfigSettings(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)

	cfg, err := settings.GetTelegramBotConfig(ctx)
	if err != nil {
		t.Fatalf("config inicial: %v", err)
	}
	if cfg.Enabled || cfg.Token != "" || len(cfg.ChatIDs) != 0 {
		t.Errorf("config inicial no vacía: %+v", cfg)
	}
	// Defaults de las settings nuevas (SPEC-084 REQ-014/016).
	if cfg.SeparatorLength != DefaultTelegramBotSeparatorLength || cfg.ShowMonths != DefaultTelegramBotShowMonths {
		t.Errorf("defaults incorrectos: %+v", cfg)
	}

	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("enabled: %v", err)
	}
	if err := settings.SetTelegramBotToken(ctx, "token-123"); err != nil {
		t.Fatalf("token: %v", err)
	}
	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111", "222"}); err != nil {
		t.Fatalf("chat_ids: %v", err)
	}
	if err := settings.SetTelegramBotSeparatorLength(ctx, 30); err != nil {
		t.Fatalf("separator_length: %v", err)
	}
	if err := settings.SetTelegramBotShowMonths(ctx, 3); err != nil {
		t.Fatalf("show_months: %v", err)
	}

	cfg, err = settings.GetTelegramBotConfig(ctx)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if !cfg.Enabled || cfg.Token != "token-123" || len(cfg.ChatIDs) != 2 {
		t.Errorf("config guardada incorrecta: %+v", cfg)
	}
	if cfg.SeparatorLength != 30 || cfg.ShowMonths != 3 {
		t.Errorf("settings nuevas guardadas incorrectas: %+v", cfg)
	}

	// Rangos inválidos: separador fuera de 0-100, meses fuera de 1-12.
	if err := settings.SetTelegramBotSeparatorLength(ctx, -1); err == nil {
		t.Error("separator_length negativo debería fallar")
	}
	if err := settings.SetTelegramBotSeparatorLength(ctx, 101); err == nil {
		t.Error("separator_length >100 debería fallar")
	}
	if err := settings.SetTelegramBotShowMonths(ctx, 0); err == nil {
		t.Error("show_months 0 debería fallar")
	}
	if err := settings.SetTelegramBotShowMonths(ctx, 13); err == nil {
		t.Error("show_months 13 debería fallar")
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if cfg.SeparatorLength != 30 || cfg.ShowMonths != 3 {
		t.Errorf("valores previos deberían mantenerse tras intentos inválidos: %+v", cfg)
	}

	// El token no debe sobrescribirse con vacío (setIfNonEmpty).
	if err := settings.SetTelegramBotToken(ctx, ""); err != nil {
		t.Fatalf("token vacío: %v", err)
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if cfg.Token != "token-123" {
		t.Errorf("token fue sobrescrito con vacío: %q", cfg.Token)
	}

	// Config pública: sin token, con flag configured y lista de chat_ids
	// (SPEC-089: la UI precarga la allowlist desde el API).
	pub, err := settings.GetTelegramBotConfigPublic(ctx)
	if err != nil {
		t.Fatalf("public: %v", err)
	}
	if !pub.Enabled || !pub.Configured {
		t.Errorf("public incorrecta: %+v", pub)
	}
	if pub.SeparatorLength != 30 || pub.ShowMonths != 3 {
		t.Errorf("public con settings nuevas incorrecta: %+v", pub)
	}
	if len(pub.ChatIDs) != 2 || pub.ChatIDs[0] != "111" || pub.ChatIDs[1] != "222" || pub.ChatIDsCount != 2 {
		t.Errorf("public con chat_ids incorrecta: %+v", pub)
	}

	// Apagar el bot conserva token y chat_ids para re-encenderlo (SPEC-089:
	// ya no existe el flujo de borrado total "Reconfigurar").
	if err := settings.SetTelegramBotEnabled(ctx, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if cfg.Enabled || cfg.Token != "token-123" || len(cfg.ChatIDs) != 2 {
		t.Errorf("config tras deshabilitar incorrecta (debe conservar credenciales): %+v", cfg)
	}
	if err := settings.SetTelegramBotEnabled(ctx, true); err != nil {
		t.Fatalf("re-enable: %v", err)
	}
	cfg, _ = settings.GetTelegramBotConfig(ctx)
	if !cfg.Enabled || cfg.Token != "token-123" || len(cfg.ChatIDs) != 2 {
		t.Errorf("config tras re-encender incorrecta: %+v", cfg)
	}
}

func TestIsAuthorized(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))

	// Sin allowlist: todos autorizados.
	if !svc.isAuthorized(ctx, 12345) {
		t.Error("sin allowlist debería autorizar")
	}

	if err := settings.SetTelegramBotChatIDs(ctx, []string{"111", "222"}); err != nil {
		t.Fatalf("chat_ids: %v", err)
	}
	if !svc.isAuthorized(ctx, 111) || !svc.isAuthorized(ctx, 222) {
		t.Error("chat en allowlist debería autorizar")
	}
	if svc.isAuthorized(ctx, 333) {
		t.Error("chat fuera de allowlist no debería autorizar")
	}
}

func TestCheckAuthorizedIgnoresNonMessage(t *testing.T) {
	ctx := context.Background()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))
	// update.Message == nil → no autorizado (no panic).
	ok := svc.checkAuthorized(ctx, nil, &tgmodels.Update{})
	if ok {
		t.Error("update sin message no debería autorizarse")
	}
}

// TestBotNewFailsInvalidToken verifica que un token inválido no tumba el
// supervisor: bot.New devuelve error y runBot retorna (CA-NF-001).
func TestBotNewFailsInvalidToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	settings := testTelegramSettings(t)
	svc := NewTelegramBotService(settings, storage.NewBillStorage(nil), storage.NewDebtBillStorage(nil))

	cfg := &appmodels.TelegramBotConfig{Enabled: true, Token: "token-invalido"}
	done := make(chan struct{})
	go func() {
		svc.runBot(ctx, cfg)
		close(done)
	}()
	select {
	case <-done:
		// runBot retornó sin panic (getMe falló).
	case <-time.After(1 * time.Second):
		t.Fatal("runBot no retornó con token inválido")
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// fakeTelegramAPI simula la Bot API de Telegram para testear el dispatcher
// real de go-telegram/bot (SPEC-080): getMe, setMyCommands y sendMessage.
func fakeTelegramAPI(t *testing.T, sent *[]string, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/botTESTTOKEN/getMe", func(w http.ResponseWriter, r *http.Request) {
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"id": 1, "is_bot": true, "first_name": "Test", "username": "testbot",
			},
		})
	})

	mux.HandleFunc("/botTESTTOKEN/setMyCommands", func(w http.ResponseWriter, r *http.Request) {
		writeTGJSON(t, w, map[string]any{"ok": true, "result": true})
	})

	mux.HandleFunc("/botTESTTOKEN/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("sendMessage: parse form: %v", err)
			writeTGJSON(t, w, map[string]any{"ok": false})
			return
		}
		mu.Lock()
		*sent = append(*sent, r.FormValue("text"))
		mu.Unlock()
		writeTGJSON(t, w, map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 1,
				"chat":       map[string]any{"id": 123},
				"text":       r.FormValue("text"),
			},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeTGJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("escribir respuesta fake: %v", err)
	}
}

// TestCommandDispatchRealMatcher verifica el dispatcher REAL de la librería
// (SPEC-080): los handlers se registran SIN slash y el matcher de
// MatchTypeCommand los hace coincidir con /pendientes y /start. Con el
// patrón viejo ("/pendientes" con slash) el matcher NO coincidía y todo
// caía al handler default ("Comando no reconocido").
func TestCommandDispatchRealMatcher(t *testing.T) {
	settings := testTelegramSettings(t)

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de test: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	svc := NewTelegramBotService(settings, storage.NewBillStorage(database), storage.NewDebtBillStorage(database))

	var sent []string
	var mu sync.Mutex
	srv := fakeTelegramAPI(t, &sent, &mu)

	b, err := bot.New("TESTTOKEN", bot.WithServerURL(srv.URL), bot.WithNotAsyncHandlers(), bot.WithDefaultHandler(svc.handleDefault))
	if err != nil {
		t.Fatalf("bot.New: %v", err)
	}
	b.RegisterHandler(bot.HandlerTypeMessageText, "servicios_pendientes", bot.MatchTypeCommand, svc.handleServiciosPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "deudas_pendientes", bot.MatchTypeCommand, svc.handleDeudasPendientes)
	b.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, svc.handleStart)

	ctx := context.Background()

	msg := func(text string) *tgmodels.Update {
		return &tgmodels.Update{
			ID: 1,
			Message: &tgmodels.Message{
				Chat: tgmodels.Chat{ID: 123},
				Text: text,
				Entities: []tgmodels.MessageEntity{
					{Type: tgmodels.MessageEntityTypeBotCommand, Offset: 0, Length: len(text)},
				},
			},
		}
	}

	t.Run("servicios_pendientes matchea y responde datos", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/servicios_pendientes"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /servicios_pendientes cayó al default handler")
		}
		last := replies[len(replies)-1]
		if strings.Contains(last, "Comando no reconocido") {
			t.Fatalf("/servicios_pendientes respondió con el default handler: %q", last)
		}
		if !strings.Contains(last, "pendientes") {
			t.Errorf("respuesta inesperada: %q", last)
		}
	})

	t.Run("deudas_pendientes matchea y responde datos", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/deudas_pendientes"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /deudas_pendientes cayó al default handler")
		}
		last := replies[len(replies)-1]
		if strings.Contains(last, "Comando no reconocido") {
			t.Fatalf("/deudas_pendientes respondió con el default handler: %q", last)
		}
		if !strings.Contains(last, "pendientes") {
			t.Errorf("respuesta inesperada: %q", last)
		}
	})

	t.Run("start matchea y responde bienvenida", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/start"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta: /start cayó al default handler")
		}
		last := replies[len(replies)-1]
		if !strings.Contains(last, "Bienvenida") && !strings.Contains(last, "Comandos") {
			t.Errorf("respuesta de /start inesperada: %q", last)
		}
	})

	t.Run("comando desconocido cae al default", func(t *testing.T) {
		before := len(sent)
		b.ProcessUpdate(ctx, msg("/otro"))
		mu.Lock()
		replies := sent[before:]
		mu.Unlock()
		if len(replies) == 0 {
			t.Fatal("no se envió respuesta para comando desconocido")
		}
		last := replies[len(replies)-1]
		if !strings.Contains(last, "Comando no reconocido") {
			t.Errorf("esperado default handler, got: %q", last)
		}
	})
}

var _ = bot.HandlerFunc(nil) // mantener import de bot en tests
