## ADDED Requirements

### Requirement: Real Mode Metadata Uses 0G Storage
When the backend is not in stub mode, persona metadata created during mint preparation SHALL be uploaded through the configured storage service and the returned pointer SHALL be used as the metadata pointer.

#### Scenario: Real mode metadata creation succeeds
- **GIVEN** `STUB_MODE=false`
- **AND** a storage client is configured and healthy enough to accept uploads
- **WHEN** the frontend submits a persona prompt to `POST /metadata`
- **THEN** the backend uploads a JSON metadata document to storage
- **AND** stores the returned pointer with the prompt and personality summary in `agent_metadata`
- **AND** returns that storage pointer in `metadataPointer`.

#### Scenario: Real mode storage is unavailable
- **GIVEN** `STUB_MODE=false`
- **AND** no storage client is configured
- **WHEN** the frontend submits a persona prompt to `POST /metadata`
- **THEN** the backend returns a clear service error
- **AND** it does not mint or store a `stub://metadata/...` pointer.

### Requirement: Stub Metadata Is Explicitly Local
The backend MAY create `stub://metadata/...` pointers only when explicit local stub mode is enabled.

#### Scenario: Stub mode metadata creation
- **GIVEN** `STUB_MODE=true`
- **WHEN** the frontend submits a persona prompt to `POST /metadata`
- **THEN** the backend may create a local `stub://metadata/...` pointer
- **AND** stores the prompt and summary locally for profile resolution.
