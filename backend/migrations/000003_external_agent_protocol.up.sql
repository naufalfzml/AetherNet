CREATE TABLE IF NOT EXISTS external_agents (
  external_agent_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL DEFAULT 'external',
  status TEXT NOT NULL DEFAULT 'pending_verification',
  display_name TEXT NOT NULL,
  handle TEXT NOT NULL UNIQUE,
  owner_wallet_address TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  personality_summary TEXT NOT NULL DEFAULT '',
  metadata_pointer TEXT NOT NULL DEFAULT '',
  linked_native_agent_id TEXT NOT NULL DEFAULT '',
  minted_token_id TEXT NOT NULL DEFAULT '',
  api_key_hash TEXT NOT NULL DEFAULT '',
  wallet_verified_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_external_agents_owner_wallet
  ON external_agents (lower(owner_wallet_address));

CREATE UNIQUE INDEX IF NOT EXISTS idx_external_agents_api_key_hash
  ON external_agents (api_key_hash)
  WHERE api_key_hash <> '';

CREATE TABLE IF NOT EXISTS external_agent_auth_challenges (
  challenge_id TEXT PRIMARY KEY,
  external_agent_id TEXT NOT NULL REFERENCES external_agents(external_agent_id) ON DELETE CASCADE,
  wallet_address TEXT NOT NULL,
  challenge_text TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_external_agent_auth_challenges_agent
  ON external_agent_auth_challenges (external_agent_id, created_at DESC);
