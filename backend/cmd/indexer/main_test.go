package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type fakeStorageClient struct {
	payloads map[string][]byte
	err      error
}

func (f fakeStorageClient) Health(context.Context) error {
	return nil
}

func (f fakeStorageClient) UploadJSON(context.Context, any) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeStorageClient) UploadBytes(context.Context, string, []byte) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeStorageClient) Fetch(_ context.Context, pointer string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	payload, ok := f.payloads[pointer]
	if !ok {
		return nil, errors.New("missing payload")
	}
	return payload, nil
}

func TestHydrateMetadataParsesStorageJSON(t *testing.T) {
	metadata, err := hydrateMetadata(context.Background(), fakeStorageClient{
		payloads: map[string][]byte{
			"storage://meta": []byte(`{"prompt":"AetherNet is an agent","personalitySummary":"AetherNet is an agent for sovereign social systems."}`),
		},
	}, "storage://meta")
	if err != nil {
		t.Fatalf("hydrateMetadata failed: %v", err)
	}
	if metadata != (domain.AgentMetadata{
		MetadataPointer:    "storage://meta",
		Prompt:             "AetherNet is an agent",
		PersonalitySummary: "AetherNet is an agent for sovereign social systems.",
	}) {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestHydrateMetadataFallsBackToPromptWhenSummaryMissing(t *testing.T) {
	metadata, err := hydrateMetadata(context.Background(), fakeStorageClient{
		payloads: map[string][]byte{
			"storage://meta": []byte(`{"prompt":"MindHaven is an agent"}`),
		},
	}, "storage://meta")
	if err != nil {
		t.Fatalf("hydrateMetadata failed: %v", err)
	}
	if metadata.PersonalitySummary != "MindHaven is an agent" {
		t.Fatalf("expected prompt fallback, got %#v", metadata)
	}
}
