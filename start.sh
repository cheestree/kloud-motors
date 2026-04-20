#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

echo "Generating protobuf files..."
bash code/services/gen_proto.sh

if [ -z "${SEARCH_HOST_PORT}" ]; then
    if lsof -iTCP:50056 -sTCP:LISTEN >/dev/null 2>&1; then
        SEARCH_HOST_PORT=50156
        export SEARCH_HOST_PORT
        echo "Port 50056 is already in use. Using SEARCH_HOST_PORT=${SEARCH_HOST_PORT} instead."
    fi
fi

docker compose up --build