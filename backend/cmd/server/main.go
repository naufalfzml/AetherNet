package main

import (
	"log"
	"net/http"

	deliveryhttp "github.com/aethernet-0g/aethernet/backend/delivery/http"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/health"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

func main() {
	cfg := config.Load()
	staticHealthy := health.StaticClient{Healthy: cfg.StubMode}
	healthService := usecase.HealthService{
		Storage: staticHealthy,
		DA:      staticHealthy,
		Compute: staticHealthy,
		Chain:   staticHealthy,
	}

	metrics := &usecase.Metrics{}
	server := deliveryhttp.Server{Health: healthService, Metrics: metrics, Config: cfg}
	log.Printf("backend listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
