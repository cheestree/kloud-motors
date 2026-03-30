#!/bin/bash
set -e

# Resolve the services/ root regardless of where the script is called from
SERVICES_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Services root: $SERVICES_ROOT"

# Shared proto — output sits next to the .proto file
protoc \
  --proto_path="$SERVICES_ROOT" \
  --go_out="$SERVICES_ROOT" \
  --go-grpc_out="$SERVICES_ROOT" \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  shared/shared.proto

# Listing proto
protoc \
  --proto_path="$SERVICES_ROOT" \
  --go_out="$SERVICES_ROOT" \
  --go-grpc_out="$SERVICES_ROOT" \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  listing/proto/listing.proto

# Search proto
protoc \
  --proto_path="$SERVICES_ROOT" \
  --go_out="$SERVICES_ROOT" \
  --go-grpc_out="$SERVICES_ROOT" \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  search/proto/search.proto

echo "Protobuf generation complete."