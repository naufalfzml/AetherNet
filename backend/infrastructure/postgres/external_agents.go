package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type ExternalAgentRepository struct {
	DB *sql.DB
}

func (r ExternalAgentRepository) ListExternalAgents(ctx context.Context, limit int) ([]domain.ExternalAgent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT external_agent_id, kind, status, display_name, handle, owner_wallet_address,
			description, personality_summary, metadata_pointer, linked_native_agent_id,
			minted_token_id, wallet_verified_at, created_at, updated_at
		FROM external_agents
		ORDER BY updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := make([]domain.ExternalAgent, 0)
	for rows.Next() {
		agent, err := scanExternalAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (r ExternalAgentRepository) CreateExternalAgent(ctx context.Context, agent domain.ExternalAgent) (domain.ExternalAgent, error) {
	agent = prepareExternalAgent(agent)

	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO external_agents (
			external_agent_id, kind, status, display_name, handle, owner_wallet_address,
			description, personality_summary, metadata_pointer, linked_native_agent_id,
			minted_token_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, agent.ID, agent.Kind, agent.Status, agent.DisplayName, agent.Handle, agent.OwnerWalletAddress,
		agent.Description, agent.PersonalitySummary, agent.MetadataPointer, agent.LinkedNativeAgentID,
		agent.MintedTokenID, agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return domain.ExternalAgent{}, err
	}
	return agent, nil
}

func (r ExternalAgentRepository) GetExternalAgentByID(ctx context.Context, agentID string) (domain.ExternalAgent, error) {
	return r.getExternalAgent(ctx, `
		SELECT external_agent_id, kind, status, display_name, handle, owner_wallet_address,
			description, personality_summary, metadata_pointer, linked_native_agent_id,
			minted_token_id, wallet_verified_at, created_at, updated_at
		FROM external_agents
		WHERE external_agent_id = $1
	`, agentID)
}

func (r ExternalAgentRepository) GetExternalAgentByHandle(ctx context.Context, handle string) (domain.ExternalAgent, error) {
	return r.getExternalAgent(ctx, `
		SELECT external_agent_id, kind, status, display_name, handle, owner_wallet_address,
			description, personality_summary, metadata_pointer, linked_native_agent_id,
			minted_token_id, wallet_verified_at, created_at, updated_at
		FROM external_agents
		WHERE lower(handle) = lower($1)
	`, handle)
}

func (r ExternalAgentRepository) GetExternalAgentByAPIKeyHash(ctx context.Context, apiKeyHash string) (domain.ExternalAgent, error) {
	return r.getExternalAgent(ctx, `
		SELECT external_agent_id, kind, status, display_name, handle, owner_wallet_address,
			description, personality_summary, metadata_pointer, linked_native_agent_id,
			minted_token_id, wallet_verified_at, created_at, updated_at
		FROM external_agents
		WHERE api_key_hash = $1 AND api_key_hash <> ''
	`, apiKeyHash)
}

func (r ExternalAgentRepository) UpdateExternalAgent(ctx context.Context, agent domain.ExternalAgent) (domain.ExternalAgent, error) {
	agent = prepareExternalAgent(agent)

	_, err := r.DB.ExecContext(ctx, `
		UPDATE external_agents
		SET display_name = $2,
			handle = $3,
			description = $4,
			personality_summary = $5,
			metadata_pointer = $6,
			linked_native_agent_id = $7,
			minted_token_id = $8,
			updated_at = $9
		WHERE external_agent_id = $1
	`, agent.ID, agent.DisplayName, agent.Handle, agent.Description, agent.PersonalitySummary,
		agent.MetadataPointer, agent.LinkedNativeAgentID, agent.MintedTokenID, agent.UpdatedAt)
	if err != nil {
		return domain.ExternalAgent{}, err
	}
	return r.GetExternalAgentByID(ctx, agent.ID)
}

func (r ExternalAgentRepository) SaveExternalAgentAPIKey(ctx context.Context, agentID string, apiKeyHash string, verifiedAt time.Time) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE external_agents
		SET api_key_hash = $2,
			status = 'active',
			wallet_verified_at = $3,
			updated_at = $3
		WHERE external_agent_id = $1
	`, agentID, apiKeyHash, verifiedAt)
	return err
}

func (r ExternalAgentRepository) CreateAuthChallenge(ctx context.Context, challenge domain.ExternalAgentAuthChallenge) (domain.ExternalAgentAuthChallenge, error) {
	if challenge.CreatedAt.IsZero() {
		challenge.CreatedAt = time.Now().UTC()
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO external_agent_auth_challenges (
			challenge_id, external_agent_id, wallet_address, challenge_text, expires_at, consumed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, challenge.ID, challenge.AgentID, challenge.WalletAddress, challenge.ChallengeText, challenge.ExpiresAt, nullTime(challenge.ConsumedAt), challenge.CreatedAt)
	if err != nil {
		return domain.ExternalAgentAuthChallenge{}, err
	}
	return challenge, nil
}

func (r ExternalAgentRepository) GetAuthChallenge(ctx context.Context, challengeID string) (domain.ExternalAgentAuthChallenge, error) {
	var challenge domain.ExternalAgentAuthChallenge
	var consumedAt sql.NullTime
	err := r.DB.QueryRowContext(ctx, `
		SELECT challenge_id, external_agent_id, wallet_address, challenge_text, expires_at, consumed_at, created_at
		FROM external_agent_auth_challenges
		WHERE challenge_id = $1
	`, challengeID).Scan(
		&challenge.ID,
		&challenge.AgentID,
		&challenge.WalletAddress,
		&challenge.ChallengeText,
		&challenge.ExpiresAt,
		&consumedAt,
		&challenge.CreatedAt,
	)
	if err != nil {
		return domain.ExternalAgentAuthChallenge{}, err
	}
	if consumedAt.Valid {
		challenge.ConsumedAt = consumedAt.Time
	}
	return challenge, nil
}

func (r ExternalAgentRepository) ConsumeAuthChallenge(ctx context.Context, challengeID string, consumedAt time.Time) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE external_agent_auth_challenges
		SET consumed_at = $2
		WHERE challenge_id = $1
	`, challengeID, consumedAt)
	return err
}

func (r ExternalAgentRepository) getExternalAgent(ctx context.Context, query string, arg string) (domain.ExternalAgent, error) {
	row := r.DB.QueryRowContext(ctx, query, arg)
	return scanExternalAgent(row)
}

func prepareExternalAgent(agent domain.ExternalAgent) domain.ExternalAgent {
	agent.Kind = "external"
	if agent.Status == "" {
		agent.Status = "pending_verification"
	}
	agent.Handle = strings.ToLower(strings.TrimSpace(agent.Handle))
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = time.Now().UTC()
	}
	agent.UpdatedAt = time.Now().UTC()
	return agent
}

type externalAgentScanner interface {
	Scan(dest ...any) error
}

func scanExternalAgent(scanner externalAgentScanner) (domain.ExternalAgent, error) {
	var agent domain.ExternalAgent
	var verifiedAt sql.NullTime
	err := scanner.Scan(
		&agent.ID,
		&agent.Kind,
		&agent.Status,
		&agent.DisplayName,
		&agent.Handle,
		&agent.OwnerWalletAddress,
		&agent.Description,
		&agent.PersonalitySummary,
		&agent.MetadataPointer,
		&agent.LinkedNativeAgentID,
		&agent.MintedTokenID,
		&verifiedAt,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)
	if err != nil {
		return domain.ExternalAgent{}, err
	}
	if verifiedAt.Valid {
		agent.WalletVerifiedAt = &verifiedAt.Time
	}
	return agent, nil
}

func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
