package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AgentRepository struct {
	DB *sql.DB
}

func (r AgentRepository) ListAgents(ctx context.Context, limit int) ([]domain.Agent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT agent_id, token_id::text, owner_address, treasury_address, metadata_pointer,
			COALESCE(NULLIF(agent_cache.personality_summary, ''), NULLIF(agent_metadata.personality_summary, ''), agent_cache.personality_summary),
			agent_cache.updated_at
		FROM agent_cache
		LEFT JOIN agent_metadata USING (metadata_pointer)
		ORDER BY agent_cache.updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := make([]domain.Agent, 0)
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (r AgentRepository) GetAgentByID(ctx context.Context, agentID string) (domain.Agent, error) {
	return r.getAgent(ctx, `
		SELECT agent_id, token_id::text, owner_address, treasury_address, metadata_pointer,
			COALESCE(NULLIF(agent_cache.personality_summary, ''), NULLIF(agent_metadata.personality_summary, ''), agent_cache.personality_summary),
			agent_cache.updated_at
		FROM agent_cache
		LEFT JOIN agent_metadata USING (metadata_pointer)
		WHERE agent_id = $1
	`, agentID)
}

func (r AgentRepository) GetAgentByTokenID(ctx context.Context, tokenID string) (domain.Agent, error) {
	return r.getAgent(ctx, `
		SELECT agent_id, token_id::text, owner_address, treasury_address, metadata_pointer,
			COALESCE(NULLIF(agent_cache.personality_summary, ''), NULLIF(agent_metadata.personality_summary, ''), agent_cache.personality_summary),
			agent_cache.updated_at
		FROM agent_cache
		LEFT JOIN agent_metadata USING (metadata_pointer)
		WHERE token_id = $1::numeric
	`, tokenID)
}

func (r AgentRepository) GetAgentByAddress(ctx context.Context, agentAddress string) (domain.Agent, error) {
	return r.getAgent(ctx, `
		SELECT agent_id, token_id::text, owner_address, treasury_address, metadata_pointer,
			COALESCE(NULLIF(agent_cache.personality_summary, ''), NULLIF(agent_metadata.personality_summary, ''), agent_cache.personality_summary),
			agent_cache.updated_at
		FROM agent_cache
		LEFT JOIN agent_metadata USING (metadata_pointer)
		WHERE lower(treasury_address) = lower($1)
	`, strings.TrimSpace(agentAddress))
}

func (r AgentRepository) UpsertAgent(ctx context.Context, agent domain.Agent) error {
	agent = prepareAgentForUpsert(agent)

	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO agent_cache (
			agent_id, token_id, owner_address, treasury_address, metadata_pointer,
			personality_summary, updated_at
		)
		VALUES ($1, $2::numeric, $3, $4, $5, $6, $7)
		ON CONFLICT (agent_id) DO UPDATE SET
			token_id = EXCLUDED.token_id,
			owner_address = EXCLUDED.owner_address,
			treasury_address = EXCLUDED.treasury_address,
			metadata_pointer = EXCLUDED.metadata_pointer,
			personality_summary = EXCLUDED.personality_summary,
			updated_at = EXCLUDED.updated_at
	`, agent.ID, agent.TokenID, agent.OwnerAddress, agent.TreasuryAddress, agent.MetadataPointer,
		agent.PersonalitySummary, agent.UpdatedAt)
	return err
}

func prepareAgentForUpsert(agent domain.Agent) domain.Agent {
	if agent.AgentAddress == "" {
		agent.AgentAddress = agent.TreasuryAddress
	}
	if agent.TreasuryAddress == "" {
		agent.TreasuryAddress = agent.AgentAddress
	}
	if agent.ID == "" {
		agent.ID = agent.AgentAddress
	}
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = time.Now().UTC()
	}
	return agent
}

func (r AgentRepository) getAgent(ctx context.Context, query string, arg string) (domain.Agent, error) {
	row := r.DB.QueryRowContext(ctx, query, arg)
	agent, err := scanAgent(row)
	if err != nil {
		return domain.Agent{}, err
	}
	return agent, nil
}

type agentScanner interface {
	Scan(dest ...any) error
}

func scanAgent(scanner agentScanner) (domain.Agent, error) {
	var agent domain.Agent
	err := scanner.Scan(
		&agent.ID,
		&agent.TokenID,
		&agent.OwnerAddress,
		&agent.TreasuryAddress,
		&agent.MetadataPointer,
		&agent.PersonalitySummary,
		&agent.UpdatedAt,
	)
	if err != nil {
		return domain.Agent{}, err
	}
	agent.AgentAddress = agent.TreasuryAddress
	if agent.PersonalitySummary == "" {
		agent.PersonalitySummary = "Indexed agent " + agent.TokenID
	}
	return agent, nil
}

var _ = sql.ErrNoRows
