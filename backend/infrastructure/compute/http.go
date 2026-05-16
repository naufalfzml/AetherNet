package compute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aethernet-0g/aethernet/backend/usecase"
)

type HTTPClient struct {
	BaseURL     string
	Client      *http.Client
	RequireReal bool
}

func (c HTTPClient) Health(ctx context.Context) error {
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
		return fmt.Errorf("compute sidecar unhealthy: %s", res.Status)
	}
	var out struct {
		StubMode  bool   `json:"stubMode"`
		ImageMode string `json:"imageMode"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return err
	}
	if c.RequireReal && out.StubMode {
		return fmt.Errorf("compute sidecar is in stub mode")
	}
	return nil
}

func (c HTTPClient) RunLLM(ctx context.Context, llmReq usecase.LLMRequest) (usecase.LLMResponse, error) {
	body, err := json.Marshal(llmReq)
	if err != nil {
		return usecase.LLMResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/infer/llm", bytes.NewReader(body))
	if err != nil {
		return usecase.LLMResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client().Do(req)
	if err != nil {
		return usecase.LLMResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return usecase.LLMResponse{}, fmt.Errorf("compute sidecar returned %s", res.Status)
	}

	var out usecase.LLMResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return usecase.LLMResponse{}, err
	}
	return out, nil
}

func (c HTTPClient) RunImageGen(ctx context.Context, imgReq usecase.ImageRequest) (usecase.ImageResponse, error) {
	body, err := json.Marshal(imgReq)
	if err != nil {
		return usecase.ImageResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/infer/image", bytes.NewReader(body))
	if err != nil {
		return usecase.ImageResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.imageClient().Do(req)
	if err != nil {
		return usecase.ImageResponse{}, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return usecase.ImageResponse{}, fmt.Errorf("compute sidecar image returned %s", res.Status)
	}

	var out usecase.ImageResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return usecase.ImageResponse{}, err
	}
	return out, nil
}

func (c HTTPClient) imageClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 300 * time.Second}
}

func (c HTTPClient) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}
