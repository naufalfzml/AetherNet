package domain

import "time"

type Agent struct {
	ID                 string    `json:"id"`
	TokenID            string    `json:"tokenId"`
	OwnerAddress       string    `json:"ownerAddress"`
	AgentAddress       string    `json:"agentAddress"`
	TreasuryAddress    string    `json:"treasuryAddress"`
	MetadataPointer    string    `json:"metadataPointer"`
	PersonalitySummary string    `json:"personalitySummary"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type AgentMetadata struct {
	MetadataPointer    string    `json:"metadataPointer"`
	Prompt             string    `json:"prompt"`
	PersonalitySummary string    `json:"personalitySummary"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Post struct {
	ID        string           `json:"id"`
	AgentID   string           `json:"agentId"`
	Text      string           `json:"text"`
	ImageRef  string           `json:"imageRef,omitempty"`
	Proof     ProofOfInference `json:"proof"`
	Likes     int              `json:"likes"`
	Comments  int              `json:"comments"`
	Reposts   int              `json:"reposts"`
	CreatedAt time.Time        `json:"createdAt"`
}

type Investor struct {
	Address            string `json:"address"`
	AgentID            string `json:"agentId"`
	ShareBalance       string `json:"shareBalance"`
	ClaimableDividends string `json:"claimableDividends"`
}

type ProofOfInference struct {
	ModelID    string `json:"modelId"`
	InputHash  string `json:"inputHash"`
	OutputHash string `json:"outputHash"`
	TEESig     string `json:"teeSig"`
}

type SocialEvent struct {
	ID        string         `json:"id"`
	BlobID    string         `json:"blobId"`
	Type      string         `json:"type"`
	AgentID   string         `json:"agentId"`
	Payload   map[string]any `json:"payload"`
	Sig       string         `json:"sig"`
	Timestamp time.Time      `json:"timestamp"`
}
