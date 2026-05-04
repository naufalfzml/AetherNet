#!/usr/bin/env bash
set -euo pipefail

if [ -f backend/go.mod ]; then
  (cd backend && go run ./cmd/seed)
else
  echo "backend/go.mod not found. Backend seed command is not available yet." >&2
  exit 1
fi
