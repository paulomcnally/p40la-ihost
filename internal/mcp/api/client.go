// Package api implementa el cliente HTTP del servidor MCP contra p40la-ihost.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

const (
	maxResponseBytes = 1 << 20 // 1 MiB
	sourceHeader     = "X-P40LA-Source"
	sourceMCP        = "mcp"
)

// API es el contrato que consumen las tools. Permite usar una API falsa en tests.
type API interface {
	GetService(ctx context.Context, id int64) (*Service, error)
	ListBills(ctx context.Context, serviceID int64) ([]Bill, error)
	GenerateBills(ctx context.Context, serviceID int64, req GenerateRequest) (*GenerateResult, error)
	PayBills(ctx context.Context, serviceID int64, req PayRequest) (*PayResult, error)

	ListBankRefs(ctx context.Context) ([]BankRef, error)
	ListCurrencyRefs(ctx context.Context) ([]CurrencyRef, error)
	ListBankAccounts(ctx context.Context, bankID int64) ([]BankAccount, error)
	GetBankAccount(ctx context.Context, id int64) (*BankAccount, error)
	CreateBankAccount(ctx context.Context, bankID int64, input BankAccountInput) (*BankAccount, error)
	UpdateBankAccount(ctx context.Context, id int64, input BankAccountInput) (*BankAccount, error)
	ListCreditCards(ctx context.Context, bankID int64) ([]CreditCard, error)
	GetCreditCard(ctx context.Context, id int64) (*CreditCard, error)
	CreateCreditCard(ctx context.Context, bankID int64, input CreditCardInput) (*CreditCard, error)
	UpdateCreditCard(ctx context.Context, id int64, input CreditCardInput) (*CreditCard, error)
}

// Client es el cliente HTTP autenticado con la API key global del sistema.
type Client struct {
	baseURL *url.URL
	http    *http.Client
	apiKey  string
}

// New crea un cliente a partir de la configuración validada.
func New(cfg config.Config) (*Client, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("URL base inválida: %w", err)
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: cfg.Timeout},
		apiKey:  cfg.APIKey,
	}, nil
}

// Health consulta GET /health. Es útil para diagnóstico previo.
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.send(ctx, http.MethodGet, "/health", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}

// GetService valida que el servicio exista y devuelve sus datos básicos.
func (c *Client) GetService(ctx context.Context, id int64) (*Service, error) {
	var service Service
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/services/%d", id), nil, &service, nil); err != nil {
		return nil, err
	}
	return &service, nil
}

// ListBills devuelve las facturas activas de un servicio.
func (c *Client) ListBills(ctx context.Context, serviceID int64) ([]Bill, error) {
	var bills []Bill
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/services/%d/bills", serviceID), nil, &bills, nil); err != nil {
		return nil, err
	}
	return bills, nil
}

// GenerateBills llama al endpoint masivo de generación.
func (c *Client) GenerateBills(ctx context.Context, serviceID int64, req GenerateRequest) (*GenerateResult, error) {
	var result GenerateResult
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/api/services/%d/bills/generate", serviceID), req, &result, headers); err != nil {
		return nil, err
	}
	return &result, nil
}

// PayBills llama al endpoint masivo de pago.
func (c *Client) PayBills(ctx context.Context, serviceID int64, req PayRequest) (*PayResult, error) {
	var result PayResult
	headers := map[string]string{sourceHeader: sourceMCP}
	if err := c.doJSON(ctx, http.MethodPost, fmt.Sprintf("/api/services/%d/bills/pay", serviceID), req, &result, headers); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload, out any, headers map[string]string) error {
	resp, err := c.send(ctx, method, path, payload, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decodificar respuesta de la API: %w", err)
	}
	return nil
}

func (c *Client) send(ctx context.Context, method, path string, payload any, headers map[string]string) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("codificar request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.urlFor(path), body)
	if err != nil {
		return nil, fmt.Errorf("crear request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "p40la-mcp")
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	resp, err := c.http.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("timeout al llamar a la API")
		}
		return nil, fmt.Errorf("llamar a la API: %w", err)
	}
	return resp, nil
}

func (c *Client) urlFor(path string) string {
	base := *c.baseURL
	base.Path = strings.TrimRight(base.Path, "/") + path
	return base.String()
}

func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	payload := struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		return &APIError{Status: resp.StatusCode, Code: payload.Error, Message: payload.Message}
	}
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	code := "http_error"
	if resp.StatusCode == http.StatusUnauthorized {
		code = "auth_failed"
	}
	return &APIError{Status: resp.StatusCode, Code: code, Message: message}
}
