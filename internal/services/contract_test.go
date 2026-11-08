package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func contractTestPDF(body string) []byte {
	return []byte("%PDF-1.4\n" + body + "\n%%EOF\n")
}

func TestContractStorePaths(t *testing.T) {
	dir := t.TempDir()
	store := NewContractStore(dir)

	debtPath, err := store.path(ContractOwnerDebt, 7)
	if err != nil {
		t.Fatalf("path deuda: %v", err)
	}
	wantDebt := filepath.Join(dir, "contracts", "debts", "7", "contract.pdf")
	if debtPath != wantDebt {
		t.Errorf("path deuda = %q, want %q", debtPath, wantDebt)
	}

	servicePath, err := store.path(ContractOwnerService, 9)
	if err != nil {
		t.Fatalf("path servicio: %v", err)
	}
	wantService := filepath.Join(dir, "contracts", "services", "9", "contract.pdf")
	if servicePath != wantService {
		t.Errorf("path servicio = %q, want %q", servicePath, wantService)
	}

	if _, err := store.path(ContractOwner("other"), 1); err == nil {
		t.Error("owner inválido debería devolver error")
	}
	if _, err := store.path(ContractOwnerDebt, 0); err == nil {
		t.Error("id 0 debería devolver error")
	}
	if _, err := store.path(ContractOwnerDebt, -1); err == nil {
		t.Error("id negativo debería devolver error")
	}
}

func TestContractStoreSaveReplaceAndDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewContractStore(dir)

	if store.Exists(ContractOwnerDebt, 1) {
		t.Fatal("no debería existir contrato inicial")
	}
	if _, err := store.Path(ContractOwnerDebt, 1); !errors.Is(err, ErrContractNotFound) {
		t.Fatalf("Path sin archivo = %v, want ErrContractNotFound", err)
	}

	first := contractTestPDF("primera versión")
	if err := store.Save(ContractOwnerDebt, 1, "Contrato Original.PDF", bytes.NewReader(first)); err != nil {
		t.Fatalf("Save inicial: %v", err)
	}
	if !store.Exists(ContractOwnerDebt, 1) {
		t.Fatal("Exists debería ser true después de Save")
	}

	path, err := store.Path(ContractOwnerDebt, 1)
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatalf("stat contrato: %v", err)
	} else if info.Mode().Perm() != 0o644 {
		t.Errorf("permisos = %o, want 644", info.Mode().Perm())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer contrato: %v", err)
	}
	if !bytes.Equal(got, first) {
		t.Error("el contenido guardado no coincide")
	}

	second := contractTestPDF("segunda versión")
	if err := store.Save(ContractOwnerDebt, 1, "contrato.pdf", bytes.NewReader(second)); err != nil {
		t.Fatalf("Save reemplazo: %v", err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer contrato reemplazado: %v", err)
	}
	if !bytes.Equal(got, second) {
		t.Error("el reemplazo no actualizó el contenido")
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, "contracts", "debts", "1", "*.tmp"))
	if err != nil {
		t.Fatalf("glob temporales: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Errorf("quedaron temporales: %v", tmpFiles)
	}

	if err := store.Delete(ContractOwnerDebt, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.Exists(ContractOwnerDebt, 1) {
		t.Error("Exists debería ser false después de Delete")
	}
	if err := store.Delete(ContractOwnerDebt, 1); !errors.Is(err, ErrContractNotFound) {
		t.Fatalf("segundo Delete = %v, want ErrContractNotFound", err)
	}
}

func TestContractStoreRejectsInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	store := NewContractStore(dir)

	tests := []struct {
		name     string
		fileName string
		data     []byte
		wantErr  error
	}{
		{name: "vacío", fileName: "contract.pdf", data: nil, wantErr: ErrContractInvalid},
		{name: "extensión incorrecta", fileName: "contract.txt", data: contractTestPDF("x"), wantErr: ErrContractInvalid},
		{name: "firma incorrecta", fileName: "contract.pdf", data: []byte("no es un pdf"), wantErr: ErrContractInvalid},
		{name: "demasiado grande", fileName: "contract.pdf", data: append([]byte("%PDF-"), bytes.Repeat([]byte("A"), MaxContractSize)...), wantErr: ErrContractTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Save(ContractOwnerService, 1, tt.fileName, bytes.NewReader(tt.data))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Save = %v, want %v", err, tt.wantErr)
			}
			if store.Exists(ContractOwnerService, 1) {
				t.Error("no debería existir contrato después de un archivo inválido")
			}
			tmpFiles, globErr := filepath.Glob(filepath.Join(dir, "contracts", "services", "1", "*.tmp"))
			if globErr != nil {
				t.Fatalf("glob temporales: %v", globErr)
			}
			if len(tmpFiles) != 0 {
				t.Errorf("quedaron temporales: %v", tmpFiles)
			}
		})
	}
}
