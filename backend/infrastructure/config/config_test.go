package config

import (
	"strings"
	"testing"
)

func TestConfiguredEndpointRejectsPlaceholders(t *testing.T) {
	cfg := Config{
		ComputeSidecarURL: "<LOCAL_COMPUTE_SIDECAR_URL>",
		StorageSidecarURL: "",
		OGRPCURL:          "<0G_EVM_RPC_URL>",
	}
	if cfg.HasComputeSidecar() {
		t.Fatalf("placeholder compute sidecar should not be configured")
	}
	if cfg.HasStorageSidecar() {
		t.Fatalf("empty storage sidecar should not be configured")
	}
	if cfg.HasChainRPC() {
		t.Fatalf("placeholder chain RPC should not be configured")
	}
}

func TestValidateBackendRuntimeAllowsStubMode(t *testing.T) {
	cfg := Config{StubMode: true}
	if err := cfg.ValidateBackendRuntime(); err != nil {
		t.Fatalf("stub backend mode should allow missing real dependencies: %v", err)
	}
}

func TestValidateBackendRuntimeRequiresRealDependencies(t *testing.T) {
	cfg := Config{StubMode: false}
	err := cfg.ValidateBackendRuntime()
	if err == nil {
		t.Fatalf("expected missing dependency error")
	}
	message := err.Error()
	for _, key := range []string{"DATABASE_URL", "OG_RPC_URL", "COMPUTE_SIDECAR_URL", "STORAGE_SIDECAR_URL"} {
		if !strings.Contains(message, key) {
			t.Fatalf("expected %s in error %q", key, message)
		}
	}
}

func TestValidateWorkerRuntimeRequiresRealSidecars(t *testing.T) {
	cfg := Config{
		StubMode:    false,
		DatabaseURL: "postgres://example",
	}
	err := cfg.ValidateWorkerRuntime()
	if err == nil {
		t.Fatalf("expected missing sidecar error")
	}
	message := err.Error()
	for _, key := range []string{"COMPUTE_SIDECAR_URL", "STORAGE_SIDECAR_URL"} {
		if !strings.Contains(message, key) {
			t.Fatalf("expected %s in error %q", key, message)
		}
	}
}

func TestValidateRuntimeAcceptsRealDependencies(t *testing.T) {
	cfg := Config{
		StubMode:          false,
		DatabaseURL:       "postgres://example",
		OGRPCURL:          "https://rpc.example",
		ComputeSidecarURL: "http://localhost:3001",
		StorageSidecarURL: "http://localhost:3002",
	}
	if err := cfg.ValidateBackendRuntime(); err != nil {
		t.Fatalf("real backend dependencies should pass: %v", err)
	}
	if err := cfg.ValidateWorkerRuntime(); err != nil {
		t.Fatalf("real worker dependencies should pass: %v", err)
	}
}
