package usecase

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type AgentProvider interface {
	ActiveAgents(ctx context.Context) ([]domain.AgentRuntime, error)
	UpdateMetadataPointer(ctx context.Context, agentID string, pointer string) error
}

type OpenClaw struct {
	Agents       AgentProvider
	Storage      StorageService
	Compute      ZGComputeClient
	DA           ZGDAClient
	Chain        ChainClient
	SocialEvents SocialEventRepository
	Metrics      *Metrics
}

type Metrics struct {
	CycleCount uint64
	Failures   uint64
}

func (m *Metrics) Snapshot() map[string]uint64 {
	if m == nil {
		return map[string]uint64{"cycle_count": 0, "failures": 0}
	}
	return map[string]uint64{
		"cycle_count": atomic.LoadUint64(&m.CycleCount),
		"failures":    atomic.LoadUint64(&m.Failures),
	}
}

func (o OpenClaw) RunScheduler(ctx context.Context) error {
	agents, err := o.Agents.ActiveAgents(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, agent := range agents {
		agent := agent
		if agent.Interval <= 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(agent.Interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case tick := <-ticker.C:
					trigger := domain.CycleTrigger{
						Type:      "schedule",
						AgentID:   agent.ID,
						Timestamp: tick,
					}
					if _, err := o.RunCycle(ctx, agent, trigger); err != nil {
						o.recordFailure("scheduled cycle", err)
					}
				}
			}
		}()
	}

	<-ctx.Done()
	wg.Wait()
	return ctx.Err()
}

func (o OpenClaw) RunCycle(ctx context.Context, agent domain.AgentRuntime, trigger domain.CycleTrigger) (domain.CycleResult, error) {
	if !hasBudget(agent.OpsBalanceWei, agent.EstimatedCycleWei) {
		log.Printf("agent.paused.no_funds agent=%s", agent.ID)
		return domain.CycleResult{}, fmt.Errorf("agent.paused.no_funds")
	}

	llm, err := o.Compute.RunLLM(ctx, LLMRequest{
		AgentID:     agent.ID,
		Personality: agent.Prompt,
		Memory:      agent.Memory,
		Trigger:     trigger.Type,
	})
	if err != nil {
		o.recordFailure("compute", err)
		return domain.CycleResult{}, err
	}

	post := domain.Post{
		ID:        fmt.Sprintf("%s-%d", agent.ID, time.Now().UnixNano()),
		AgentID:   agent.ID,
		Text:      llm.OutputText,
		Proof:     llm.Proof,
		CreatedAt: time.Now().UTC(),
	}
	pointer, err := o.Storage.AppendEncryptedMemory(ctx, agent.ID, make([]byte, 32), []byte(agent.Memory), map[string]any{
		"trigger": trigger,
		"post":    post,
	})
	if err != nil {
		o.recordFailure("storage", err)
		return domain.CycleResult{}, err
	}

	event := domain.SocialEvent{
		Type:      "post",
		AgentID:   agent.ID,
		Payload:   map[string]any{"text": post.Text, "proof": post.Proof, "memoryPointer": pointer},
		Sig:       "stub",
		Timestamp: post.CreatedAt,
	}
	blobID, err := o.DA.Publish(ctx, event)
	if err != nil {
		o.recordFailure("da", err)
		return domain.CycleResult{}, err
	}
	event.BlobID = blobID
	if o.SocialEvents != nil {
		if err := o.SocialEvents.UpsertSocialEvent(ctx, event); err != nil {
			o.recordFailure("persist social event", err)
			return domain.CycleResult{}, err
		}
	}
	if err := o.Agents.UpdateMetadataPointer(ctx, agent.ID, pointer); err != nil {
		o.recordFailure("metadata", err)
		return domain.CycleResult{}, err
	}
	if _, err := o.Chain.SubmitInferenceProof(ctx, agent.TokenID, llm.Proof); err != nil {
		o.recordFailure("chain", err)
		return domain.CycleResult{}, err
	}

	if o.Metrics != nil {
		atomic.AddUint64(&o.Metrics.CycleCount, 1)
	}
	return domain.CycleResult{Post: post, SocialEventID: blobID, Proof: llm.Proof}, nil
}

func (o OpenClaw) RunReplyLoop(ctx context.Context, agents map[string]domain.AgentRuntime) error {
	if o.SocialEvents == nil {
		log.Println("RunReplyLoop disabled: SocialEventRepository is nil")
		<-ctx.Done()
		return ctx.Err()
	}

	lastSeen := make(map[string]time.Time)
	for id := range agents {
		lastSeen[id] = time.Now().UTC()
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for id, agent := range agents {
				events, err := o.SocialEvents.ListAgentSocialEvents(ctx, id, 10)
				if err != nil {
					continue
				}

				// Process events from oldest to newest
				for i := len(events) - 1; i >= 0; i-- {
					event := events[i]
					if event.Timestamp.After(lastSeen[id]) && (event.Type == "mention" || event.Type == "comment") {
						lastSeen[id] = event.Timestamp
						_, err := o.RunCycle(ctx, agent, domain.CycleTrigger{
							Type:      event.Type,
							AgentID:   event.AgentID,
							Payload:   event.Payload,
							Timestamp: event.Timestamp,
						})
						if err != nil {
							o.recordFailure("reply cycle", err)
						}
					}
				}
			}
		}
	}
}

func (o OpenClaw) recordFailure(stage string, err error) {
	if o.Metrics != nil {
		atomic.AddUint64(&o.Metrics.Failures, 1)
	}
	log.Printf("openclaw.%s.failed: %v", stage, err)
}

func hasBudget(balanceWei, estimatedWei string) bool {
	if estimatedWei == "" || estimatedWei == "0" {
		return true
	}
	balance, ok := new(big.Int).SetString(balanceWei, 10)
	if !ok {
		return false
	}
	estimated, ok := new(big.Int).SetString(estimatedWei, 10)
	if !ok {
		return false
	}
	return balance.Cmp(estimated) >= 0
}
