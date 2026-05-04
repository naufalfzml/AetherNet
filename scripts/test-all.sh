#!/usr/bin/env bash
set -euo pipefail

if [ -f contracts/foundry.toml ]; then
  (cd contracts && forge test)
fi

if [ -f backend/go.mod ]; then
  (cd backend && go test ./...)
fi

pnpm -r test
