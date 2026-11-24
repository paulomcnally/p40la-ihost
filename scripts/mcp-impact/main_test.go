package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func loadTestManifest(t *testing.T) *Manifest {
	t.Helper()
	manifest, err := loadManifest("../../docs/mcp/tools/impact.json")
	if err != nil {
		t.Fatalf("loadManifest: %v", err)
	}
	return manifest
}

func impactNames(impact []ToolImpact) []string {
	names := make([]string, 0, len(impact))
	for _, tool := range impact {
		names = append(names, tool.Name)
	}
	return names
}

func TestManifestMatchesCode(t *testing.T) {
	manifest := loadTestManifest(t)
	if err := validateManifest("../..", manifest); err != nil {
		t.Fatalf("validateManifest: %v", err)
	}
}

func TestImpactBulkService(t *testing.T) {
	manifest := loadTestManifest(t)
	impact := computeImpact(manifest, []string{"internal/services/bill_bulk.go"})
	want := []string{"bills_generate", "bills_pay"}
	if got := impactNames(impact); !reflect.DeepEqual(got, want) {
		t.Fatalf("tools afectadas = %v, quiero %v", got, want)
	}
}

func TestImpactSharedRoute(t *testing.T) {
	manifest := loadTestManifest(t)
	impact := computeImpact(manifest, []string{"internal/api/routes.go"})
	want := make([]string, 0, len(manifest.Tools))
	for _, tool := range manifest.Tools {
		want = append(want, tool.Name)
	}
	sort.Strings(want)
	if got := impactNames(impact); !reflect.DeepEqual(got, want) {
		t.Fatalf("tools afectadas = %v, quiero %v", got, want)
	}
}

func TestImpactListHandler(t *testing.T) {
	manifest := loadTestManifest(t)
	impact := computeImpact(manifest, []string{"internal/api/bill_handlers.go"})
	want := []string{"bills_list"}
	if got := impactNames(impact); !reflect.DeepEqual(got, want) {
		t.Fatalf("tools afectadas = %v, quiero %v", got, want)
	}
}

func TestImpactNoMCPSurface(t *testing.T) {
	manifest := loadTestManifest(t)
	impact := computeImpact(manifest, []string{
		"docs/mcp/tools/bills.md",
		"README.md",
		"internal/mcp/tools/all/contract_test.go",
	})
	if len(impact) != 0 {
		t.Fatalf("tools afectadas = %v, quiero ninguna", impactNames(impact))
	}
}

func TestImpactSection(t *testing.T) {
	content := `# Spec

## 6. Impacto en tools MCP

| Tool | Afectada |
|---|---|
| bills_pay | Sí |

**Validación subagente**: explore — sesión demo — veredicto: revisar pago.

## 7. Criterios
`
	section := impactSection(content)
	if section == "" {
		t.Fatal("no se encontró la sección de impacto")
	}
	if !containsWord(section, "bills_pay") {
		t.Fatal("la sección no menciona bills_pay")
	}
	if _, ok := subagentValidation(section); !ok {
		t.Fatal("la validación de subagente debería ser válida")
	}

	pending := `## Impacto en tools MCP
**Validación subagente**: pendiente
`
	if _, ok := subagentValidation(pending); ok {
		t.Fatal("pendiente no debe considerarse válido")
	}
}

func TestValidateNewSpecs(t *testing.T) {
	root := t.TempDir()
	specsDir := filepath.Join(root, "docs", "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	good := "docs/specs/SPEC-999-demo.md"
	goodContent := `# Demo
## Impacto en tools MCP
Ninguna tool afectada.
**Validación subagente**: explore — sesión demo — veredicto: ninguna.
`
	if err := os.WriteFile(filepath.Join(root, good), []byte(goodContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateNewSpecs(root, []string{good}); err != nil {
		t.Fatalf("spec válida rechazada: %v", err)
	}

	bad := "docs/specs/SPEC-998-bad.md"
	if err := os.WriteFile(filepath.Join(root, bad), []byte("# Sin impacto\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateNewSpecs(root, []string{bad}); err == nil {
		t.Fatal("una spec sin impacto debería fallar")
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern string
		file    string
		want    bool
	}{
		{"internal/models/", "internal/models/bill.go", true},
		{"internal/models/", "internal/model/bill.go", false},
		{"internal/services/bill*.go", "internal/services/bill_bulk.go", true},
		{"internal/services/bill*.go", "internal/services/currency.go", false},
		{"internal/api/routes.go", "internal/api/routes.go", true},
		{"internal/api/routes.go", "internal/api/routes_test.go", false},
	}
	for _, tc := range cases {
		if got := matchPattern(tc.pattern, tc.file); got != tc.want {
			t.Errorf("matchPattern(%q, %q) = %v, quiero %v", tc.pattern, tc.file, got, tc.want)
		}
	}
}
