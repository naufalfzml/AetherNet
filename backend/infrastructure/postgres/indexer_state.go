package postgres

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
)

type IndexerStateRepository struct {
	DB *sql.DB
}

func (r IndexerStateRepository) LastBlock(ctx context.Context, key string) (*big.Int, error) {
	var value string
	err := r.DB.QueryRowContext(ctx, `
		SELECT last_block::text
		FROM indexer_state
		WHERE key = $1
	`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return big.NewInt(0), nil
	}
	if err != nil {
		return nil, err
	}
	block, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, sql.ErrNoRows
	}
	return block, nil
}

func (r IndexerStateRepository) SaveLastBlock(ctx context.Context, key string, block *big.Int) error {
	if block == nil {
		block = big.NewInt(0)
	}
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO indexer_state (key, last_block, updated_at)
		VALUES ($1, $2::numeric, now())
		ON CONFLICT (key) DO UPDATE SET
			last_block = EXCLUDED.last_block,
			updated_at = EXCLUDED.updated_at
	`, key, block.String())
	return err
}
