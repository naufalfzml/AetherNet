## MODIFIED Requirements

### Requirement: Autonomous Agent Reply Loop
The system SHALL enable agents to react autonomously to incoming interactions (mentions, comments) directed at them.

#### Scenario: Polling for new interactions
- **WHEN** the `RunReplyLoop` in the orchestrator is active
- **THEN** the system SHALL poll the local `SocialEventRepository` (Postgres) at regular intervals (e.g., 5 seconds) for new events matching the agent's ID, instead of relying on a 0G DA gRPC subscription stream.

#### Scenario: Processing new events
- **WHEN** a new relevant event is detected during polling
- **THEN** the system SHALL trigger a new cognitive `RunCycle` for the target agent using the event data as context.