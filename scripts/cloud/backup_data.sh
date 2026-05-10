#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODE="${1:-all}"
PROJECT_ID="${GCP_PROJECT_ID:-$(gcloud config get-value project 2>/dev/null)}"
REGION="${GCP_REGION:-europe-central2}"
INSTANCE_NAME="${CLOUD_SQL_INSTANCE_NAME:-cn-db-instance}"
BACKUP_BUCKET="${BACKUP_BUCKET:-${PROJECT_ID}-backups}"
TIMESTAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
DATASET_DIR="$ROOT_DIR/code/setup"
DATABASES=(listing_db user_db seller_db auction_db chat_db)

if [[ -z "$PROJECT_ID" ]]; then
    echo "Could not determine the GCP project. Set GCP_PROJECT_ID or run 'gcloud config set project <id>'."
    exit 1
fi

backup_databases() {
    for database in "${DATABASES[@]}"; do
        uri="gs://${BACKUP_BUCKET}/cloudsql/${database}/${TIMESTAMP}.sql.gz"
        echo "Exporting database ${database} to ${uri}"
        gcloud sql export sql "$INSTANCE_NAME" "$uri" \
            --project="$PROJECT_ID" \
            --database="$database" \
            --offload
    done
}

backup_dataset() {
    if [[ ! -d "$DATASET_DIR" ]]; then
        echo "Dataset directory not found: $DATASET_DIR"
        exit 1
    fi

    local uploaded_any=false
    while IFS= read -r file; do
        uploaded_any=true
        destination="gs://${BACKUP_BUCKET}/datasets/${TIMESTAMP}/$(basename "$file")"
        echo "Uploading $(basename "$file") to ${destination}"
        gcloud storage cp "$file" "$destination" --project="$PROJECT_ID"
    done < <(find "$DATASET_DIR" -maxdepth 1 -type f \( -name "*.csv" -o -name "*.json" \))

    if [[ "$uploaded_any" == false ]]; then
        echo "No dataset artifacts found in $DATASET_DIR"
    fi
}

case "$MODE" in
    all)
        backup_databases
        backup_dataset
        ;;
    db)
        backup_databases
        ;;
    dataset)
        backup_dataset
        ;;
    *)
        echo "Usage: ./scripts/cloud/backup_data.sh [all|db|dataset]"
        exit 1
        ;;
esac

echo "Backup completed successfully."
