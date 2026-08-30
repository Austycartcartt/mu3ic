#!/usr/bin/env bash
# Nightly Postgres backup. Audio/artwork durability is Cloudflare R2's job
# (enable bucket versioning); this only protects the metadata database.
#
# Install as a host cron entry, e.g.:
#   15 3 * * * /opt/mu3ic/deploy/backup.sh >> /var/log/mu3ic-backup.log 2>&1
set -euo pipefail

# Directory holding this script — the compose project directory.
cd "$(dirname "${BASH_SOURCE[0]}")"

# Load POSTGRES_* for the pg_dump invocation below.
set -a
# shellcheck disable=SC1091
source .env
set +a

BACKUP_DIR="${MU3IC_BACKUP_DIR:-/var/backups/mu3ic}"
RETENTION_DAYS="${MU3IC_BACKUP_RETENTION_DAYS:-14}"
mkdir -p "$BACKUP_DIR"

stamp="$(date +%F-%H%M)"
out="$BACKUP_DIR/mu3ic-$stamp.dump"

docker compose -f docker-compose.yml exec -T db \
	pg_dump -U "${POSTGRES_USER:-mu3ic}" -Fc "${POSTGRES_DB:-mu3ic}" > "$out"

echo "wrote $out ($(du -h "$out" | cut -f1))"

# Prune old dumps.
find "$BACKUP_DIR" -name 'mu3ic-*.dump' -mtime "+$RETENTION_DAYS" -delete

# Optional off-box copy (needs rclone configured with an "r2backups" remote):
#   rclone copy "$BACKUP_DIR" r2backups:mu3ic-backups
