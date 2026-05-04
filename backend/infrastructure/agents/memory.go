package agents

import (
	"context"
	"sync"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type MemoryProvider struct {
	mu     sync.RWMutex
	agents map[string]domain.AgentRuntime
}

func NewMemoryProvider(initial []domain.AgentRuntime) *MemoryProvider {
	agents := make(map[string]domain.AgentRuntime, len(initial))
	for _, agent := range initial {
		agents[agent.ID] = agent
	}
	return &MemoryProvider{agents: agents}
}

func (p *MemoryProvider) ActiveAgents(context.Context) ([]domain.AgentRuntime, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agents := make([]domain.AgentRuntime, 0, len(p.agents))
	for _, agent := range p.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

func (p *MemoryProvider) UpdateMetadataPointer(_ context.Context, agentID string, pointer string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	agent := p.agents[agentID]
	agent.MetadataPointer = pointer
	p.agents[agentID] = agent
	return nil
}
