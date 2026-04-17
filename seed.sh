#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

export $(grep -v '^#' .env | xargs)

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
PREPARED_CSV="${1:-$REPO_ROOT/code/setup/dataset_prepared.csv}"
LISTING_LOAD_SCRIPT="$REPO_ROOT/code/setup/listing-db/load_listings.py"

if [ ! -f "$PREPARED_CSV" ]; then
    echo "Prepared dataset not found at $PREPARED_CSV"
    echo "Run ./prepare.sh first"
    exit 1
fi

if [ ! -f "$LISTING_LOAD_SCRIPT" ]; then
    echo "Listing load script not found at $LISTING_LOAD_SCRIPT"
    exit 1
fi

echo "Waiting for listing-db to be ready..."
until docker compose exec -T listing-db pg_isready -U ${LISTING_POSTGRES_USER} -d ${LISTING_POSTGRES_DB}; do
    sleep 2
done

echo "Loading into database from $PREPARED_CSV..."
docker run --rm \
    --network host \
    -v "$REPO_ROOT:/workspace" \
    -w /workspace/code/setup \
    python:3.12-slim \
    bash -c "pip install pandas sqlalchemy python-dotenv psycopg2-binary --quiet && \
             python3 listing-db/load_listings.py --dataset '/workspace/code/setup/$(basename "$PREPARED_CSV")'"

echo "Seeding complete."