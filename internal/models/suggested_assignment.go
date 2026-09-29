package models

// SuggestedAssignment es la respuesta del endpoint de sugerencia de asignación
// de una categoría (SPEC-098): la suma de la factura más reciente por cada
// servicio vinculado, agrupada por moneda (map currency_id -> monto).
type SuggestedAssignment struct {
	Suggested          map[int64]float64 `json:"suggested"`
	Applies            bool              `json:"applies"`
	SourceServiceIDs   []int64           `json:"source_service_ids,omitempty"`
	SourceServiceNames []string          `json:"source_service_names,omitempty"`
}
