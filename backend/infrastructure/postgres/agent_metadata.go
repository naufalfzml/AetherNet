package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AgentMetadataRepository struct {
	DB *sql.DB
}

func (r AgentMetadataRepository) UpsertMetadata(ctx context.Context, metadata domain.AgentMetadata) error {
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now().UTC()
	}
	if metadata.UpdatedAt.IsZero() {
		metadata.UpdatedAt = metadata.CreatedAt
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO agent_metadata (metadata_pointer, prompt, personality_summary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (metadata_pointer) DO UPDATE SET
			prompt = EXCLUDED.prompt,
			personality_summary = EXCLUDED.personality_summary,
			updated_at = EXCLUDED.updated_at
	`, metadata.MetadataPointer, metadata.Prompt, metadata.PersonalitySummary, metadata.CreatedAt, metadata.UpdatedAt)
	return err
}

func (r AgentMetadataRepository) GetMetadata(ctx context.Context, metadataPointer string) (domain.AgentMetadata, error) {
	var metadata domain.AgentMetadata
	err := r.DB.QueryRowContext(ctx, `
		SELECT metadata_pointer, prompt, personality_summary, created_at, updated_at
		FROM agent_metadata
		WHERE metadata_pointer = $1
	`, metadataPointer).Scan(
		&metadata.MetadataPointer,
		&metadata.Prompt,
		&metadata.PersonalitySummary,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)
	if err != nil {
		return domain.AgentMetadata{}, err
	}
	return metadata, nil
}
