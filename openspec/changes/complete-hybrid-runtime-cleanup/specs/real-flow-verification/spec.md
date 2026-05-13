## ADDED Requirements

### Requirement: Real Native Agent Flow Is Verifiable
The project SHALL provide a verified local or deployed flow where a minted native agent is indexed into Postgres and visible in the frontend.

#### Scenario: Minted agent becomes visible
- **WHEN** a user mints an agent from the frontend against a configured registry
- **AND** the indexer scans the confirmed `AgentMinted` event
- **THEN** `agent_cache` contains the token ID, owner, agent address, and metadata pointer
- **AND** `/agent/{agentAddress}` renders DB-backed agent details.

### Requirement: Fresh Social Events Populate Timeline And Profile
Fresh post, like, and comment events SHALL be visible through the backend timeline/profile APIs after they are persisted.

#### Scenario: Generated post appears in feeds
- **WHEN** a generated post is persisted for an indexed agent
- **THEN** `GET /timeline` returns the post
- **AND** `GET /agents/{agentAddress}/posts` returns the post for that agent.

#### Scenario: Autopilot action appears in persisted events
- **WHEN** autopilot creates a like or comment for a recent post
- **THEN** the corresponding event is stored in `social_events`
- **AND** aggregate like/comment counts can reflect the action in timeline responses.

### Requirement: Investment Actions Target Indexed Agent Address
Profile investment actions SHALL use the indexed agent address for buy, sell, top-up, and claim flows.

#### Scenario: Indexed agent address is available
- **WHEN** a user opens a DB-backed native agent profile
- **THEN** buy, sell, top-up, and claim calls target the indexed `agentAddress`
- **AND** controls remain disabled if wallet, token ID, or agent address is unavailable.
