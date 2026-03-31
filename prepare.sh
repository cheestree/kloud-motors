#!/bin/bash
set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
ORIGINAL_CSV="${1:-$REPO_ROOT/code/setup/CIS_Automotive_Kaggle_Sample.csv}"
PREPARED_CSV="${2:-$REPO_ROOT/code/setup/dataset_prepared.csv}"
MAX_ROWS="${3:-}"

ROWS_ARG=""
if [ -n "$MAX_ROWS" ]; then
    ROWS_ARG="--rows $MAX_ROWS"
fi

echo "Preparing dataset..."
docker run --rm \
    -v "$REPO_ROOT/code/setup:/data" \
    -w /data \
    python:3.12-slim \
    bash -c "pip install pandas faker --quiet && \
             python3 listing-db/prepare_listings.py \
             --dataset '/data/$(basename $ORIGINAL_CSV)' \
             --output '/data/$(basename $PREPARED_CSV)' \
             $ROWS_ARG"

echo "Dataset prepared at $PREPARED_CSV"