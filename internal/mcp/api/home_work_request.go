package api

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// HomeWorkRequestAPI es el contrato del módulo home.workrequest que consumen
// las tools MCP (lectura y escritura).
type HomeWorkRequestAPI interface {
	ListHomeWorkRequests(ctx context.Context, homeID int64, filter HomeWorkRequestFilter) (*HomeWorkRequestList, error)
	GetHomeWorkRequest(ctx context.Context, homeID, workRequestID int64) (*HomeWorkRequest, error)
	CreateHomeWorkRequest(ctx context.Context, homeID int64, req HomeWorkRequestRequest) (*HomeWorkRequest, error)
	UpdateHomeWorkRequest(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestRequest) (*HomeWorkRequest, error)
	ConvertHomeWorkRequest(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestConvertRequest) (*HomeWorkRequestConvertResult, error)
	SetHomeWorkRequestStatus(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestStatusRequest) (*HomeWorkRequest, error)

	CreateHomeWorkRequestUploadLink(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestUploadLinkRequest) (*HomeWorkRequestUploadLink, error)
	AddHomeWorkRequestEvidence(ctx context.Context, homeID, workRequestID int64, fileName string, data []byte) (*HomeWorkRequestEvidenceResult, error)
	DeleteHomeWorkRequestAttachment(ctx context.Context, homeID, workRequestID, attachmentID int64) error
}

// HomeWorkRequestFilter acota el listado de reportes de una vivienda.
type HomeWorkRequestFilter struct {
	Status   string
	Category string
}

