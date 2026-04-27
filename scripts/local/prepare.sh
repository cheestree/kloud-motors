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
SETUP_AUTH_SCRIPT="$REPO_ROOT/code/setup/setup_auth_db.py"
AUTH_INIT_SCRIPT="$REPO_ROOT/code/setup/auth-db/init_auth_db.py"
AUTH_LOAD_SCRIPT="$REPO_ROOT/code/setup/auth-db/load_auth_users.py"
AUCTION_INIT_SCRIPT="$REPO_ROOT/code/setup/auction-db/init_auction_db.py"
USER_PREP_SCRIPT="$REPO_ROOT/code/setup/user-db/prepare_users.py"
USER_LOAD_SCRIPT="$REPO_ROOT/code/setup/user-db/load_users.py"
SELLER_LOAD_SCRIPT="$REPO_ROOT/code/setup/seller-db/load_sellers.py"
CHAT_INIT_SCRIPT="$REPO_ROOT/code/setup/chat-db/init_chat_db.py"

ROWS_ARG=""
if [ -n "$MAX_ROWS" ]; then
    ROWS_ARG="--rows $MAX_ROWS"
fi

if [ ! -f "$SETUP_AUTH_SCRIPT" ]; then
    echo "User/seller init script not found at $SETUP_AUTH_SCRIPT"
    exit 1
fi

if [ ! -f "$AUTH_INIT_SCRIPT" ]; then
    echo "Auth init script not found at $AUTH_INIT_SCRIPT"
    exit 1
fi

if [ ! -f "$AUTH_LOAD_SCRIPT" ]; then
    echo "Auth load script not found at $AUTH_LOAD_SCRIPT"
    exit 1
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

SETUP_AUTH_SCRIPT_NAME="$(basename "$SETUP_AUTH_SCRIPT")"
AUTH_INIT_SCRIPT_NAME="$(basename "$AUTH_INIT_SCRIPT")"
AUTH_LOAD_SCRIPT_NAME="$(basename "$AUTH_LOAD_SCRIPT")"
AUCTION_INIT_SCRIPT_NAME="$(basename "$AUCTION_INIT_SCRIPT")"
USER_PREP_SCRIPT_NAME="$(basename "$USER_PREP_SCRIPT")"
USER_LOAD_SCRIPT_NAME="$(basename "$USER_LOAD_SCRIPT")"
SELLER_LOAD_SCRIPT_NAME="$(basename "$SELLER_LOAD_SCRIPT")"
CHAT_INIT_SCRIPT_NAME="$(basename "$CHAT_INIT_SCRIPT")"

echo "Ensuring required DB containers are running..."
docker compose up -d listing-db user-db auth-db seller-db auction-db

echo "Waiting for user-db to be ready..."
until docker exec user-db pg_isready -U ${USER_POSTGRES_USER} -d ${USER_POSTGRES_DB}; do
    sleep 2
done

echo "Waiting for auth-db to be ready..."
until docker exec auth-db pg_isready -U ${AUTH_POSTGRES_USER} -d ${AUTH_POSTGRES_DB}; do
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

echo "Preparing dataset and initializing databases inside docker run..."
docker run --rm \
    --network host \
    -v "$REPO_ROOT:/workspace" \
    -w /workspace/code/setup \
    python:3.12-slim \
    bash -c "pip install pandas faker sqlalchemy python-dotenv psycopg2-binary bcrypt --quiet && \
                         if [ -f '/workspace/code/setup/$(basename $PREPARED_CSV)' ]; then \
                             echo 'Prepared dataset already exists, skipping prepare_listings.py'; \
                         else \
                             python3 listing-db/prepare_listings.py \
                             --dataset '/workspace/code/setup/$(basename $ORIGINAL_CSV)' \
                             --output '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                             $ROWS_ARG; \
                         fi && \
             python3 $SETUP_AUTH_SCRIPT_NAME && \
             python3 auth-db/$AUTH_INIT_SCRIPT_NAME && \
             python3 user-db/$USER_PREP_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                     --output '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 auth-db/$AUTH_LOAD_SCRIPT_NAME \
                 --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 user-db/$USER_LOAD_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 seller-db/$SELLER_LOAD_SCRIPT_NAME \
                     --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 chat-db/$CHAT_INIT_SCRIPT_NAME && \
             python3 auction-db/$AUCTION_INIT_SCRIPT_NAME"

echo "Dataset prepared at $PREPARED_CSV"

echo "Preparation complete."
