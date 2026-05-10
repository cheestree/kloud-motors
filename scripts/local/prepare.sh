#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

export $(grep -v '^#' .env | xargs)

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ORIGINAL_CSV="${1:-$REPO_ROOT/code/setup/CIS_Automotive_Kaggle_Sample.csv}"
PREPARED_CSV="${2:-$REPO_ROOT/code/setup/dataset_prepared.csv}"
USERS_PREPARED_CSV="$REPO_ROOT/code/setup/users_prepared.csv"
MAX_ROWS="${3:-}"
AUCTION_INIT_SCRIPT="$REPO_ROOT/code/setup/auction-db/init_auction_db.py"
USER_PREP_SCRIPT="$REPO_ROOT/code/setup/user-db/prepare_users.py"
USER_LOAD_SCRIPT="$REPO_ROOT/code/setup/user-db/load_users.py"
SELLER_LOAD_SCRIPT="$REPO_ROOT/code/setup/seller-db/load_sellers.py"
CHAT_INIT_SCRIPT="$REPO_ROOT/code/setup/chat-db/init_chat_db.py"

ROWS_ARG=""
if [ -n "$MAX_ROWS" ]; then
    ROWS_ARG="--rows $MAX_ROWS"
fi

if [ ! -f "$AUCTION_INIT_SCRIPT" ]; then
    echo "Auction init script not found at $AUCTION_INIT_SCRIPT"
    exit 1
fi

if [ ! -f "$USER_PREP_SCRIPT" ]; then
    echo "User preparation script not found at $USER_PREP_SCRIPT"
    exit 1
fi

if [ ! -f "$USER_LOAD_SCRIPT" ]; then
    echo "User load script not found at $USER_LOAD_SCRIPT"
    exit 1
fi

AUCTION_INIT_SCRIPT_NAME="$(basename "$AUCTION_INIT_SCRIPT")"
USER_PREP_SCRIPT_NAME="$(basename "$USER_PREP_SCRIPT")"
USER_LOAD_SCRIPT_NAME="$(basename "$USER_LOAD_SCRIPT")"
SELLER_LOAD_SCRIPT_NAME="$(basename "$SELLER_LOAD_SCRIPT")"
CHAT_INIT_SCRIPT_NAME="$(basename "$CHAT_INIT_SCRIPT")"

echo "Ensuring required DB containers are running..."
docker compose up -d listing-db user-db seller-db auction-db chat-db

echo "Waiting for user-db to be ready..."
until docker exec user-db pg_isready -U ${USER_POSTGRES_USER} -d ${USER_POSTGRES_DB}; do
    sleep 2
done

echo "Waiting for seller-db to be ready..."
until docker exec seller-db pg_isready -U ${SELLER_POSTGRES_USER} -d ${SELLER_POSTGRES_DB}; do
    sleep 2
done

echo "Waiting for auction-db to be ready..."
until docker exec auction-db pg_isready -U ${AUCTION_POSTGRES_USER} -d ${AUCTION_POSTGRES_DB}; do
    sleep 2
done

echo "Waiting for chat-db to be ready..."
until docker exec chat-db pg_isready -U ${CHAT_POSTGRES_USER} -d ${CHAT_POSTGRES_DB}; do
    sleep 2
done

echo "Preparing dataset and initializing databases inside docker run..."
docker run --rm \
    --network host \
    -v "$REPO_ROOT:/workspace" \
    -e FIREBASE_PROJECT_ID=${FIREBASE_PROJECT_ID} \
    -e GOOGLE_APPLICATION_CREDENTIALS="/workspace/$(basename ${GOOGLE_APPLICATION_CREDENTIALS})" \
    -e USER_PYTHON_DATABASE_URL=${USER_PYTHON_DATABASE_URL} \
    -w /workspace/code/setup \
    python:3.12-slim \
    bash -c "pip install pandas faker sqlalchemy python-dotenv psycopg2-binary firebase-admin --quiet && \
                         if [ -f '/workspace/code/setup/$(basename $PREPARED_CSV)' ]; then \
                             echo 'Prepared dataset already exists, skipping prepare_listings.py'; \
                         else \
                             python3 listing-db/prepare_listings.py \
                             --dataset '/workspace/code/setup/$(basename $ORIGINAL_CSV)' \
                             --output '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                             $ROWS_ARG; \
                         fi && \
             python3 user-db/$USER_PREP_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                     --output '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 user-db/$USER_LOAD_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 seller-db/$SELLER_LOAD_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 chat-db/$CHAT_INIT_SCRIPT_NAME && \
             python3 auction-db/$AUCTION_INIT_SCRIPT_NAME"

echo "Dataset prepared at $PREPARED_CSV"

echo "Preparation complete."