// HomeWorkRequest es un reporte de problema o trabajo deseado.
type HomeWorkRequest struct {
	ID              int64                       `json:"id"`
	HomeID          int64                       `json:"home_id"`
	Title           string                      `json:"title"`
	Description     string                      `json:"description"`
	Category        string                      `json:"category"`
	Priority        string                      `json:"priority"`
	Status          string                      `json:"status"`
	ReportedAt      string                      `json:"reported_at"`
	MaintenanceID   *int64                      `json:"maintenance_id"`
	ConvertedAt     *string                     `json:"converted_at"`
	AttachmentCount int                         `json:"attachment_count"`
	Attachments     []HomeWorkRequestAttachment `json:"attachments,omitempty"`
	Deduplicated    bool                        `json:"deduplicated,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

// HomeWorkRequestAttachment es la evidencia (foto o video) de un reporte.
type HomeWorkRequestAttachment struct {
	ID            int64     `json:"id"`
	WorkRequestID int64     `json:"work_request_id"`
	MediaType     string    `json:"media_type"`
	FileName      string    `json:"file_name"`
	MimeType      string    `json:"mime_type"`
	SizeBytes     int64     `json:"size_bytes"`
	SHA256        string    `json:"sha256"`
	CreatedAt     time.Time `json:"created_at"`
	Deduplicated  bool      `json:"deduplicated,omitempty"`
}

// HomeWorkRequestList es la respuesta de GET /api/homes/{id}/workrequests.
type HomeWorkRequestList struct {
	Items        []HomeWorkRequest `json:"items"`
	PendingCount int               `json:"pending_count"`
	Truncated    bool              `json:"truncated"`
}

// HomeWorkRequestRequest es el cuerpo de POST y PUT. El PUT reemplaza los
// campos editables completos: la tool envía el estado actual con el merge.
type HomeWorkRequestRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Priority        string  `json:"priority"`
	ReportedAt      string  `json:"reported_at"`
	ClientRequestID *string `json:"client_request_id,omitempty"`
}

// HomeWorkRequestUploadLinkRequest es el cuerpo de POST .../upload-link.
type HomeWorkRequestUploadLinkRequest struct {
	Count      int `json:"count,omitempty"`
	TTLMinutes int `json:"ttl_minutes,omitempty"`
}

// HomeWorkRequestUploadLink es la respuesta del backend para emitir un link.
type HomeWorkRequestUploadLink struct {
	Token           string   `json:"token"`
	Path            string   `json:"path"`
	ExpiresAt       string   `json:"expires_at"`
	MaxBytes        int64    `json:"max_bytes"`
	MaxRequestBytes int64    `json:"max_request_bytes"`
	MaxFiles        int      `json:"max_files"`
	AcceptedMIME    []string `json:"accepted_mime"`
	AttachmentCount int      `json:"attachment_count"`
	RemainingSlots  int      `json:"remaining_slots"`
	WorkRequestID   int64    `json:"work_request_id"`
	HomeID          int64    `json:"home_id"`
}

// HomeWorkRequestEvidenceResult es la respuesta de la carga inline.
type HomeWorkRequestEvidenceResult struct {
	Attachment   HomeWorkRequestAttachment `json:"attachment"`
	Deduplicated bool                      `json:"deduplicated"`
}

// HomeWorkRequestConvertRequest es el cuerpo de POST .../convert.
type HomeWorkRequestConvertRequest struct {
	Kind string `json:"kind,omitempty"`
	Area string `json:"area,omitempty"`
}

// HomeWorkRequestConvertResult es la respuesta de POST .../convert.
type HomeWorkRequestConvertResult struct {
	MaintenanceID int64            `json:"maintenance_id"`
	WorkRequest   *HomeWorkRequest `json:"work_request"`
}

// HomeWorkRequestStatusRequest es el cuerpo de POST .../status.
type HomeWorkRequestStatusRequest struct {
	Status string `json:"status"`
}

// ListHomeWorkRequests devuelve los reportes de una vivienda con filtros.
func (c *Client) ListHomeWorkRequests(ctx context.Context, homeID int64, filter HomeWorkRequestFilter) (*HomeWorkRequestList, error) {
	query := url.Values{}
	if filter.Status != "" {
		query.Set("status", filter.Status)
	}
	if filter.Category != "" {
		query.Set("category", filter.Category)
	}
	path := fmt.Sprintf("/api/homes/%d/workrequests", homeID)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var list HomeWorkRequestList
	if err := c.doJSON(ctx, "GET", path, nil, &list, nil); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetHomeWorkRequest devuelve un reporte con sus metadatos de evidencia.
func (c *Client) GetHomeWorkRequest(ctx context.Context, homeID, workRequestID int64) (*HomeWorkRequest, error) {
	var item HomeWorkRequest
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d", homeID, workRequestID)
	if err := c.doJSON(ctx, "GET", path, nil, &item, nil); err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateHomeWorkRequest registra un reporte en estado reported.
func (c *Client) CreateHomeWorkRequest(ctx context.Context, homeID int64, req HomeWorkRequestRequest) (*HomeWorkRequest, error) {
	var item HomeWorkRequest
	path := fmt.Sprintf("/api/homes/%d/workrequests", homeID)
	if err := c.doJSON(ctx, "POST", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateHomeWorkRequest reemplaza los campos editables de un reporte.
func (c *Client) UpdateHomeWorkRequest(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestRequest) (*HomeWorkRequest, error) {
	var item HomeWorkRequest
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d", homeID, workRequestID)
	if err := c.doJSON(ctx, "PUT", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// ConvertHomeWorkRequest crea una reparación vacía y enlaza el reporte.
func (c *Client) ConvertHomeWorkRequest(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestConvertRequest) (*HomeWorkRequestConvertResult, error) {
	var result HomeWorkRequestConvertResult
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d/convert", homeID, workRequestID)
	if err := c.doJSON(ctx, "POST", path, req, &result, sourceHeaders()); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetHomeWorkRequestStatus descarta o reactiva un reporte.
func (c *Client) SetHomeWorkRequestStatus(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestStatusRequest) (*HomeWorkRequest, error) {
	var item HomeWorkRequest
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d/status", homeID, workRequestID)
	if err := c.doJSON(ctx, "POST", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateHomeWorkRequestUploadLink emite un token de carga acotado al reporte.
func (c *Client) CreateHomeWorkRequestUploadLink(ctx context.Context, homeID, workRequestID int64, req HomeWorkRequestUploadLinkRequest) (*HomeWorkRequestUploadLink, error) {
	var link HomeWorkRequestUploadLink
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d/upload-link", homeID, workRequestID)
	if err := c.doJSON(ctx, "POST", path, req, &link, sourceHeaders()); err != nil {
		return nil, err
	}
	return &link, nil
}

// AddHomeWorkRequestEvidence sube un archivo pequeño por la vía autenticada.
func (c *Client) AddHomeWorkRequestEvidence(ctx context.Context, homeID, workRequestID int64, fileName string, data []byte) (*HomeWorkRequestEvidenceResult, error) {
	var result HomeWorkRequestEvidenceResult
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d/attachments/evidence", homeID, workRequestID)
	if err := c.postMultipart(ctx, path, fileName, data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteHomeWorkRequestAttachment elimina la evidencia del reporte.
func (c *Client) DeleteHomeWorkRequestAttachment(ctx context.Context, homeID, workRequestID, attachmentID int64) error {
	path := fmt.Sprintf("/api/homes/%d/workrequests/%d/attachments/%d", homeID, workRequestID, attachmentID)
	return c.doJSON(ctx, "DELETE", path, nil, nil, sourceHeaders())
}
