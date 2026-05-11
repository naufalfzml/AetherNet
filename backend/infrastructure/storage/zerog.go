package storage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient struct {
	BaseURL string
	Client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{BaseURL: baseURL}
}

func (c *HTTPClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	res, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("storage sidecar unhealthy: %s", res.Status)
	}
	return nil
}

func (c *HTTPClient) UploadJSON(ctx context.Context, value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return c.UploadBytes(ctx, "application/json", payload)
}

func (c *HTTPClient) UploadBytes(ctx context.Context, contentType string, payload []byte) (string, error) {
	body, err := json.Marshal(map[string]string{
		"contentType": contentType,
		"base64":      base64.StdEncoding.EncodeToString(payload),
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/upload", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("storage upload %s: %s", res.Status, string(raw))
	}
	var out struct {
		RootHash string `json:"rootHash"`
		Tx       string `json:"tx"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.RootHash == "" {
		return "", fmt.Errorf("storage upload missing rootHash")
	}
	return out.RootHash, nil
}

func (c *HTTPClient) Fetch(ctx context.Context, pointer string) ([]byte, error) {
	target := c.BaseURL + "/download/" + url.PathEscape(pointer)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		raw, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("storage download %s: %s", res.Status, string(raw))
	}
	return io.ReadAll(res.Body)
}

func (c *HTTPClient) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 120 * time.Second}
}
