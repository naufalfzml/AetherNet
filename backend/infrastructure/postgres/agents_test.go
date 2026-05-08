package postgres

import (
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

func TestPrepareAgentForUpsertUsesAgentAddressAsPublicID(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := prepareAgentForUpsert(domain.Agent{
		TokenID:         "1",
		OwnerAddress:    "0xowner",
		AgentAddress:    "0xagent",
		MetadataPointer: "stub://agent",
		UpdatedAt:       now,
	})

	if agent.ID != "0xagent" {
		t.Fatalf("expected ID to default to agent address, got %q", agent.ID)
	}
	if agent.TreasuryAddress != "0xagent" {
		t.Fatalf("expected storage address to mirror agent address, got %q", agent.TreasuryAddress)
	}
	if !agent.UpdatedAt.Equal(now) {
		t.Fatalf("expected explicit updated_at to be preserved")
	}
}

func TestPrepareAgentForUpsertMirrorsExistingStorageAddress(t *testing.T) {
	agent := prepareAgentForUpsert(domain.Agent{
		TokenID:         "1",
		OwnerAddress:    "0xowner",
		TreasuryAddress: "0xagent",
		MetadataPointer: "stub://agent",
	})

	if agent.AgentAddress != "0xagent" {
		t.Fatalf("expected public agent address from storage address, got %q", agent.AgentAddress)
	}
	if agent.ID != "0xagent" {
		t.Fatalf("expected ID to default to agent address, got %q", agent.ID)
	}
	if agent.UpdatedAt.IsZero() {
		t.Fatalf("expected updated_at to default")
	}
}
