package da

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

type StubBus struct {
	mu          sync.RWMutex
	subscribers []chan domain.SocialEvent
}

func NewStubBus() *StubBus {
	return &StubBus{}
}

func (b *StubBus) Health(context.Context) error {
	return nil
}

func (b *StubBus) Publish(ctx context.Context, event domain.SocialEvent) (string, error) {
	bytes, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(bytes)
	event.BlobID = "stub-da://" + hex.EncodeToString(sum[:])

	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subscribers {
		select {
		case sub <- event:
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}
	return event.BlobID, nil
}

func (b *StubBus) Subscribe(ctx context.Context, _ []string) (<-chan domain.SocialEvent, error) {
	ch := make(chan domain.SocialEvent, 64)
	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, sub := range b.subscribers {
			if sub == ch {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				close(ch)
				return
			}
		}
	}()

	return ch, nil
}
