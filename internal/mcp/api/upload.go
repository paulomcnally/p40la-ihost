package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// postMultipart envía una sola parte "file" al backend autenticado por API key.
// No escribe en disco: los bytes ya vienen decodificados en memoria.
func (c *Client) postMultipart(ctx context.Context, path, fileName string, data []byte, out any) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return fmt.Errorf("crear multipart: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("escribir multipart: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("cerrar multipart: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.urlFor(path), &body)
	if err != nil {
		return fmt.Errorf("crear request multipart: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "p40la-mcp")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set(sourceHeader, sourceMCP)
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("llamar a la API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(out); err != nil {
		return fmt.Errorf("decodificar respuesta de la API: %w", err)
	}
	return nil
}
