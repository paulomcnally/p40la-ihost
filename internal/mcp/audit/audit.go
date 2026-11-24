// Package audit escribe un log JSONL de llamadas MCP con rotación por tamaño.
// Nunca registra cuerpos de request ni datos de negocio (SPEC-147, SEC-LOGS).
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	fileName  = "audit.jsonl"
	maxFiles  = 3
	filePerms = 0o600
	dirPerms  = 0o700
)

// Entry es una línea de auditoría. No incluye argumentos ni resultados.
type Entry struct {
	Time       string `json:"ts"`
	RequestID  string `json:"request_id,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Tool       string `json:"tool"`
	Result     string `json:"result"`
	DurationMS int64  `json:"duration_ms"`
}

// Logger escribe entradas de auditoría con rotación.
type Logger struct {
	dir      string
	maxBytes int64

	mu   sync.Mutex
	file *os.File
	size int64
}

// New crea el logger en dir y abre (o crea) audit.jsonl con permisos 0600.
func New(dir string, maxMB int) (*Logger, error) {
	if dir == "" {
		return nil, fmt.Errorf("directorio de auditoría vacío")
	}
	if maxMB <= 0 {
		maxMB = 5
	}
	if err := os.MkdirAll(dir, dirPerms); err != nil {
		return nil, fmt.Errorf("crear directorio de auditoría: %w", err)
	}
	logger := &Logger{dir: dir, maxBytes: int64(maxMB) * 1024 * 1024}
	if err := logger.open(); err != nil {
		return nil, err
	}
	return logger, nil
}

func (l *Logger) open() error {
	path := filepath.Join(l.dir, fileName)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerms)
	if err != nil {
		return fmt.Errorf("abrir auditoría: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("estadísticas de auditoría: %w", err)
	}
	l.file = file
	l.size = info.Size()
	return nil
}

// Log agrega una entrada. La hora se completa si viene vacía.
func (l *Logger) Log(entry Entry) error {
	if entry.Time == "" {
		entry.Time = time.Now().UTC().Format(time.RFC3339)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		if err := l.open(); err != nil {
			return err
		}
	}
	if l.size > 0 && l.size+int64(len(line)) > l.maxBytes {
		if err := l.rotate(); err != nil {
			return err
		}
	}
	n, err := l.file.Write(line)
	l.size += int64(n)
	if err != nil {
		return fmt.Errorf("escribir auditoría: %w", err)
	}
	return nil
}

// rotate cierra el archivo actual y desplaza audit.jsonl -> .1 -> .2 -> .3.
func (l *Logger) rotate() error {
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("cerrar auditoría: %w", err)
		}
		l.file = nil
	}
	for i := maxFiles; i >= 1; i-- {
		from := filepath.Join(l.dir, fileName)
		if i > 1 {
			from = filepath.Join(l.dir, fmt.Sprintf("audit.%d.jsonl", i-1))
		}
		to := filepath.Join(l.dir, fmt.Sprintf("audit.%d.jsonl", i))
		if _, err := os.Stat(from); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if i == maxFiles {
			if err := os.Remove(to); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("rotar auditoría: %w", err)
		}
	}
	l.size = 0
	return l.open()
}

// Close cierra el archivo de auditoría.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}
