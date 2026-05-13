package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr                       string
	DatabaseURL                    string
	StubMode                       bool
	OGRPCURL                       string
	OGChainID                      string
	OGExplorerURL                  string
	IndexerStartBlock              string
	IndexerConfirmations           string
	ComputeSidecarURL              string
	StorageSidecarURL              string
	ImageProvider                  string
	INFTRegistry                   string
	TreasuryFactory                string
	DemoTokenID                    string
	DemoTreasury                   string
	PlatformWallet                 string
	Orchestrator                   string
	AutopilotWorkerIntervalSeconds int
	AutopilotPostIntervalSeconds   int
	AutopilotMaxPostsPerTick       int
	AutopilotMaxLikesPerPost       int
	AutopilotMaxCommentsPerPost    int
}

func Load() Config {
	loadDotEnv("../.env")
	loadDotEnv(".env")

	return Config{
		HTTPAddr:                       env("HTTP_ADDR", env("BACKEND_HTTP_ADDR", ":8080")),
		DatabaseURL:                    env("DATABASE_URL", ""),
		StubMode:                       envBool("STUB_MODE", true),
		OGRPCURL:                       env("OG_RPC_URL", ""),
		OGChainID:                      env("OG_CHAIN_ID", "16602"),
		OGExplorerURL:                  env("OG_EXPLORER_URL", ""),
		IndexerStartBlock:              env("INDEXER_START_BLOCK", "0"),
		IndexerConfirmations:           env("INDEXER_CONFIRMATIONS", "2"),
		ComputeSidecarURL:              env("COMPUTE_SIDECAR_URL", ""),
		StorageSidecarURL:              env("STORAGE_SIDECAR_URL", ""),
		ImageProvider:                  env("IMAGE_PROVIDER", "none"),
		INFTRegistry:                   env("INFT_REGISTRY_ADDRESS", ""),
		TreasuryFactory:                env("TREASURY_FACTORY_ADDRESS", ""),
		DemoTokenID:                    env("DEMO_TOKEN_ID", "1"),
		DemoTreasury:                   env("DEMO_TREASURY_ADDRESS", ""),
		PlatformWallet:                 env("PLATFORM_WALLET", ""),
		Orchestrator:                   env("ORCHESTRATOR_ADDRESS", ""),
		AutopilotWorkerIntervalSeconds: envInt("AUTOPILOT_WORKER_INTERVAL_SECONDS", 10),
		AutopilotPostIntervalSeconds:   envInt("AUTOPILOT_POST_INTERVAL_SECONDS", 120),
		AutopilotMaxPostsPerTick:       envInt("AUTOPILOT_MAX_POSTS_PER_TICK", 5),
		AutopilotMaxLikesPerPost:       envInt("AUTOPILOT_MAX_LIKES_PER_POST", 3),
		AutopilotMaxCommentsPerPost:    envInt("AUTOPILOT_MAX_COMMENTS_PER_POST", 2),
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

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
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

func (c Config) HasComputeSidecar() bool {
	return configuredEndpoint(c.ComputeSidecarURL)
}

func (c Config) HasStorageSidecar() bool {
	return configuredEndpoint(c.StorageSidecarURL)
}

func (c Config) HasChainRPC() bool {
	return configuredEndpoint(c.OGRPCURL)
}

func (c Config) ValidateBackendRuntime() error {
	if c.StubMode {
		return nil
	}
	missing := make([]string, 0)
	if strings.TrimSpace(c.DatabaseURL) == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if !c.HasChainRPC() {
		missing = append(missing, "OG_RPC_URL")
	}
	if !c.HasComputeSidecar() {
		missing = append(missing, "COMPUTE_SIDECAR_URL")
	}
	if !c.HasStorageSidecar() {
		missing = append(missing, "STORAGE_SIDECAR_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("real backend mode requires %s (set STUB_MODE=true for local stub mode)", strings.Join(missing, ", "))
	}
	return nil
}

func (c Config) ValidateWorkerRuntime() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.StubMode {
		return nil
	}
	missing := make([]string, 0)
	if !c.HasComputeSidecar() {
		missing = append(missing, "COMPUTE_SIDECAR_URL")
	}
	if !c.HasStorageSidecar() {
		missing = append(missing, "STORAGE_SIDECAR_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("real autopilot mode requires %s (set STUB_MODE=true for local stub mode)", strings.Join(missing, ", "))
	}
	return nil
}

func configuredEndpoint(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "<") && !strings.HasSuffix(value, ">")
}
