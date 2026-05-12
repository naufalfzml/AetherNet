package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	StubMode             bool
	OGRPCURL             string
	OGChainID            string
	OGExplorerURL        string
	IndexerStartBlock    string
	IndexerConfirmations string
	ComputeSidecarURL    string
	StorageSidecarURL    string
	DASidecarURL         string
	ImageProvider        string
	INFTRegistry         string
	TreasuryFactory      string
	DemoTokenID          string
	DemoTreasury         string
	PlatformWallet       string
	Orchestrator         string
}

func Load() Config {
	loadDotEnv("../.env")
	loadDotEnv(".env")

	return Config{
		HTTPAddr:             env("HTTP_ADDR", env("BACKEND_HTTP_ADDR", ":8080")),
		DatabaseURL:          env("DATABASE_URL", ""),
		StubMode:             envBool("STUB_MODE", true),
		OGRPCURL:             env("OG_RPC_URL", ""),
		OGChainID:            env("OG_CHAIN_ID", "16602"),
		OGExplorerURL:        env("OG_EXPLORER_URL", ""),
		IndexerStartBlock:    env("INDEXER_START_BLOCK", "0"),
		IndexerConfirmations: env("INDEXER_CONFIRMATIONS", "2"),
		ComputeSidecarURL:    env("COMPUTE_SIDECAR_URL", ""),
		StorageSidecarURL:    env("STORAGE_SIDECAR_URL", ""),
		DASidecarURL:         env("DA_SIDECAR_URL", ""),
		ImageProvider:        env("IMAGE_PROVIDER", "none"),
		INFTRegistry:         env("INFT_REGISTRY_ADDRESS", ""),
		TreasuryFactory:      env("TREASURY_FACTORY_ADDRESS", ""),
		DemoTokenID:          env("DEMO_TOKEN_ID", "1"),
		DemoTreasury:         env("DEMO_TREASURY_ADDRESS", ""),
		PlatformWallet:       env("PLATFORM_WALLET", ""),
		Orchestrator:         env("ORCHESTRATOR_ADDRESS", ""),
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

func loadDotEnv(path string) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

func (c Config) ORCHOwnerFallback() string {
	if c.Orchestrator != "" {
		return c.Orchestrator
	}
	if c.PlatformWallet != "" {
		return c.PlatformWallet
	}
	return c.INFTRegistry
}
