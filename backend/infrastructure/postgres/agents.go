package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AgentRepository struct {
	DB *sql.DB
}

func (r AgentRepository) ListAgents(ctx context.Context, limit int, offset int) ([]domain.Agent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT ac.agent_id, ac.token_id::text, ac.owner_address, ac.treasury_address, ac.metadata_pointer,
			COALESCE(NULLIF(am.personality_summary, ''), NULLIF(ac.personality_summary, ''), ''),
			ac.updated_at,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND payload->>'targetAgentId' = ac.agent_id) as followers,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND agent_id = ac.agent_id) as following
		FROM agent_cache ac
		LEFT JOIN agent_metadata am ON ac.metadata_pointer = am.metadata_pointer
		ORDER BY ac.updated_at DESC, ac.agent_id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
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

func (r AgentRepository) SearchAgents(ctx context.Context, query string, kind string, sortBy string, limit int, offset int) ([]domain.Agent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	sqlQuery := `
		SELECT ac.agent_id, ac.token_id::text, ac.owner_address, ac.treasury_address, ac.metadata_pointer,
			COALESCE(NULLIF(am.personality_summary, ''), NULLIF(ac.personality_summary, ''), ''),
			ac.updated_at,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND payload->>'targetAgentId' = ac.agent_id) as followers,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND agent_id = ac.agent_id) as following
		FROM agent_cache ac
		LEFT JOIN agent_metadata am ON ac.metadata_pointer = am.metadata_pointer
		WHERE (ac.agent_id ILIKE $1 OR ac.personality_summary ILIKE $1 OR am.prompt ILIKE $1)
	`
	args := []any{"%" + query + "%"}

	// Note: for this hackathon we assume kind is always 'native' for this table
	// but we filter if provided
	if kind != "" && kind != "all" {
		// handle filter
	}

	switch sortBy {
	case "popularity":
		sqlQuery += " ORDER BY followers DESC, ac.updated_at DESC"
	case "alphabetical":
		sqlQuery += " ORDER BY ac.agent_id ASC"
	default:
		sqlQuery += " ORDER BY ac.updated_at DESC"
	}

	sqlQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	rows, err := r.DB.QueryContext(ctx, sqlQuery, append(args, limit, offset)...)
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
		SELECT ac.agent_id, ac.token_id::text, ac.owner_address, ac.treasury_address, ac.metadata_pointer,
			COALESCE(NULLIF(am.personality_summary, ''), NULLIF(ac.personality_summary, ''), ''),
			ac.updated_at,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND payload->>'targetAgentId' = ac.agent_id) as followers,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND agent_id = ac.agent_id) as following
		FROM agent_cache ac
		LEFT JOIN agent_metadata am ON ac.metadata_pointer = am.metadata_pointer
		WHERE ac.agent_id = $1
	`, agentID)
}

func (r AgentRepository) GetAgentByTokenID(ctx context.Context, tokenID string) (domain.Agent, error) {
	return r.getAgent(ctx, `
		SELECT ac.agent_id, ac.token_id::text, ac.owner_address, ac.treasury_address, ac.metadata_pointer,
			COALESCE(NULLIF(am.personality_summary, ''), NULLIF(ac.personality_summary, ''), ''),
			ac.updated_at,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND payload->>'targetAgentId' = ac.agent_id) as followers,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND agent_id = ac.agent_id) as following
		FROM agent_cache ac
		LEFT JOIN agent_metadata am ON ac.metadata_pointer = am.metadata_pointer
		WHERE ac.token_id = $1::numeric
	`, tokenID)
}

func (r AgentRepository) GetAgentByAddress(ctx context.Context, agentAddress string) (domain.Agent, error) {
	return r.getAgent(ctx, `
		SELECT ac.agent_id, ac.token_id::text, ac.owner_address, ac.treasury_address, ac.metadata_pointer,
			COALESCE(NULLIF(am.personality_summary, ''), NULLIF(ac.personality_summary, ''), ''),
			ac.updated_at,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND payload->>'targetAgentId' = ac.agent_id) as followers,
			(SELECT COUNT(*) FROM social_events WHERE type = 'follow' AND agent_id = ac.agent_id) as following
		FROM agent_cache ac
		LEFT JOIN agent_metadata am ON ac.metadata_pointer = am.metadata_pointer
		WHERE lower(ac.treasury_address) = lower($1)
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
		&agent.Followers,
		&agent.Following,
	)
	if err != nil {
		return domain.Agent{}, err
	}
	agent.Kind = "native"
	agent.AgentAddress = agent.TreasuryAddress
	if agent.PersonalitySummary == "" {
		agent.PersonalitySummary = ""
	}
	return agent, nil
}

var _ = sql.ErrNoRows
