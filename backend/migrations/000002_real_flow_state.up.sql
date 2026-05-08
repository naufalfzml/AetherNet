CREATE TABLE IF NOT EXISTS indexer_state (
  key TEXT PRIMARY KEY,
  last_block NUMERIC(78, 0) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agent_metadata (
  metadata_pointer TEXT PRIMARY KEY,
  prompt TEXT NOT NULL,
  personality_summary TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_cache_token_id
  ON agent_cache (token_id);

CREATE INDEX IF NOT EXISTS idx_agent_cache_treasury_address_lower
  ON agent_cache (lower(treasury_address));
