#!/bin/bash
# CryptoJackal Backup Script
# Run via cron: 0 2 * * * /home/twadelij/testapp/CryptoJackal/scripts/backup.sh

set -e

APP_DIR="/home/twadelij/testapp/CryptoJackal"
DATA_DIR="/home/twadelij/.cryptojackal"
BACKUP_DIR="/home/twadelij/backups/cryptojackal"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# Backup SQLite database
if [ -f "$DATA_DIR/cryptojackal.db" ]; then
    cp "$DATA_DIR/cryptojackal.db" "$BACKUP_DIR/cryptojackal_$DATE.db"
    echo "Database backed up: $BACKUP_DIR/cryptojackal_$DATE.db"
fi

# Backup config (env file)
if [ -f "$APP_DIR/.env" ]; then
    cp "$APP_DIR/.env" "$BACKUP_DIR/env_$DATE"
    echo "Config backed up: $BACKUP_DIR/env_$DATE"
fi

# Keep only last 7 backups
ls -t "$BACKUP_DIR"/cryptojackal_*.db | tail -n +8 | xargs -r rm
ls -t "$BACKUP_DIR"/env_* | tail -n +8 | xargs -r rm

echo "Backup complete. Files in $BACKUP_DIR:"
ls -la "$BACKUP_DIR"
