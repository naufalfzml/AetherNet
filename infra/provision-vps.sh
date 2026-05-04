#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this script as root on the target VPS." >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

: "${NODE_SOURCE_GPG_URL:?Set NODE_SOURCE_GPG_URL in the VPS shell}"
: "${NODE_SOURCE_REPO_URL:?Set NODE_SOURCE_REPO_URL in the VPS shell}"
: "${GO_TARBALL_URL:?Set GO_TARBALL_URL in the VPS shell}"
: "${FOUNDRY_INSTALLER_URL:?Set FOUNDRY_INSTALLER_URL in the VPS shell}"

apt-get update
apt-get install -y \
  ca-certificates \
  curl \
  git \
  gnupg \
  nginx \
  docker.io \
  docker-compose-plugin \
  certbot \
  python3-certbot-nginx

install -m 0755 -d /etc/apt/keyrings
curl -fsSL "$NODE_SOURCE_GPG_URL" | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] $NODE_SOURCE_REPO_URL nodistro main" >/etc/apt/sources.list.d/nodesource.list
apt-get update
apt-get install -y nodejs

corepack enable
corepack prepare pnpm@9.15.4 --activate
npm install -g pm2

curl -fsSL "$GO_TARBALL_URL" -o /tmp/go.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
ln -sf /usr/local/go/bin/go /usr/local/bin/go
ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

curl -fsSL "$FOUNDRY_INSTALLER_URL" | bash

systemctl enable --now docker
systemctl enable --now nginx

echo "Provisioning complete. Install Foundry binaries with foundryup from the deploy user shell."
