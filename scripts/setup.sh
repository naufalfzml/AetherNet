#!/usr/bin/env bash
set -euo pipefail

pnpm install

if [ -f backend/go.mod ]; then
  (cd backend && go mod download)
fi

if command -v forge >/dev/null 2>&1 && [ -f contracts/foundry.toml ]; then
  (cd contracts && forge install)
fi
