package da

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
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
		return fmt.Errorf("da sidecar unhealthy: %s", res.Status)
	}
	return nil
}

func (c *HTTPClient) Publish(ctx context.Context, event domain.SocialEvent) (string, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/publish", bytes.NewReader(body))
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
		return "", fmt.Errorf("da publish %s: %s", res.Status, string(raw))
	}

	var out struct {
		BlobID string `json:"blobId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.BlobID == "" {
		return "", fmt.Errorf("da publish missing blobId")
	}
	return out.BlobID, nil
}

// Subscribe is a mock for now, ideally it would use WebSockets or long-polling to the sidecar
func (c *HTTPClient) Subscribe(ctx context.Context, eventTypes []string) (<-chan domain.SocialEvent, error) {
	ch := make(chan domain.SocialEvent)
	// Mock implementation: just return an empty channel that never receives to avoid blocking errors
	// Real implementation would poll the sidecar or DA retriever
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func (c *HTTPClient) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 120 * time.Second} // Dispersal can take a while
}
