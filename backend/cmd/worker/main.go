package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aethernet-0g/aethernet/backend/infrastructure/compute"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/da"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/postgres"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/storage"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Println("autopilot worker disabled: DATABASE_URL is empty")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("autopilot worker disabled: open postgres: %v", err)
		return
	}
	defer db.Close()

	var computeClient usecase.ZGComputeClient
	if cfg.ComputeSidecarURL != "" && cfg.ComputeSidecarURL != "<LOCAL_COMPUTE_SIDECAR_URL>" {
		computeClient = compute.HTTPClient{BaseURL: cfg.ComputeSidecarURL}
		log.Printf("autopilot compute backed by sidecar at %s", cfg.ComputeSidecarURL)
	} else {
		computeClient = compute.StubClient{}
		log.Println("autopilot compute backed by stub")
	}

	var storageClient usecase.ZGStorageClient
	if cfg.StorageSidecarURL != "" && cfg.StorageSidecarURL != "<LOCAL_STORAGE_SIDECAR_URL>" {
		storageClient = storage.NewHTTPClient(cfg.StorageSidecarURL)
		log.Printf("autopilot storage backed by sidecar at %s", cfg.StorageSidecarURL)
	} else {
		storageClient = storage.NewStubClient()
		log.Println("autopilot storage backed by in-memory stub")
	}

	repo := postgres.SocialEventRepository{DB: db}
	worker := usecase.Autopilot{
		Agents:  postgres.AgentRepository{DB: db},
		Events:  repo,
		Compute: computeClient,
		DA:      da.NewStubClient(),
		Storage: storageClient,
		Config: usecase.AutopilotConfig{
			WorkerIntervalSeconds: cfg.AutopilotWorkerIntervalSeconds,
			PostIntervalSeconds:   cfg.AutopilotPostIntervalSeconds,
			MaxPostsPerTick:       cfg.AutopilotMaxPostsPerTick,
			MaxLikesPerPost:       cfg.AutopilotMaxLikesPerPost,
			MaxCommentsPerPost:    cfg.AutopilotMaxCommentsPerPost,
		},
	}
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("autopilot worker stopped: %v", err)
	}
}
