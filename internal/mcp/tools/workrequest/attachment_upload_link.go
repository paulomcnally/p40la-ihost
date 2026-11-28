package workrequest

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// attachmentUploadLinkInput es la entrada de attachment_upload_link.
type attachmentUploadLinkInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64 `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	Count         *int  `json:"count,omitempty" jsonschema:"Cantidad de archivos que admitirá el link (1..10); por defecto 10 y se acota al cupo libre"`
	TTLMinutes    *int  `json:"ttl_minutes,omitempty" jsonschema:"Vigencia del link en minutos (1..30); por defecto 10"`
	DryRun        *bool `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerAttachmentUploadLink(server *mcp.Server) {
	destructive := false
	notIdempotent := false
	openWorld := true
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_attachment_upload_link",
		Description: "Emite un link HTTP de corta vida para subir de 1 a 10 fotos o videos a un reporte " +
			"(JPEG/PNG/WebP/HEIC/MP4/QuickTime/WebM, hasta 25 MB por archivo). Devuelve upload_url, el token " +
			"(para el header Authorization), report_url y la alcanzabilidad. El agente sube los archivos con curl " +
			"y verifica con attachment_list. Solo reportes en estado reported. dry_run=true no emite token.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.attachmentUploadLink)
}

func (t *Toolset) attachmentUploadLink(ctx context.Context, _ *mcp.CallToolRequest, input attachmentUploadLinkInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	count := maxUploadFiles
	if input.Count != nil {
		count = *input.Count
	}
	if count < 1 || count > maxUploadFiles {
		return toolkit.ToolError("invalid_request", "count debe estar entre 1 y 10", "")
	}
	ttl := 10
	if input.TTLMinutes != nil {
		ttl = *input.TTLMinutes
	}
	if ttl < 1 || ttl > maxUploadTTLMinutes {
		return toolkit.ToolError("invalid_request", "ttl_minutes debe estar entre 1 y 30", "")
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
	remaining := maxAttachments - current.AttachmentCount
	if remaining <= 0 {
		return toolkit.ToolError("attachment_limit",
			"El reporte alcanzó el máximo de 10 adjuntos.",
			"Borrar un adjunto con home_work_request_attachment_delete o usar otro reporte.")
	}
	if count > remaining {
		count = remaining
	}
	reportURL := t.reportURL(input.HomeID, input.WorkRequestID)
	reachability := t.cfg.UploadReachability()
	base := map[string]any{
		"home_id":               input.HomeID,
		"work_request_id":       input.WorkRequestID,
		"max_bytes":             int64(maxAttachmentBytes),
		"max_request_bytes":     int64(count)*int64(maxAttachmentBytes) + (1 << 20),
		"max_files":             count,
		"accepted_mime":         append([]string(nil), acceptedMIME...),
		"attachments_remaining": remaining,
		"report_url":            reportURL,
		"reachability":          reachability,
		"curl_hint":             `curl -sS -X POST "<UPLOAD_URL>" -H "Authorization: Bearer <TOKEN>" -F "file=@/ruta/demo-foto-1.jpg"`,
	}
	if dryRun {
		base["status"] = "planned"
		base["dry_run"] = true
		base["upload_url"] = t.uploadURL("/api/work-request-uploads")
		base["token"] = nil
		return toolkit.JSONResult(base)
	}

	link, err := t.client.CreateHomeWorkRequestUploadLink(ctx, input.HomeID, input.WorkRequestID,
		api.HomeWorkRequestUploadLinkRequest{Count: count, TTLMinutes: ttl})
	if err != nil {
		return toolkit.APIError(err)
	}
	path := link.Path
	if path == "" {
		path = "/api/work-request-uploads"
	}
	base["status"] = "created"
	base["dry_run"] = false
	base["upload_url"] = t.uploadURL(path)
	base["token"] = link.Token
	base["expires_at"] = link.ExpiresAt
	base["max_bytes"] = link.MaxBytes
	base["max_request_bytes"] = link.MaxRequestBytes
	base["max_files"] = link.MaxFiles
	base["accepted_mime"] = link.AcceptedMIME
	base["attachments_remaining"] = link.RemainingSlots
	return toolkit.JSONResult(base)
}
