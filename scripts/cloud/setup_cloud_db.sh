#!/bin/bash
set -e

INSTANCE_NAME="cn-project-491618:europe-central2:cn-db-instance"
DB_USER=${DB_USER:-"postgres"}
DB_PASS=${DB_PASS:-"uma_password_forte_aqui"}
DB_HOST="host.docker.internal"
DB_PORT="5432"

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ORIGINAL_CSV="$REPO_ROOT/code/setup/CIS_Automotive_Kaggle_Sample.csv"
PREPARED_CSV="$REPO_ROOT/code/setup/dataset_prepared.csv"
USERS_PREPARED_CSV="$REPO_ROOT/code/setup/users_prepared.csv"

echo "Starting Cloud SQL Proxy..."
"$REPO_ROOT/cloud-sql-proxy" "$INSTANCE_NAME" --address 0.0.0.0 --port "$DB_PORT" > "$REPO_ROOT/proxy.log" 2>&1 &
PROXY_PID=$!

trap 'echo "Stopping Cloud SQL Proxy..."; kill $PROXY_PID 2>/dev/null' EXIT

echo "Waiting 10 seconds for the proxy to open the local connection..."
sleep 10

echo "Proxy running (PID: $PROXY_PID). Starting data preparation..."
docker run --rm \
    --add-host host.docker.internal:host-gateway \
    -e AUTH_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/auth_db" \
    -e USER_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/user_db" \
    -e SELLER_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/seller_db" \
    -e AUCTION_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/auction_db" \
    -e LISTING_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/listing_db" \
    -e CHAT_PYTHON_DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/chat_db" \
    -v "$REPO_ROOT:/workspace" \
    -w /workspace/code/setup \
    python:3.12-slim \
    bash -c "pip install pandas faker sqlalchemy python-dotenv psycopg2-binary bcrypt --quiet && \
             if [ -f '/workspace/code/setup/$(basename $PREPARED_CSV)' ]; then \
                 echo 'Dataset already prepared. Skipping initial cleaning.'; \
             else \
                 echo 'Preparing main dataset...' && \
                 python3 listing-db/prepare_listings.py \
                 --dataset '/workspace/code/setup/$(basename $ORIGINAL_CSV)' \
                 --output '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                 --rows 1000; \
             fi && \
             echo 'Creating tables...' && \
             python3 auth-db/init_auth_db.py && \
             python3 setup_auth_db.py && \
             python3 user-db/prepare_users.py \
                 --dataset '/workspace/code/setup/$(basename $PREPARED_CSV)' \
                 --output '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 auth-db/load_auth_users.py \
                 --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 user-db/load_users.py \
                 --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 seller-db/load_sellers.py \
                 --dataset '/workspace/code/setup/$(basename $USERS_PREPARED_CSV)' && \
             python3 auction-db/init_auction_db.py && \
             python3 chat-db/init_chat_db.py && \
             echo 'Loading vehicles (listings) to the Cloud...' && \
             python3 listing-db/load_listings.py \
                 --dataset '/workspace/code/setup/$(basename $PREPARED_CSV)'"

echo "All done! The databases in the Cloud have been prepared and populated."