#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# One-Command Production Bootstrap & Redeploy Script (Ubuntu 22.04 / 24.04 VPS)
# Usage:
#   chmod +x scripts/deploy-vps.sh
#   ./scripts/deploy-vps.sh [optional-domain, e.g. oms.fransalwan.my.id]
# ==============================================================================

DOMAIN_ARG="${1:-}"
ENV_FILE=".env.prod"

echo "==> [1/4] Checking Docker & Docker Compose availability..."
if ! command -v docker >/dev/null 2>&1; then
  echo "    Docker not found. Installing official Docker Engine..."
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker "$USER" || true
fi

echo "==> [2/4] Preparing production environment ($ENV_FILE)..."
if [ ! -f "$ENV_FILE" ]; then
  AES_KEY="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)"
  DB_PASS="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24)"
  REDIS_PASS="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24)"
  HMAC_KEY="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)"
  WEBHOOK_KEY="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)"

  cat > "$ENV_FILE" <<EOF
APP_DOMAIN=${DOMAIN_ARG:-:80}
DB_USER=spg_prod
DB_PASSWORD=${DB_PASS}
DB_NAME=secure_payment_gateway
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
REDIS_PASSWORD=${REDIS_PASS}
AES_SECRET_KEY=${AES_KEY}
HMAC_SECRET_KEY=${HMAC_KEY}
HMAC_TIMESTAMP_TOLERANCE_SECONDS=300
WEBHOOK_SECRET=${WEBHOOK_KEY}
WEBHOOK_TIMESTAMP_TOLERANCE_SECONDS=300
IDEMPOTENCY_TTL_SECONDS=120
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.0-flash
EOF
  chmod 600 "$ENV_FILE"
  echo "    Generated $ENV_FILE with fresh cryptographic secrets (32-byte AES-256 + random DB/Redis/HMAC keys)."
elif [ -n "$DOMAIN_ARG" ]; then
  sed -i "s|^APP_DOMAIN=.*|APP_DOMAIN=${DOMAIN_ARG}|" "$ENV_FILE"
  echo "    Updated APP_DOMAIN=${DOMAIN_ARG} in $ENV_FILE."
fi

echo "==> [3/4] Building & starting decoupled production stack (postgres + redis + oms-api + oms-worker + caddy)..."
docker compose --env-file "$ENV_FILE" -f docker-compose.prod.yml up -d --build --remove-orphans

echo "==> [4/4] Verifying container health..."
docker compose --env-file "$ENV_FILE" -f docker-compose.prod.yml ps

CURRENT_DOMAIN="$(grep '^APP_DOMAIN=' "$ENV_FILE" | cut -d'=' -f2-)"
echo ""
if [ "$CURRENT_DOMAIN" = ":80" ]; then
  echo "Deployment active on HTTP Port 80! Open http://<YOUR_VPS_IP> in your browser."
  echo "Once your domain DNS A-record points to this VPS, enable HTTPS with:"
  echo "  ./scripts/deploy-vps.sh oms.yourdomain.my.id"
else
  echo "Deployment active with Automatic HTTPS! Open: https://${CURRENT_DOMAIN}"
fi

