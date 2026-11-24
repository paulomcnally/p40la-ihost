package banking

import (
	"errors"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// resolveLimit valida el límite de resultados de las búsquedas.
func resolveLimit(limit *int) (int, *mcp.CallToolResult, any) {
	value := 10
	if limit != nil {
		value = *limit
	}
	if value < 1 || value > 50 {
		result, output, _ := toolError("invalid_value", "limit debe estar entre 1 y 50.", "")
		return 0, result, output
	}
	return value, nil, nil
}

// isNotFound indica si el error de la API es un 404.
func isNotFound(err error) bool {
	var apiErr *api.APIError
	return errors.As(err, &apiErr) && apiErr.Status == 404
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
