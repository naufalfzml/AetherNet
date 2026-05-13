package main

import (
	"context"
	"log"
	"net/http"

	deliveryhttp "github.com/aethernet-0g/aethernet/backend/delivery/http"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/chain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/compute"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/postgres"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/storage"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidateBackendRuntime(); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	metrics := &usecase.Metrics{}
	server := deliveryhttp.Server{Metrics: metrics, Config: cfg}
	if cfg.HasComputeSidecar() {
		server.Compute = compute.HTTPClient{BaseURL: cfg.ComputeSidecarURL, RequireReal: !cfg.StubMode}
	} else {
		server.Compute = compute.StubClient{}
	}
	if cfg.HasStorageSidecar() {
		server.Storage = storage.NewHTTPClient(cfg.StorageSidecarURL)
		log.Printf("storage backed by sidecar at %s", cfg.StorageSidecarURL)
	} else {
		server.Storage = storage.NewStubClient()
		log.Printf("storage backed by in-memory stub (set STORAGE_SIDECAR_URL to use 0G Storage)")
	}
	if cfg.DatabaseURL != "" {
		db, err := postgres.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			if !cfg.StubMode {
				log.Fatalf("open postgres: %v", err)
			}
			log.Printf("postgres disabled: %v", err)
		} else {
			defer db.Close()
			server.Agents = postgres.AgentRepository{DB: db}
			server.ExternalAgents = postgres.ExternalAgentRepository{DB: db}
			server.Events = postgres.SocialEventRepository{DB: db}
			server.Metadata = postgres.AgentMetadataRepository{DB: db}
		}
	}
	server.Health = usecase.HealthService{
		Storage: server.Storage,
		Compute: server.Compute,
	}
	if cfg.HasChainRPC() {
		server.Health.Chain = chain.LogScanner{RPCURL: cfg.OGRPCURL}
	} else {
		server.Health.Chain = usecase.StaticHealthClient{Name: "chain"}
	}
	log.Printf("backend listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
