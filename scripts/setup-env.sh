#!/usr/bin/env bash
# Gera o .env do projeto. Rode você mesmo: as chaves não passam pelo chat.
#   ./scripts/setup-env.sh [pasta-do-docker-compose-do-arena]
set -euo pipefail
cd "$(dirname "$0")/.."
ARENA_DIR="${1:-$HOME/Downloads/Arena}"

if [[ -f .env ]]; then
  read -r -p ".env já existe. Sobrescrever? [y/N] " ok
  [[ "$ok" == "y" ]] || exit 0
fi

read -r -p "Arena URL [http://localhost:8080]: " url; url="${url:-http://localhost:8080}"
read -r -p "App ID (Settings → Apps): " cid
read -r -s -p "App Secret: " csec; echo
read -r -s -p "API Key do usuário de placar (Settings → Users): " akey; echo
read -r -p "ID do evento (vazio = todos): " ev

mkey="$(cd "$ARENA_DIR" && docker compose exec -T caddy sh -c 'printf %s "$MERCURE_PUBLISHER_JWT_KEY"')"
[[ -n "$mkey" ]] || { echo "Não consegui ler a chave do Mercure no container caddy." >&2; exit 1; }

umask 077
cat > .env <<EOF
ARENA_URL=$url
ARENA_CLIENT_ID=$cid
ARENA_CLIENT_SECRET=$csec
ARENA_API_KEY=$akey
ARENA_EVENT_ID=$ev
MERCURE_HUB_URL=$url/.well-known/mercure
MERCURE_JWT_KEY=$mkey
LISTEN_ADDR=:8095
DATA_DIR=data
WEBHOOK_TOKEN=$(openssl rand -hex 24)
WEBHOOK_TOKEN_ENFORCE=false
RESYNC_SECONDS=15
EOF
echo ".env criado (permissão 600)."
