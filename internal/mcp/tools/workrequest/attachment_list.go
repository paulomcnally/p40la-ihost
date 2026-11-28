package workrequest

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// attachmentListInput es la entrada de home_work_request_attachment_list.
type attachmentListInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64 `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
}

func (t *Toolset) registerAttachmentList(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_attachment_list",
		Description: "Lista la evidencia (fotos y videos) de un reporte con sus metadatos, sin binarios. " +
			"Devuelve el cupo usado, el máximo de 10 adjuntos, el tamaño máximo por archivo y el sha256 de cada uno. " +
			"Es la vía canónica para verificar que una carga terminó. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.attachmentList)
}

func (t *Toolset) attachmentList(ctx context.Context, _ *mcp.CallToolRequest, input attachmentListInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	item, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	attachments := item.Attachments
	if attachments == nil {
		attachments = []api.HomeWorkRequestAttachment{}
	}
	remaining := maxAttachments - item.AttachmentCount
	if remaining < 0 {
		remaining = 0
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":          input.HomeID,
		"work_request_id":  input.WorkRequestID,
		"attachment_count": item.AttachmentCount,
		"max_attachments":  maxAttachments,
		"max_bytes":        maxAttachmentBytes,
		"remaining_slots":  remaining,
		"attachments":      attachments,
	})
}
