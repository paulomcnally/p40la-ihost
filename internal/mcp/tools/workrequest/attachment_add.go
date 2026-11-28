package workrequest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// attachmentAddInput es la entrada de home_work_request_attachment_add.
type attachmentAddInput struct {
	HomeID        int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64   `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	FileName      string  `json:"filename" jsonschema:"Nombre del archivo con extensión, por ejemplo foto-demo.jpg"`
	MIME          *string `json:"mime,omitempty" jsonschema:"MIME declarado; el backend valida la firma real (image/jpeg, image/png, image/webp, image/heic, video/mp4, video/quicktime, video/webm)"`
	MediaType     *string `json:"media_type,omitempty" jsonschema:"image o video; por defecto se deriva del MIME o se asume image"`
	DataBase64    string  `json:"data_base64" jsonschema:"Contenido real del archivo en base64; máximo 256 KiB decodificados"`
	SHA256        *string `json:"sha256,omitempty" jsonschema:"Hash SHA-256 hex opcional del contenido; si no coincide se rechaza"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerAttachmentAdd(server *mcp.Server) {
	destructive := false
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_attachment_add",
		Description: "Agrega una foto o video pequeño (hasta 256 KiB reales) a un reporte en estado reported. " +
			"El contenido viaja en base64 y el backend valida la firma real; es idempotente por sha256, así que " +
			"repetir el mismo archivo devuelve status=duplicate. Para archivos grandes o varios archivos, usá " +
			"attachment_upload_link. dry_run viene activado por defecto.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.attachmentAdd)
}

func (t *Toolset) attachmentAdd(ctx context.Context, _ *mcp.CallToolRequest, input attachmentAddInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	fileName := strings.TrimSpace(input.FileName)
	if fileName == "" {
		return toolkit.ToolError("invalid_request", "filename es obligatorio", "")
	}
	encoded := strings.TrimSpace(input.DataBase64)
	if encoded == "" {
		return toolkit.ToolError("invalid_request", "data_base64 es obligatorio", "")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return toolkit.ToolError("invalid_request", "data_base64 no es base64 válido", "")
	}
	if len(data) > maxInlineBytes {
		return toolkit.ToolError("invalid_request",
			"El contenido supera los 256 KiB de la vía inline.",
			"Usá home_work_request_attachment_upload_link y subí el archivo con curl.")
	}
	detectedMedia, detectedMIME := detectEvidenceType(data)
	if detectedMedia == "" {
		return toolkit.ToolError("invalid_file",
			"El contenido no es JPEG, PNG, WebP, HEIC, MP4, QuickTime ni WebM.",
			"Verificá que el archivo sea una foto o un video válido.")
	}

	mime := ""
	if input.MIME != nil {
		mime = strings.ToLower(strings.TrimSpace(*input.MIME))
		if mime != "" && !contains(acceptedMIME, mime) {
			return toolkit.ToolError("invalid_request",
				"mime debe ser image/jpeg, image/png, image/webp, image/heic, video/mp4, video/quicktime o video/webm", "")
		}
	}
	mediaType := ""
	if input.MediaType != nil {
		mediaType = strings.ToLower(strings.TrimSpace(*input.MediaType))
	}
	if mediaType != "" && mediaType != "image" && mediaType != "video" {
		return toolkit.ToolError("invalid_request", "media_type debe ser image o video", "")
	}
	if mediaType == "" {
		mediaType = detectedMedia
	}
	if mediaType != detectedMedia {
		return toolkit.ToolError("invalid_file",
			"media_type no coincide con la firma real del archivo.",
			"Enviá el archivo correcto o omití media_type para que se infiera.")
	}
	if mime == "" {
		mime = detectedMIME
	} else if strings.HasPrefix(mime, "video/") != (detectedMedia == "video") {
		return toolkit.ToolError("invalid_file",
			"mime no coincide con la firma real del archivo.", "")
	}
	if input.SHA256 != nil {
		expected := strings.ToLower(strings.TrimSpace(*input.SHA256))
		if expected != "" {
			if len(expected) != 64 {
				return toolkit.ToolError("invalid_request", "sha256 debe tener 64 caracteres hexadecimales", "")
			}
			if _, err := hex.DecodeString(expected); err != nil {
				return toolkit.ToolError("invalid_request", "sha256 debe tener 64 caracteres hexadecimales", "")
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != expected {
				return toolkit.ToolError("invalid_request", "sha256 no coincide con el contenido enviado", "")
			}
		}
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	current, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	if current.Status != "reported" {
		return toolkit.ToolError("invalid_status",
			"Solo un reporte en estado reported acepta evidencia por MCP.",
			"Reactivá el reporte con home_work_request_status o usá la web para adjuntar.")
	}
	if current.AttachmentCount >= maxAttachments {
		return toolkit.ToolError("attachment_limit",
			"El reporte alcanzó el máximo de 10 adjuntos.",
			"Borrar un adjunto con home_work_request_attachment_delete o usar otro reporte.")
	}

	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":           "planned",
			"dry_run":          true,
			"filename":         fileName,
			"size_bytes":       len(data),
			"media_type":       mediaType,
			"mime_type":        mime,
			"attachment_count": current.AttachmentCount,
		})
	}

	result, err := t.client.AddHomeWorkRequestEvidence(ctx, input.HomeID, input.WorkRequestID, fileName, data)
	if err != nil {
		return toolkit.APIError(err)
	}
	status := "added"
	count := current.AttachmentCount + 1
	if result.Deduplicated {
		status = "duplicate"
		count = current.AttachmentCount
	}
	if count > maxAttachments {
		count = maxAttachments
	}
	return toolkit.JSONResult(map[string]any{
		"status":           status,
		"dry_run":          false,
		"attachment":       result.Attachment,
		"attachment_count": count,
	})
}

// detectEvidenceType replica la allowlist de firmas del backend para que la
// tool rechace un media_type que no coincide antes de llamar a la API.
func detectEvidenceType(data []byte) (string, string) {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image", "image/jpeg"
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "image", "image/png"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image", "image/webp"
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		head := data
		if len(head) > 128 {
			head = head[:128]
		}
		if bytes.Contains(head, []byte("webm")) {
			return "video", "video/webm"
		}
		return "", ""
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		switch string(data[8:12]) {
		case "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1":
			return "image", "image/heic"
		case "qt  ":
			return "video", "video/quicktime"
		case "isom", "iso2", "iso4", "iso5", "iso6", "mp41", "mp42", "avc1", "dash", "M4V ", "mmp4":
			return "video", "video/mp4"
		}
	}
	return "", ""
}
