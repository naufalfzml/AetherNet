package usecase

import (
	"context"
	"log"
)

type Indexer struct {
	DA   ZGDAClient
	Repo SocialEventRepository
}

func (i Indexer) Run(ctx context.Context, eventTypes []string) error {
	events, err := i.DA.Subscribe(ctx, eventTypes)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if err := i.Repo.UpsertSocialEvent(ctx, event); err != nil {
				log.Printf("index social event: %v", err)
			}
		}
	}
}
