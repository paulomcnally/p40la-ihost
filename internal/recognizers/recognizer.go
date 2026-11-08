// Package recognizers contiene reconocedores de documentos a nivel proyecto.
//
// A diferencia de internal/analyzers (orientado a facturas de servicios por
// institución), este paquete reconoce comprobantes de pago genéricos y
// devuelve un contrato reutilizable por deudas, servicios u otros módulos.
package recognizers

import (
	"errors"
	"io"
	"sync"
	"time"
)

var (
	// ErrUnsupportedFormat indica que ningún reconocedor identificó el PDF.
	ErrUnsupportedFormat = errors.New("formato de comprobante no soportado")
	// ErrNeedsOCR indica que el PDF no tiene una capa de texto extraíble.
	ErrNeedsOCR = errors.New("el PDF no tiene texto extraíble")
	// ErrInvalidPDF indica que el archivo no es un PDF válido.
	ErrInvalidPDF = errors.New("el PDF no es válido")
)

// RecognizerInfo describe un reconocedor registrado.
type RecognizerInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Source es el PDF a reconocer. ReaderAt permite leer por páginas sin cargar
// el archivo completo en memoria; Size es el tamaño en bytes.
type Source struct {
	ReaderAt io.ReaderAt
	Size     int64
	MimeType string
}

// RecognizedDocument es el resultado normalizado de un comprobante.
type RecognizedDocument struct {
	RecognizerID      string     `json:"recognizer_id,omitempty"`
	DocumentType      string     `json:"document_type"`
	Reference         string     `json:"reference,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency,omitempty"`
	InstallmentNumber int        `json:"installment_number,omitempty"`
	Concept           string     `json:"concept,omitempty"`
	Confidence        float64    `json:"confidence"`
	Warnings          []string   `json:"warnings,omitempty"`
}

// ReceiptRecognizer detecta y extrae datos de un comprobante.
type ReceiptRecognizer interface {
	Info() RecognizerInfo
	Recognize(src Source) (*RecognizedDocument, error)
}

// Registry mantiene los reconocedores disponibles y prueba cada uno hasta que
// uno detecte el formato.
type Registry struct {
	mu    sync.RWMutex
	items []ReceiptRecognizer
}

// NewRegistry crea un registry vacío.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register agrega un reconocedor al registry.
func (r *Registry) Register(recognizer ReceiptRecognizer) {
	if recognizer == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, recognizer)
}

// List devuelve la información de los reconocedores registrados.
func (r *Registry) List() []RecognizerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infos := make([]RecognizerInfo, 0, len(r.items))
	for _, item := range r.items {
		infos = append(infos, item.Info())
	}
	return infos
}

// Recognize prueba los reconocedores registrados en orden. Si ninguno
// reconoce el formato devuelve ErrUnsupportedFormat; si el PDF es inválido o
// no tiene texto devuelve el error específico para que la UI pueda explicarlo.
func (r *Registry) Recognize(src Source) (*RecognizedDocument, error) {
	if src.ReaderAt == nil || src.Size <= 0 {
		return nil, ErrInvalidPDF
	}

	r.mu.RLock()
	items := append([]ReceiptRecognizer(nil), r.items...)
	r.mu.RUnlock()

	var firstErr error
	for _, item := range items {
		doc, err := item.Recognize(src)
		if err == nil {
			if doc.RecognizerID == "" {
				doc.RecognizerID = item.Info().ID
			}
			return doc, nil
		}
		if errors.Is(err, ErrUnsupportedFormat) {
			continue
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, ErrUnsupportedFormat
}

var defaultRegistry = NewRegistry()

// Register agrega un reconocedor al registry global.
func Register(recognizer ReceiptRecognizer) {
	defaultRegistry.Register(recognizer)
}

// List devuelve los reconocedores globales registrados.
func List() []RecognizerInfo {
	return defaultRegistry.List()
}

// Recognize ejecuta el reconocimiento con el registry global.
func Recognize(src Source) (*RecognizedDocument, error) {
	return defaultRegistry.Recognize(src)
}
