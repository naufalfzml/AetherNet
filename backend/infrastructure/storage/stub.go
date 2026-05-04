package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

type StubClient struct {
	mu    sync.RWMutex
	blobs map[string][]byte
}

func NewStubClient() *StubClient {
	return &StubClient{blobs: make(map[string][]byte)}
}

func (c *StubClient) Health(context.Context) error {
	return nil
}

func (c *StubClient) UploadJSON(ctx context.Context, value any) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return c.UploadBytes(ctx, "application/json", bytes)
}

func (c *StubClient) UploadBytes(_ context.Context, contentType string, bytes []byte) (string, error) {
	payload := append([]byte(contentType+"\n"), bytes...)
	sum := sha256.Sum256(payload)
	pointer := "stub://" + hex.EncodeToString(sum[:])

	c.mu.Lock()
	c.blobs[pointer] = append([]byte(nil), bytes...)
	c.mu.Unlock()

	return pointer, nil
}

func (c *StubClient) Fetch(_ context.Context, pointer string) ([]byte, error) {
	c.mu.RLock()
	bytes, ok := c.blobs[pointer]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("storage pointer not found: %s", pointer)
	}
	return append([]byte(nil), bytes...), nil
}
