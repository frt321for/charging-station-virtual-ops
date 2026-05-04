package simulator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIError describes a backend API failure.
type APIError struct {
	Method  string
	Path    string
	Status  int
	Code    int
	Message string
	Details string
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s %s failed: http=%d code=%d message=%s details=%s", e.Method, e.Path, e.Status, e.Code, e.Message, e.Details)
}

// Client calls the charging operations backend API.
type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	authorization string
}

// NewClient creates an API client.
func NewClient(apiBase string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(apiBase, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse api base: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("api-base must be absolute: %s", apiBase)
	}
	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// SetAuthorization forwards an already authenticated browser request to protected API calls.
func (c *Client) SetAuthorization(authorization string) {
	c.authorization = strings.TrimSpace(authorization)
}

func (c *Client) Register(ctx context.Context, req registerRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v1/gateway/chargers/register", req, nil)
}

func (c *Client) Heartbeat(ctx context.Context, chargerCode string, req heartbeatRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v1/gateway/chargers/"+url.PathEscape(chargerCode)+"/heartbeat", req, nil)
}

func (c *Client) Offline(ctx context.Context, chargerCode string) error {
	return c.do(ctx, http.MethodPost, "/api/v1/gateway/chargers/"+url.PathEscape(chargerCode)+"/offline", nil, nil)
}

func (c *Client) Status(ctx context.Context, chargerCode string, req statusRequest) error {
	return c.do(ctx, http.MethodPost, "/api/v1/gateway/chargers/"+url.PathEscape(chargerCode)+"/status", req, nil)
}

func (c *Client) Reservation(ctx context.Context, req reservationRequest) (sessionData, error) {
	var data sessionData
	err := c.do(ctx, http.MethodPost, "/api/v1/reservations", req, &data)
	return data, err
}

func (c *Client) Command(ctx context.Context, sessionNo string, commandType string, req commandRequest) (commandData, error) {
	var data commandData
	path := "/api/v1/sessions/" + url.PathEscape(sessionNo) + "/commands/" + commandPath(commandType)
	err := c.do(ctx, http.MethodPost, path, req, &data)
	return data, err
}

func (c *Client) Receipt(ctx context.Context, chargerCode string, req receiptRequest) error {
	path := "/api/v1/gateway/chargers/" + url.PathEscape(chargerCode) + "/command-receipts"
	return c.do(ctx, http.MethodPost, path, req, nil)
}

func (c *Client) MeterValue(ctx context.Context, chargerCode string, req meterValueRequest) error {
	path := "/api/v1/gateway/chargers/" + url.PathEscape(chargerCode) + "/meter-values"
	return c.do(ctx, http.MethodPost, path, req, nil)
}

func (c *Client) Alarm(ctx context.Context, chargerCode string, req alarmRequest) error {
	path := "/api/v1/gateway/chargers/" + url.PathEscape(chargerCode) + "/alarms"
	return c.do(ctx, http.MethodPost, path, req, nil)
}

func (c *Client) do(ctx context.Context, method string, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.resolve(path), reader)
	if err != nil {
		return fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Trace-Id", "sim-"+time.Now().UTC().Format("20060102150405.000000000"))
	if c.authorization != "" {
		req.Header.Set("Authorization", c.authorization)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Details string          `json:"details"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 || envelope.Code != 0 {
		return APIError{Method: method, Path: path, Status: resp.StatusCode, Code: envelope.Code, Message: envelope.Message, Details: envelope.Details}
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("decode %s %s data: %w", method, path, err)
		}
	}
	return nil
}

func (c *Client) resolve(path string) string {
	copyValue := *c.baseURL
	copyValue.Path = strings.TrimRight(copyValue.Path, "/") + path
	return copyValue.String()
}

func commandPath(commandType string) string {
	if commandType == "limit_power" {
		return "limit-power"
	}
	return commandType
}
