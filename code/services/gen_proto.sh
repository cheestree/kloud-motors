#!/bin/bash
set -e

SERVICES_ROOT="$(cd "$(dirname "$0")" && pwd)"

docker build -f "$SERVICES_ROOT/proto-gen.Dockerfile" -t proto-gen "$SERVICES_ROOT"
docker create --name proto-gen-container proto-gen
docker cp proto-gen-container:/workspace/shared/. "$SERVICES_ROOT/shared/"
docker cp proto-gen-container:/workspace/listing/proto/. "$SERVICES_ROOT/listing/proto/"
docker cp proto-gen-container:/workspace/search/proto/. "$SERVICES_ROOT/search/proto/"
docker cp proto-gen-container:/workspace/chat/proto/. "$SERVICES_ROOT/chat/proto/"
docker cp proto-gen-container:/workspace/seller/proto/. "$SERVICES_ROOT/seller/proto/"
docker cp proto-gen-container:/workspace/user/proto/. "$SERVICES_ROOT/user/proto/"
docker cp proto-gen-container:/workspace/geographic-maket-insights/proto/. "$SERVICES_ROOT/geographic-maket-insights/proto/"
docker rm proto-gen-container

echo "Protobuf generation complete."