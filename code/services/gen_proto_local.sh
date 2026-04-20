#!/bin/bash
set -e


SERVICES_ROOT="$(cd "$(dirname "$0")" && pwd)"

if ! command -v proto-gen &> /dev/null; then
    echo "Error: proto-gen binary not found in PATH."
    exit 1
fi

proto-gen --out_shared "$SERVICES_ROOT/shared/" \
    --out_listing "$SERVICES_ROOT/listing/proto/" \
    --out_search "$SERVICES_ROOT/search/proto/" \
    --out_chat "$SERVICES_ROOT/chat/proto/" \
    --out_seller "$SERVICES_ROOT/seller/proto/" \
    --out_user "$SERVICES_ROOT/user/proto/" \
    --out_auth "$SERVICES_ROOT/auth/proto/" \
    --out_geo "$SERVICES_ROOT/geographic-maket-insights/proto/" \
    --out_auction "$SERVICES_ROOT/auction/proto/"

echo "Protobuf generation complete (local binary)."
