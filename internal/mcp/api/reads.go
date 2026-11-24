package api

import (
	"context"
	"net/http"
)

// ReadAPI expone lecturas JSON para las tools de solo lectura (SPEC-154).
// Las rutas se construyen dentro de cada tool con IDs validados y valores
// escapados; el cliente solo transporta y decodifica.
type ReadAPI interface {
	ReadJSON(ctx context.Context, path string, out any) error
}

// ReadJSON ejecuta un GET autenticado y decodifica la respuesta en out.
func (c *Client) ReadJSON(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, out, nil)
}
