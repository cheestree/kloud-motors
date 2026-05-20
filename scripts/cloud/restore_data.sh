#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "Usage: ./scripts/cloud/restore_data.sh <database> <gs://backup-uri.sql.gz>"
    exit 1
fi

DATABASE="$1"
BACKUP_URI="$2"
PROJECT_ID="${GCP_PROJECT_ID:-$(gcloud config get-value project 2>/dev/null)}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE_NAME:-cn-db-instance}"

if [[ -z "$PROJECT_ID" ]]; then
    echo "Could not determine the GCP project. Set GCP_PROJECT_ID or run 'gcloud config set project <id>'."
    exit 1
fi

echo "Importing ${BACKUP_URI} into ${DATABASE} on instance ${INSTANCE_NAME}"
gcloud sql import sql "$INSTANCE_NAME" "$BACKUP_URI" \
    --project="$PROJECT_ID" \
    --database="$DATABASE" \
    --quiet

echo "Restore completed successfully."
