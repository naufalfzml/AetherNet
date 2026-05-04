#!/usr/bin/env bash
set -euo pipefail

: "${OG_RPC_URL:?Set OG_RPC_URL before deploying contracts}"
: "${PRIVATE_KEY:?Set PRIVATE_KEY before deploying contracts}"

if [ ! -f contracts/foundry.toml ]; then
  echo "contracts/foundry.toml not found. Run the Foundry setup task first." >&2
  exit 1
fi

mkdir -p deployments
(cd contracts && forge script script/Deploy.s.sol --rpc-url "$OG_RPC_URL" --private-key "$PRIVATE_KEY" --broadcast)
