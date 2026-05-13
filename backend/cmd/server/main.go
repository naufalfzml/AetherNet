package main

import (
	"context"
	"log"
	"net/http"

	deliveryhttp "github.com/aethernet-0g/aethernet/backend/delivery/http"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/compute"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/health"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/postgres"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/storage"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	staticHealthy := health.StaticClient{Healthy: cfg.StubMode}
	healthService := usecase.HealthService{
		Storage: staticHealthy,
		Compute: staticHealthy,
		Chain:   staticHealthy,
	}

	metrics := &usecase.Metrics{}
	server := deliveryhttp.Server{Health: healthService, Metrics: metrics, Config: cfg}
	if cfg.ComputeSidecarURL != "" && cfg.ComputeSidecarURL != "<LOCAL_COMPUTE_SIDECAR_URL>" {
		server.Compute = compute.HTTPClient{BaseURL: cfg.ComputeSidecarURL}
	} else {
		server.Compute = compute.StubClient{}
	}
	if cfg.StorageSidecarURL != "" {
		server.Storage = storage.NewHTTPClient(cfg.StorageSidecarURL)
		log.Printf("storage backed by sidecar at %s", cfg.StorageSidecarURL)
	} else {
		server.Storage = storage.NewStubClient()
		log.Printf("storage backed by in-memory stub (set STORAGE_SIDECAR_URL to use 0G Storage)")
	}
	if cfg.DatabaseURL != "" {
		db, err := postgres.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("postgres disabled: %v", err)
		} else {
			defer db.Close()
			server.Agents = postgres.AgentRepository{DB: db}
			server.ExternalAgents = postgres.ExternalAgentRepository{DB: db}
			server.Events = postgres.SocialEventRepository{DB: db}
			server.Metadata = postgres.AgentMetadataRepository{DB: db}
		}
	}
	log.Printf("backend listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
