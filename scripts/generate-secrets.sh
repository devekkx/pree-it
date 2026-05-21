#!/usr/bin/env bash
#
# Generates all Docker secret files.
# Run once before `docker compose up` on a fresh environment.
#
# Usage:
#   chmod +x scripts/generate-secrets.sh
#   ./scripts/generate-secrets.sh

set -euo pipefail

SECRETS_DIR="$(cd "$(dirname "$0")/.." && pwd)/secrets"

generate_secret() {
  local bytes="${1:-32}"
  openssl rand -hex "$bytes"
}

write_secret() {
  local name="$1"
  local value="$2"
  local file="$SECRETS_DIR/$name"

  if [ -f "$file" ]; then
    echo "  [skip]  $name (already exists)"
  else
    printf '%s' "$value" > "$file"
    chmod 600 "$file"
    echo "  [ok]    $name"
  fi
}

echo ""
echo "Generating secrets - $SECRETS_DIR"
echo "────────────────────────────────────────────"

mkdir -p "$SECRETS_DIR"

write_secret "postgres_user"            "preeit"
write_secret "postgres_db"              "preeit"
write_secret "postgres_password"        "$(generate_secret 24)"
write_secret "redis_password"           "$(generate_secret 24)"
write_secret "jwt_secret"               "$(generate_secret 64)"
write_secret "grafana_admin_password"   "$(generate_secret 16)"

echo "────────────────────────────────────────────"
echo "Done. Files written to ./secrets/ (mode 600)"
echo ""
echo "Next: docker compose up -d"
echo ""