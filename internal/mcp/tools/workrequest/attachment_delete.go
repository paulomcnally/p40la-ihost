package workrequest

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// attachmentDeleteInput es la entrada de home_work_request_attachment_delete.
type attachmentDeleteInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64 `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	AttachmentID  int64 `json:"attachment_id" jsonschema:"ID del adjunto (obligatorio)"`
	DryRun        *bool `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerAttachmentDelete(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_attachment_delete",
		Description: "Elimina un adjunto de un reporte y su archivo. dry_run viene activado por defecto. " +
			"Se permite en reportes reported y dismissed; se rechaza en converted. Si el adjunto ya no existe " +
			"devuelve status=unchanged sin error.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.attachmentDelete)
}

func (t *Toolset) attachmentDelete(ctx context.Context, _ *mcp.CallToolRequest, input attachmentDeleteInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	if input.AttachmentID <= 0 {
		return toolkit.ToolError("invalid_request", "attachment_id debe ser mayor que cero", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	current, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	if current.Status == "converted" {
		return toolkit.ToolError("invalid_status",
			"La evidencia de un reporte convertido no se puede borrar.",
			"Usá la web si necesitás administrar la evidencia histórica.")
	}
	if !hasAttachment(current.Attachments, input.AttachmentID) {
		return toolkit.JSONResult(map[string]any{
			"status":           "unchanged",
			"dry_run":          dryRun,
			"attachment_id":    input.AttachmentID,
			"deleted":          false,
			"attachment_count": current.AttachmentCount,
		})
	}
	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":           "planned",
			"dry_run":          true,
			"attachment_id":    input.AttachmentID,
			"deleted":          false,
			"attachment_count": current.AttachmentCount,
		})
	}

	if err := t.client.DeleteHomeWorkRequestAttachment(ctx, input.HomeID, input.WorkRequestID, input.AttachmentID); err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "attachment_not_found" {
			return toolkit.JSONResult(map[string]any{
				"status":           "unchanged",
				"dry_run":          false,
				"attachment_id":    input.AttachmentID,
				"deleted":          false,
				"attachment_count": current.AttachmentCount,
			})
		}
		return toolkit.APIError(err)
	}
	count := current.AttachmentCount - 1
	if count < 0 {
		count = 0
	}
	return toolkit.JSONResult(map[string]any{
		"status":           "deleted",
		"dry_run":          false,
		"attachment_id":    input.AttachmentID,
		"deleted":          true,
		"attachment_count": count,
	})
}

func hasAttachment(attachments []api.HomeWorkRequestAttachment, id int64) bool {
	for _, attachment := range attachments {
		if attachment.ID == id {
			return true
		}
	}
	return false
}
