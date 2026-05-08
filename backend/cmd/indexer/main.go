package main

import (
	"context"
	"log"
	"math/big"
	"strings"

	"github.com/aethernet-0g/aethernet/backend/domain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/chain"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/config"
	"github.com/aethernet-0g/aethernet/backend/infrastructure/postgres"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Println("chain indexer disabled: DATABASE_URL is empty")
		return
	}
	if cfg.OGRPCURL == "" || isZeroAddress(cfg.INFTRegistry) {
		log.Println("chain indexer disabled: OG_RPC_URL or INFT_REGISTRY_ADDRESS is missing")
		return
	}

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	stateRepo := postgres.IndexerStateRepository{DB: db}
	agentRepo := postgres.AgentRepository{DB: db}
	scanner := chain.LogScanner{RPCURL: cfg.OGRPCURL}

	key := "agent_minted:" + strings.ToLower(cfg.INFTRegistry)
	lastBlock, err := stateRepo.LastBlock(ctx, key)
	if err != nil {
		log.Fatalf("load indexer cursor: %v", err)
	}
	startBlock := decimalBig(cfg.IndexerStartBlock)
	fromBlock := new(big.Int).Add(lastBlock, big.NewInt(1))
	if lastBlock.Sign() == 0 && startBlock.Sign() > 0 {
		fromBlock = startBlock
	}

	latest, err := scanner.LatestBlock(ctx)
	if err != nil {
		log.Fatalf("load latest block: %v", err)
	}
	toBlock := new(big.Int).Sub(latest, decimalBig(cfg.IndexerConfirmations))
	if toBlock.Cmp(fromBlock) < 0 {
		log.Printf("chain indexer idle: from=%s latest=%s confirmations=%s", fromBlock, latest, cfg.IndexerConfirmations)
		return
	}

	log.Printf("chain indexer scanning AgentMinted from=%s to=%s registry=%s", fromBlock, toBlock, cfg.INFTRegistry)
	events, err := scanner.AgentMintedEvents(ctx, cfg.INFTRegistry, fromBlock, toBlock)
	if err != nil {
		log.Fatalf("scan AgentMinted logs: %v", err)
	}
	for _, event := range events {
		agent := domain.Agent{
			ID:                 event.AgentAddress,
			TokenID:            event.TokenID,
			OwnerAddress:       event.OwnerAddress,
			AgentAddress:       event.AgentAddress,
			TreasuryAddress:    event.AgentAddress,
			MetadataPointer:    event.MetadataPointer,
			PersonalitySummary: "Indexed agent " + event.TokenID,
		}
		if err := agentRepo.UpsertAgent(ctx, agent); err != nil {
			log.Fatalf("upsert agent token=%s: %v", event.TokenID, err)
		}
	}
	if err := stateRepo.SaveLastBlock(ctx, key, toBlock); err != nil {
		log.Fatalf("save indexer cursor: %v", err)
	}
	log.Printf("chain indexer complete: indexed=%d cursor=%s", len(events), toBlock)
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
