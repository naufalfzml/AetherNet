CREATE TABLE IF NOT EXISTS social_events (
  id BIGSERIAL PRIMARY KEY,
  blob_id TEXT NOT NULL UNIQUE,
  type TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  sig TEXT NOT NULL,
  event_timestamp TIMESTAMPTZ NOT NULL,
  ingested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_social_events_agent_id
  ON social_events (agent_id);

CREATE INDEX IF NOT EXISTS idx_social_events_type
  ON social_events (type);

CREATE INDEX IF NOT EXISTS idx_social_events_timestamp
  ON social_events (event_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_social_events_agent_type_timestamp
  ON social_events (agent_id, type, event_timestamp DESC);

CREATE TABLE IF NOT EXISTS agent_cache (
  agent_id TEXT PRIMARY KEY,
  token_id NUMERIC(78, 0) NOT NULL,
  owner_address TEXT NOT NULL,
  treasury_address TEXT NOT NULL,
  metadata_pointer TEXT NOT NULL,
  personality_summary TEXT NOT NULL DEFAULT '',
  share_supply NUMERIC(78, 0) NOT NULL DEFAULT 0,
  operational_balance NUMERIC(78, 0) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_cache_owner_address
  ON agent_cache (owner_address);
