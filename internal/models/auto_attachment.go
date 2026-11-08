package models

import "time"

// AutoAttachment representa un PDF adjunto a un mantenimiento o a una
// inspección mecánica (SPEC-124). El archivo vive en DATA_DIR y la base solo
// guarda sus metadatos.
type AutoAttachment struct {
	ID        int64     `json:"id"`
	AutoID    int64     `json:"auto_id"`
	OwnerType string    `json:"owner_type"`
	OwnerID   int64     `json:"owner_id"`
	FileName  string    `json:"file_name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	AutoAttachmentOwnerMaintenance = "maintenance"
	AutoAttachmentOwnerInspection  = "inspection"
)
