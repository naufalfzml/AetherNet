package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type fakeAgentRepo struct {
	agents    []domain.Agent
	upserts   []domain.Agent
	listCalls int
}

func (f *fakeAgentRepo) ListAgents(_ context.Context, limit int, offset int) ([]domain.Agent, error) {
	f.listCalls++
	if offset >= len(f.agents) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.agents) {
		end = len(f.agents)
	}
	out := make([]domain.Agent, 0, end-offset)
	out = append(out, f.agents[offset:end]...)
	return out, nil
}

func (f *fakeAgentRepo) UpsertAgent(_ context.Context, agent domain.Agent) error {
	f.upserts = append(f.upserts, agent)
	return nil
}

type fakeMetadataRepo struct {
	items []domain.AgentMetadata
}

func (f *fakeMetadataRepo) UpsertMetadata(_ context.Context, metadata domain.AgentMetadata) error {
	f.items = append(f.items, metadata)
	return nil
}

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

func TestBackfillMissingAgentMetadataRecoversOnlyAgentsMissingSummary(t *testing.T) {
	agentRepo := &fakeAgentRepo{
		agents: []domain.Agent{
			{
				ID:              "0x1",
				TokenID:         "1",
				MetadataPointer: "storage://one",
			},
			{
				ID:                 "0x2",
				TokenID:            "2",
				MetadataPointer:    "storage://two",
				PersonalitySummary: "Already Present",
			},
		},
	}
	metadataRepo := &fakeMetadataRepo{}
	storageClient := fakeStorageClient{
		payloads: map[string][]byte{
			"storage://one": []byte(`{"prompt":"Futurist is an agent","personalitySummary":"Futurist is an agent tracking paradigm shifts."}`),
		},
	}

	if err := backfillMissingAgentMetadata(context.Background(), agentRepo, metadataRepo, storageClient); err != nil {
		t.Fatalf("backfillMissingAgentMetadata failed: %v", err)
	}
	if len(metadataRepo.items) != 1 {
		t.Fatalf("expected 1 metadata upsert, got %d", len(metadataRepo.items))
	}
	if len(agentRepo.upserts) != 1 {
		t.Fatalf("expected 1 agent cache refresh, got %d", len(agentRepo.upserts))
	}
	if agentRepo.upserts[0].PersonalitySummary != "Futurist is an agent tracking paradigm shifts." {
		t.Fatalf("unexpected backfilled summary: %#v", agentRepo.upserts[0])
	}
}
