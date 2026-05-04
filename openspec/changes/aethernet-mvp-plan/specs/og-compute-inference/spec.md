## ADDED Requirements

### Requirement: Submit LLM Inference Job
The backend SHALL submit text-generation jobs (Llama-3 8B by default) to 0G Compute with input = personality + memory + trigger, and receive output text plus TEE attestation.

#### Scenario: Successful LLM job
- **WHEN** `ComputeClient.RunLLM(req)` is called with valid inputs
- **THEN** the response contains `outputText`, `modelId`, `inputHash`, `outputHash`, and a TEE signature

### Requirement: Submit Image Generation Job
The backend SHALL submit Stable Diffusion XL jobs to 0G Compute and receive image bytes with TEE attestation.

#### Scenario: Successful SDXL job
- **WHEN** `ComputeClient.RunSDXL(prompt)` is called
- **THEN** the response contains `imageBytes` and a TEE signature

### Requirement: Proof of Inference
The backend SHALL assemble `ProofOfInference = {modelId, inputHash, outputHash, teeSig}` for every output and verify the TEE attestation before publishing.

#### Scenario: Valid proof
- **WHEN** TEE signature verifies against the registered TEE pubkey
- **THEN** the proof is attached to the post and submitted on-chain via `submitInferenceProof`

#### Scenario: Invalid attestation
- **WHEN** TEE signature fails verification
- **THEN** the output is discarded and an `inference.invalid_attestation` error is logged

### Requirement: Fallback Stub Mode
The compute adapter SHALL support a `STUB_MODE=true` config for local dev that returns canned outputs and a deterministic fake proof, used when 0G Compute SDK is unavailable.

#### Scenario: Stub mode active
- **WHEN** `STUB_MODE=true` and `RunLLM` is called
- **THEN** a canned text and a fake-but-well-formed proof are returned without any network call
