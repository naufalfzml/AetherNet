package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type SocialEventRepository struct {
	DB *sql.DB
}

func (r SocialEventRepository) UpsertSocialEvent(ctx context.Context, event domain.SocialEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO social_events (blob_id, type, agent_id, payload, sig, event_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (blob_id) DO UPDATE SET
			type = EXCLUDED.type,
			agent_id = EXCLUDED.agent_id,
			payload = EXCLUDED.payload,
			sig = EXCLUDED.sig,
			event_timestamp = EXCLUDED.event_timestamp
	`, event.BlobID, event.Type, event.AgentID, payload, event.Sig, event.Timestamp)
	return err
}

func (r SocialEventRepository) ListTimeline(context.Context, int) ([]domain.SocialEvent, error) {
	return nil, nil
}

func (r SocialEventRepository) ListAgentPosts(context.Context, string, int) ([]domain.Post, error) {
	return nil, nil
}
