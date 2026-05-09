package main

import (
	"context"
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
	scanner := chain.LogScanner{RPCURL: cfg.OGRPCURL}
	key := "agent_minted:" + strings.ToLower(cfg.INFTRegistry)
	interval := pollInterval()

	log.Printf("chain indexer starting: registry=%s interval=%s", cfg.INFTRegistry, interval)

	for {
		if err := runOnce(ctx, cfg, scanner, stateRepo, agentRepo, key); err != nil {
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
