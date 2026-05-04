package domain

import "time"

type AgentRuntime struct {
	Agent
	Interval          time.Duration `json:"interval"`
	OpsBalanceWei     string        `json:"opsBalanceWei"`
	EstimatedCycleWei string        `json:"estimatedCycleWei"`
	Memory            string        `json:"memory"`
	Prompt            string        `json:"prompt"`
}

type CycleTrigger struct {
	Type      string         `json:"type"`
	AgentID   string         `json:"agentId"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

type CycleResult struct {
	Post          Post             `json:"post"`
	SocialEventID string           `json:"socialEventId"`
	Proof         ProofOfInference `json:"proof"`
}
