#!/bin/bash
set -e

SERVICES_ROOT="$(cd "$(dirname "$0")" && pwd)"

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc not found in PATH."
    echo "Install it with: brew install protobuf (macOS) or apt-get install protobuf-compiler (Linux)"
    exit 1
fi

# Check if Go protobuf plugins are installed
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

echo "Generating protobuf files..."

protoc \
    --proto_path="$SERVICES_ROOT" \
    --go_out="$SERVICES_ROOT" \
    --go-grpc_out="$SERVICES_ROOT" \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    "$SERVICES_ROOT/shared/shared.proto" \
    "$SERVICES_ROOT/listing/proto/listing.proto" \
    "$SERVICES_ROOT/search/proto/search.proto" \
    "$SERVICES_ROOT/chat/proto/chat.proto" \
    "$SERVICES_ROOT/seller/proto/seller.proto" \
    "$SERVICES_ROOT/user/proto/user.proto" \
    "$SERVICES_ROOT/auth/proto/auth.proto" \
    "$SERVICES_ROOT/geographic-market-insights/proto/geo-market-insights.proto" \
    "$SERVICES_ROOT/auction/proto/auction.proto"

echo "Protobuf generation complete (local)."
