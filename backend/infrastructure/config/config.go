package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr          string
	DatabaseURL       string
	StubMode          bool
	OGRPCURL          string
	OGChainID         string
	OGExplorerURL     string
	ComputeSidecarURL string
	INFTRegistry      string
	TreasuryFactory   string
}

func Load() Config {
	return Config{
		HTTPAddr:          env("HTTP_ADDR", env("BACKEND_HTTP_ADDR", ":8080")),
		DatabaseURL:       env("DATABASE_URL", ""),
		StubMode:          envBool("STUB_MODE", true),
		OGRPCURL:          env("OG_RPC_URL", ""),
		OGChainID:         env("OG_CHAIN_ID", "16601"),
		OGExplorerURL:     env("OG_EXPLORER_URL", ""),
		ComputeSidecarURL: env("COMPUTE_SIDECAR_URL", ""),
		INFTRegistry:      env("INFT_REGISTRY_ADDRESS", ""),
		TreasuryFactory:   env("TREASURY_FACTORY_ADDRESS", ""),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
