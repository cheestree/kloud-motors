#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

export $(grep -v '^#' .env | xargs)

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
PREPARED_CSV="${1:-$REPO_ROOT/code/setup/dataset_prepared.csv}"

if [ ! -f "$PREPARED_CSV" ]; then
    echo "Prepared dataset not found at $PREPARED_CSV"
    echo "Run ./prepare.sh first"
    exit 1
fi

echo "Waiting for listing-db to be ready..."
until docker exec listing-db pg_isready -U ${LISTING_POSTGRES_USER} -d ${LISTING_POSTGRES_DB}; do
    sleep 2
done

echo "Loading into database from $PREPARED_CSV..."
python3 "$REPO_ROOT/code/setup/listing-db/load_listings.py" --dataset "$PREPARED_CSV"

echo "Seeding complete."