package modules

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type fakeImporter struct {
	id        string
	supported bool
	err       error
}

func (f fakeImporter) Info() Info { return Info{ID: f.id, Version: "1.0.0"} }

func (f fakeImporter) CanParse(string, string) bool { return true }

func (f fakeImporter) Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error) {
	if f.err != nil {
		return nil, f.err
	}
	if !f.supported {
		return nil, fmt.Errorf("%w: otro layout", ErrUnsupportedStatement)
	}
	return &models.ParsedStatement{ModuleID: f.id}, nil
}

func TestParseStatementsPicksMatchingImporter(t *testing.T) {
	registry := NewRegistry()
	registry.Register(fakeImporter{id: "bank.a.ni.x"})
	registry.Register(fakeImporter{id: "bank.b.ni.x", supported: true})

	parsed, importer, err := registry.ParseStatements(context.Background(), "estado.pdf", "application/pdf", []byte("%PDF-1"))
	if err != nil {
		t.Fatalf("ParseStatements: %v", err)
	}
	if parsed.ModuleID != "bank.b.ni.x" {
		t.Fatalf("módulo = %q, esperaba bank.b.ni.x", parsed.ModuleID)
	}
	if importer.Info().ID != "bank.b.ni.x" {
		t.Fatalf("importador = %q", importer.Info().ID)
	}
}

func TestParseStatementsUnsupported(t *testing.T) {
	registry := NewRegistry()
	registry.Register(fakeImporter{id: "bank.a.ni.x"})
	registry.Register(fakeImporter{id: "bank.b.ni.x"})

	if _, _, err := registry.ParseStatements(context.Background(), "estado.pdf", "application/pdf", nil); !errors.Is(err, ErrUnsupportedStatement) {
		t.Fatalf("err = %v, esperaba ErrUnsupportedStatement", err)
	}
}

func TestParseStatementsPropagatesRealErrors(t *testing.T) {
	registry := NewRegistry()
	boom := errors.New("pdf corrupto")
	registry.Register(fakeImporter{id: "bank.a.ni.x", err: boom})

	if _, _, err := registry.ParseStatements(context.Background(), "estado.pdf", "application/pdf", nil); !errors.Is(err, boom) {
		t.Fatalf("err = %v, esperaba el error del parser", err)
	}
}

func TestParseStatementsWithoutImporters(t *testing.T) {
	registry := NewRegistry()
	if _, _, err := registry.ParseStatements(context.Background(), "estado.pdf", "application/pdf", nil); !errors.Is(err, ErrUnsupportedStatement) {
		t.Fatalf("err = %v, esperaba ErrUnsupportedStatement", err)
	}
}
