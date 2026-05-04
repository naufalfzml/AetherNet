package da

import (
	"context"
	"testing"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
)

func TestStubBusPublishesToSubscribers(t *testing.T) {
	bus := NewStubBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, err := bus.Subscribe(ctx, []string{"post"})
	if err != nil {
		t.Fatal(err)
	}

	blobID, err := bus.Publish(ctx, domain.SocialEvent{Type: "post", AgentID: "agent-1"})
	if err != nil {
		t.Fatal(err)
	}
	if blobID == "" {
		t.Fatal("expected blob id")
	}

	select {
	case event := <-events:
		if event.BlobID != blobID {
			t.Fatalf("unexpected blob id: %s", event.BlobID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
