#!/bin/bash
set -e

if [ ! -f .env ]; then
    echo ".env not found, copying from .env.example..."
    cp .env.example .env
fi

docker compose up listing listing-db search