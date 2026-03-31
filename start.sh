#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

echo "Generating protobuf files..."
bash code/services/gen_proto.sh

docker compose up listing listing-db search