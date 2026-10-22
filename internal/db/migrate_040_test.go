package db

import (
	"testing"
)

func TestMigrate040NormalizeAutoColors(t *testing.T) {
	database := openMigrateTest(t, "mig040")

	rows := []struct{ color, want string }{
		{"Blanco", "white"},
		{"GRIS", "gray"},
		{"Marrón", "brown"},
		{"Rojo ", "red"},
		{"light-blue", "light_blue"},
		{"white", "white"},
		{"Azul Marino", ""},
		{"", ""},
		{"White", "white"},
		{" red ", "red"},
	}
	for _, r := range rows {
		if _, err := database.Exec(
			"INSERT INTO autos (year, model, brand, color) VALUES (2026, 'Corolla', 'Toyota', ?)",
			r.color,
		); err != nil {
			t.Fatalf("insertar auto con color %q: %v", r.color, err)
		}
	}

	up, err := readMigrationFile(t, "0040_normalize_auto_colors.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("ejecutar migración 0040 up: %v", err)
	}

	id := int64(0)
	for _, r := range rows {
		id++
		var got string
		if err := database.QueryRow("SELECT color FROM autos WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("leer color id %d: %v", id, err)
		}
		if got != r.want {
			t.Errorf("color %q: esperado %q, got %q", r.color, r.want, got)
		}
	}

	// Idempotencia: un segundo pase no cambia los valores.
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("segundo pase de la migración 0040: %v", err)
	}
	wants := []string{"white", "gray", "brown", "red", "light_blue", "white", "", "", "white", "red"}
	for i, want := range wants {
		var got string
		if err := database.QueryRow("SELECT color FROM autos WHERE id = ?", i+1).Scan(&got); err != nil {
			t.Fatalf("leer color id %d en segundo pase: %v", i+1, err)
		}
		if got != want {
			t.Errorf("segundo pase id %d: esperado %q, got %q", i+1, want, got)
		}
	}

	// Down best-effort: los códigos vuelven a etiquetas en español.
	down, err := readMigrationFile(t, "0040_normalize_auto_colors.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar migración 0040 down: %v", err)
	}
	var got string
	if err := database.QueryRow("SELECT color FROM autos WHERE id = 1").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "Blanco" {
		t.Errorf("down id 1: esperado Blanco, got %q", got)
	}
	if err := database.QueryRow("SELECT color FROM autos WHERE id = 7").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("down de valor limpiado: esperado vacío, got %q", got)
	}
}
