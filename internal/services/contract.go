package services

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MaxContractSize es el tamaño máximo permitido para un contrato PDF (SPEC-123).
const MaxContractSize = 10 * 1024 * 1024

// ContractOwner identifica la entidad dueña de un contrato.
type ContractOwner string

const (
	ContractOwnerDebt    ContractOwner = "debts"
	ContractOwnerService ContractOwner = "services"
)

var (
	// ErrContractNotFound indica que el contrato no existe en disco.
	ErrContractNotFound = errors.New("el contrato no existe")
	// ErrContractEntityNotFound indica que la deuda o el servicio no existe.
	ErrContractEntityNotFound = errors.New("la deuda o el servicio no existe")
	// ErrContractInvalid indica que el archivo no es un PDF válido.
	ErrContractInvalid = errors.New("el archivo de contrato no es un PDF válido")
	// ErrContractTooLarge indica que el archivo supera el límite de 10 MB.
	ErrContractTooLarge = errors.New("el contrato supera el tamaño máximo de 10 MB")
)

// ContractStore guarda contratos PDF en el volumen de datos (SPEC-123).
// No conoce la base de datos: la existencia de la entidad la validan los
// services de deuda/servicio antes de llamar a estos métodos.
type ContractStore struct {
	rootDir string // {DATA_DIR}/contracts
}

// NewContractStore crea un almacén de contratos bajo {dataDir}/contracts.
func NewContractStore(dataDir string) *ContractStore {
	return &ContractStore{rootDir: filepath.Join(dataDir, "contracts")}
}

// Exists informa si existe el contrato de una entidad.
func (s *ContractStore) Exists(owner ContractOwner, id int64) bool {
	path, err := s.path(owner, id)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Save valida y persiste un contrato de forma atómica.
// El nombre original se usa solo para validar la extensión; el archivo final
// siempre se llama contract.pdf.
func (s *ContractStore) Save(owner ContractOwner, id int64, fileName string, r io.Reader) error {
	if strings.ToLower(filepath.Ext(fileName)) != ".pdf" {
		return ErrContractInvalid
	}

	dest, err := s.path(owner, id)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("crear directorio de contratos: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".contract-*.tmp")
	if err != nil {
		return fmt.Errorf("crear archivo temporal de contrato: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("permisos del contrato: %w", err)
	}

	br := bufio.NewReader(r)
	head, err := br.Peek(5)
	if err != nil || string(head) != "%PDF-" {
		cleanup()
		return ErrContractInvalid
	}

	written, err := io.Copy(tmp, io.LimitReader(br, MaxContractSize+1))
	if err != nil {
		cleanup()
		return fmt.Errorf("guardar contrato: %w", err)
	}
	if written > MaxContractSize {
		cleanup()
		return ErrContractTooLarge
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sincronizar contrato: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("cerrar contrato: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("reemplazar contrato: %w", err)
	}
	return nil
}

// Path devuelve la ruta del contrato si existe.
func (s *ContractStore) Path(owner ContractOwner, id int64) (string, error) {
	path, err := s.path(owner, id)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", ErrContractNotFound
	}
	return path, nil
}

// Delete elimina el contrato de una entidad.
func (s *ContractStore) Delete(owner ContractOwner, id int64) error {
	path, err := s.path(owner, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrContractNotFound
		}
		return fmt.Errorf("eliminar contrato: %w", err)
	}
	return nil
}

func (s *ContractStore) path(owner ContractOwner, id int64) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("id de contrato inválido: %d", id)
	}
	base, err := s.ownerDir(owner)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, strconv.FormatInt(id, 10), "contract.pdf"), nil
}

func (s *ContractStore) ownerDir(owner ContractOwner) (string, error) {
	switch owner {
	case ContractOwnerDebt, ContractOwnerService:
		return filepath.Join(s.rootDir, string(owner)), nil
	default:
		return "", fmt.Errorf("tipo de contrato inválido: %q", owner)
	}
}
