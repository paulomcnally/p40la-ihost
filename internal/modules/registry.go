package modules

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sort"
	"sync"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// Registry mantiene los módulos disponibles por ID.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module
	order   []string
}

// NewRegistry crea un registry vacío.
func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]Module)}
}

// Register agrega un módulo. Un ID repetido reemplaza al anterior para que los
// tests puedan registrar dobles.
func (r *Registry) Register(m Module) {
	if m == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := m.Info().ID
	if _, exists := r.modules[id]; !exists {
		r.order = append(r.order, id)
	}
	r.modules[id] = m
}

// Get devuelve un módulo por ID.
func (r *Registry) Get(id string) (Module, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.modules[id]
	return m, ok
}

// List devuelve los metadatos de todos los módulos, ordenados por ID.
func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.modules))
	for _, m := range r.modules {
		out = append(out, m.Info())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Importers devuelve los módulos que implementan StatementImporter.
func (r *Registry) Importers() []StatementImporter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]StatementImporter, 0, len(r.order))
	for _, id := range r.order {
		if imp, ok := r.modules[id].(StatementImporter); ok {
			out = append(out, imp)
		}
	}
	return out
}

// FindImporter devuelve el primer importador que acepta el archivo.
func (r *Registry) FindImporter(filename, mimeType string) (StatementImporter, bool) {
	for _, imp := range r.Importers() {
		if imp.CanParse(filename, mimeType) {
			return imp, true
		}
	}
	return nil, false
}

// ParseStatements prueba los importadores registrados y devuelve el primero cuyo
// layout coincide con el archivo (SPEC-139). Un error distinto de
// ErrUnsupportedStatement se propaga: un PDF corrupto no debe confundirse con un
// layout desconocido.
func (r *Registry) ParseStatements(ctx context.Context, filename, mimeType string, data []byte) (*models.ParsedStatement, StatementImporter, error) {
	var unsupported error
	for _, imp := range r.Importers() {
		if !imp.CanParse(filename, mimeType) {
			continue
		}
		parsed, err := imp.Parse(ctx, bytes.NewReader(data))
		if err == nil {
			return parsed, imp, nil
		}
		if errors.Is(err, ErrUnsupportedStatement) {
			unsupported = err
			continue
		}
		return nil, nil, err
	}
	if unsupported != nil {
		return nil, nil, unsupported
	}
	return nil, nil, ErrUnsupportedStatement
}

// RegisterRoutes invoca el registro de rutas de cada módulo que lo implemente.
func (r *Registry) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, id := range r.order {
		if rt, ok := r.modules[id].(Router); ok {
			rt.RegisterRoutes(mux, auth)
		}
	}
}
