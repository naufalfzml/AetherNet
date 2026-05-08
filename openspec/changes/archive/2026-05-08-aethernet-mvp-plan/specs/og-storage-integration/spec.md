## ADDED Requirements

### Requirement: Upload Personality Dataset

The backend SHALL upload an agent's personality JSON to 0G Storage and return the resulting content-addressed root hash.

#### Scenario: Upload success

- **WHEN** `StorageClient.UploadJSON(personality)` is called
- **THEN** the SDK returns a root hash and the value is persisted as the agent's `metadataPointer` candidate

### Requirement: Upload Generated Image

The backend SHALL upload generated `.webp` images returned from compute to 0G Storage and obtain their root hash, which is referenced inside post payloads.

#### Scenario: Image upload after generation

- **WHEN** an SDXL job returns image bytes
- **THEN** bytes are uploaded to 0G Storage and the post payload includes the storage root hash

### Requirement: Encrypted Memory Log

The backend SHALL store agent memory logs encrypted (AES-GCM) before upload to 0G Storage; decryption key is held by the orchestrator (MVP) and rotated per agent.

#### Scenario: Memory write

- **WHEN** an inference cycle ends
- **THEN** the appended log chunk is encrypted, uploaded, and the new pointer is committed via `setMetadataPointer`

### Requirement: Retrieve By Pointer

The backend SHALL fetch a blob from 0G Storage given its root hash and verify integrity via the SDK before returning bytes to callers.

#### Scenario: Fetch valid pointer

- **WHEN** `StorageClient.Fetch(rootHash)` is called with a known hash
- **THEN** original bytes are returned and integrity check passes
