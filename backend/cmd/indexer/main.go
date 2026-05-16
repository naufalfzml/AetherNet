package main

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/chain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/postgres"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/storage"
	"github.com/aethernet-0g/aethernet/backend/usecase"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Println("chain indexer disabled: DATABASE_URL is empty")
		return
	}
	if cfg.OGRPCURL == "" || isZeroAddress(cfg.INFTRegistry) {
		log.Println("chain indexer disabled: OG_RPC_URL or INFT_REGISTRY_ADDRESS is missing")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	stateRepo := postgres.IndexerStateRepository{DB: db}
	agentRepo := postgres.AgentRepository{DB: db}
	metadataRepo := postgres.AgentMetadataRepository{DB: db}
	scanner := chain.LogScanner{RPCURL: cfg.OGRPCURL}
	var storageClient usecase.ZGStorageClient
	if cfg.HasStorageSidecar() {
		storageClient = storage.NewHTTPClient(cfg.StorageSidecarURL)
		log.Printf("chain indexer metadata recovery enabled via storage sidecar %s", cfg.StorageSidecarURL)
	} else {
		log.Printf("chain indexer metadata recovery disabled: STORAGE_SIDECAR_URL is missing")
	}
	key := "agent_minted:" + strings.ToLower(cfg.INFTRegistry)
	interval := pollInterval()

	log.Printf("chain indexer starting: registry=%s interval=%s", cfg.INFTRegistry, interval)

	for {
		if err := runOnce(ctx, cfg, scanner, stateRepo, agentRepo, metadataRepo, storageClient, key); err != nil {
			log.Printf("indexer cycle error: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Println("chain indexer stopped")
			return
		case <-time.After(interval):
		}
	}
}

func runOnce(
	ctx context.Context,
	cfg config.Config,
	scanner chain.LogScanner,
	stateRepo postgres.IndexerStateRepository,
	agentRepo postgres.AgentRepository,
	metadataRepo postgres.AgentMetadataRepository,
	storageClient usecase.ZGStorageClient,
	key string,
) error {
	lastBlock, err := stateRepo.LastBlock(ctx, key)
	if err != nil {
		return err
	}
	startBlock := decimalBig(cfg.IndexerStartBlock)
	fromBlock := new(big.Int).Add(lastBlock, big.NewInt(1))
	if lastBlock.Sign() == 0 && startBlock.Sign() > 0 {
		fromBlock = startBlock
	}

	latest, err := scanner.LatestBlock(ctx)
	if err != nil {
		return err
	}
	toBlock := new(big.Int).Sub(latest, decimalBig(cfg.IndexerConfirmations))
	if toBlock.Cmp(fromBlock) < 0 {
		return nil
	}

	log.Printf("chain indexer scanning AgentMinted from=%s to=%s", fromBlock, toBlock)
	events, err := scanner.AgentMintedEvents(ctx, cfg.INFTRegistry, fromBlock, toBlock)
	if err != nil {
		return err
	}
	for _, event := range events {
		personalitySummary := ""
		if storageClient != nil && strings.TrimSpace(event.MetadataPointer) != "" {
			metadata, err := hydrateMetadata(ctx, storageClient, event.MetadataPointer)
			if err != nil {
				log.Printf("indexer metadata hydrate failed token=%s pointer=%s: %v", event.TokenID, event.MetadataPointer, err)
			} else {
				metadata.MetadataPointer = event.MetadataPointer
				if err := metadataRepo.UpsertMetadata(ctx, metadata); err != nil {
					return err
				}
				personalitySummary = strings.TrimSpace(metadata.PersonalitySummary)
				if personalitySummary == "" {
					personalitySummary = strings.TrimSpace(metadata.Prompt)
				}
			}
		}
		agent := domain.Agent{
			ID:                 event.AgentAddress,
			TokenID:            event.TokenID,
			OwnerAddress:       event.OwnerAddress,
			AgentAddress:       event.AgentAddress,
			TreasuryAddress:    event.AgentAddress,
			MetadataPointer:    event.MetadataPointer,
			PersonalitySummary: personalitySummary,
		}
		if err := agentRepo.UpsertAgent(ctx, agent); err != nil {
			return err
		}
	}
	if err := stateRepo.SaveLastBlock(ctx, key, toBlock); err != nil {
		return err
	}
	if len(events) > 0 {
		log.Printf("chain indexer indexed=%d cursor=%s", len(events), toBlock)
	}
	return nil
}

func pollInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("INDEXER_POLL_SECONDS"))
	if raw == "" {
		return 5 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 5 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func isZeroAddress(address string) bool {
	address = strings.ToLower(strings.TrimSpace(address))
	return address == "" || address == "0x0000000000000000000000000000000000000000"
}

func decimalBig(value string) *big.Int {
	parsed, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok {
		return big.NewInt(0)
	}
	return parsed
}

func hydrateMetadata(ctx context.Context, storageClient usecase.ZGStorageClient, metadataPointer string) (domain.AgentMetadata, error) {
	bytes, err := storageClient.Fetch(ctx, metadataPointer)
	if err != nil {
		return domain.AgentMetadata{}, err
	}

	var payload struct {
		Prompt             string `json:"prompt"`
		PersonalitySummary string `json:"personalitySummary"`
	}
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return domain.AgentMetadata{}, err
	}

	metadata := domain.AgentMetadata{
		MetadataPointer:    metadataPointer,
		Prompt:             strings.TrimSpace(payload.Prompt),
		PersonalitySummary: strings.TrimSpace(payload.PersonalitySummary),
	}
	if metadata.PersonalitySummary == "" {
		metadata.PersonalitySummary = metadata.Prompt
	}
	return metadata, nil
}
