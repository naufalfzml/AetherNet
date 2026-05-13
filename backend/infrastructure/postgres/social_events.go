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

func (r SocialEventRepository) ListTimeline(ctx context.Context, limit int) ([]domain.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			p.blob_id,
			p.type,
			p.agent_id,
			p.payload,
			p.sig,
			p.event_timestamp,
			COUNT(*) FILTER (WHERE a.type = 'like') AS likes,
			COUNT(*) FILTER (WHERE a.type = 'comment') AS comments,
			COUNT(*) FILTER (WHERE a.type = 'repost') AS reposts
		FROM social_events p
		LEFT JOIN social_events a
			ON a.agent_id = p.agent_id
			AND a.payload->>'postId' = p.blob_id
			AND a.type IN ('like', 'comment', 'repost')
		WHERE p.type = 'post'
		GROUP BY p.blob_id, p.type, p.agent_id, p.payload, p.sig, p.event_timestamp
		ORDER BY p.event_timestamp DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (r SocialEventRepository) ListAgentPosts(ctx context.Context, agentID string, limit int) ([]domain.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT
			p.blob_id,
			p.type,
			p.agent_id,
			p.payload,
			p.sig,
			p.event_timestamp,
			COUNT(*) FILTER (WHERE a.type = 'like') AS likes,
			COUNT(*) FILTER (WHERE a.type = 'comment') AS comments,
			COUNT(*) FILTER (WHERE a.type = 'repost') AS reposts
		FROM social_events p
		LEFT JOIN social_events a
			ON a.agent_id = p.agent_id
			AND a.payload->>'postId' = p.blob_id
			AND a.type IN ('like', 'comment', 'repost')
		WHERE p.type = 'post' AND p.agent_id = $1
		GROUP BY p.blob_id, p.type, p.agent_id, p.payload, p.sig, p.event_timestamp
		ORDER BY p.event_timestamp DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]domain.Post, 0)
	for rows.Next() {
		post, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (r SocialEventRepository) ListAgentSocialEvents(ctx context.Context, agentID string, limit int) ([]domain.SocialEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT blob_id, type, agent_id, payload, sig, event_timestamp
		FROM social_events
		WHERE agent_id = $1
		ORDER BY event_timestamp DESC
		LIMIT $2
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.SocialEvent, 0)
	for rows.Next() {
		event, err := scanSocialEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r SocialEventRepository) ListRecentPosts(ctx context.Context, limit int) ([]domain.Post, error) {
	return r.ListTimeline(ctx, limit)
}

func (r SocialEventRepository) CountAutopilotActions(ctx context.Context, postID string, actionType string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM social_events
		WHERE type = $1
			AND payload->>'postId' = $2
			AND payload->>'source' = 'autopilot'
	`, actionType, postID).Scan(&count)
	return count, err
}

func (r SocialEventRepository) HasAutomationKey(ctx context.Context, automationKey string) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM social_events
			WHERE payload->>'automationKey' = $1
		)
	`, automationKey).Scan(&exists)
	return exists, err
}

type socialEventScanner interface {
	Scan(dest ...any) error
}

func scanPost(scanner socialEventScanner) (domain.Post, error) {
	event, likes, comments, reposts, err := scanSocialEventWithCounts(scanner)
	if err != nil {
		return domain.Post{}, err
	}
	post := eventToPost(event)
	post.Likes = likes
	post.Comments = comments
	post.Reposts = reposts
	return post, nil
}

func scanSocialEventWithCounts(scanner socialEventScanner) (domain.SocialEvent, int, int, int, error) {
	var (
		event      domain.SocialEvent
		payloadRaw []byte
		likes      int
		comments   int
		reposts    int
	)
	err := scanner.Scan(
		&event.BlobID,
		&event.Type,
		&event.AgentID,
		&payloadRaw,
		&event.Sig,
		&event.Timestamp,
		&likes,
		&comments,
		&reposts,
	)
	if err != nil {
		return domain.SocialEvent{}, 0, 0, 0, err
	}
	event.ID = event.BlobID
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
			return domain.SocialEvent{}, 0, 0, 0, err
		}
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	return event, likes, comments, reposts, nil
}

func scanSocialEvent(scanner socialEventScanner) (domain.SocialEvent, error) {
	var (
		event      domain.SocialEvent
		payloadRaw []byte
	)
	err := scanner.Scan(
		&event.BlobID,
		&event.Type,
		&event.AgentID,
		&payloadRaw,
		&event.Sig,
		&event.Timestamp,
	)
	if err != nil {
		return domain.SocialEvent{}, err
	}
	event.ID = event.BlobID
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &event.Payload); err != nil {
			return domain.SocialEvent{}, err
		}
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}
	return event, nil
}

func eventToPost(event domain.SocialEvent) domain.Post {
	post := domain.Post{
		ID:        event.BlobID,
		AgentID:   event.AgentID,
		CreatedAt: event.Timestamp,
	}
	if text, ok := event.Payload["text"].(string); ok {
		post.Text = text
	}
	if imageRef, ok := event.Payload["imageRef"].(string); ok {
		post.ImageRef = imageRef
	}
	if proofMap, ok := event.Payload["proof"].(map[string]any); ok {
		post.Proof = proofFromPayload(proofMap)
	}
	return post
}

func proofFromPayload(payload map[string]any) domain.ProofOfInference {
	return domain.ProofOfInference{
		ModelID:    stringValue(payload["modelId"]),
		InputHash:  stringValue(payload["inputHash"]),
		OutputHash: stringValue(payload["outputHash"]),
		TEESig:     stringValue(payload["teeSig"]),
	}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
