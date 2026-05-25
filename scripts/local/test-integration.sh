#!/bin/bash
set -euo pipefail

TARGET="all"
DO_DOWN=false
DO_DELETE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    listing|search|all)
      TARGET="$1"
      shift
      ;;
    --down)
      DO_DOWN=true
      shift
      ;;
    --delete)
      DO_DELETE=true
      shift
      ;;
    -h|--help)
      echo "Usage: $0 [listing|search|all] [--down] [--delete]"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [listing|search|all] [--down] [--delete]"
      exit 1
      ;;
  esac
done

case "$TARGET" in
  listing)
    packages="./listing"
    ;;
  search)
    packages="./search"
    ;;
  all)
    packages="./listing ./search"
    ;;
  *)
    echo "Usage: $0 [listing|search|all]"
    exit 1
    ;;
esac

docker compose -f docker-compose.test.yml up --build -d \
  listing-db-test redis-cache-test listing-seed-test listing-test search-test

docker compose -f docker-compose.test.yml run --rm \
  -e TEST_PACKAGES="$packages" \
  test-runner-test

if [[ "$DO_DELETE" == "true" ]]; then
  docker compose -f docker-compose.test.yml down --volumes --remove-orphans --rmi local
elif [[ "$DO_DOWN" == "true" ]]; then
  docker compose -f docker-compose.test.yml down
fi
