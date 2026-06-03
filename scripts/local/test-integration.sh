#!/bin/bash
set -euo pipefail

TARGET="all"
DO_DOWN=false
DO_DELETE=false

cleanup() {
  if [[ "$DO_DELETE" == "true" ]]; then
    docker compose -f docker-compose.test.yml down --volumes --remove-orphans --rmi local
  elif [[ "$DO_DOWN" == "true" ]]; then
    docker compose -f docker-compose.test.yml down
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    listing|search|geo|geographic-market-insights|marketprice|seller|user|all)
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
      echo "Usage: $0 [listing|search|geo|geographic-market-insights|marketprice|seller|user|all] [--down] [--delete]"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [listing|search|geo|geographic-market-insights|marketprice|seller|user|all] [--down] [--delete]"
      exit 1
      ;;
  esac
done

trap cleanup EXIT

case "$TARGET" in
  listing)
    packages="./listing"
    ;;
  search)
    packages="./search"
    ;;
  geo|geographic-market-insights)
    packages="./geographic-market-insights"
    ;;
  marketprice)
    packages="./marketprice"
    ;;
  seller)
    packages="./seller"
    ;;
  user)
    packages="./user"
    ;;
  all)
    packages="./listing ./search ./geographic-market-insights ./marketprice ./seller ./user"
    ;;
  *)
    echo "Usage: $0 [listing|search|geo|geographic-market-insights|marketprice|seller|user|all]"
    exit 1
    ;;
esac

docker compose -f docker-compose.test.yml up --build -d \
  listing-db-test user-db-test seller-db-test redis-cache-test \
  listing-seed-test seller-seed-test \
  listing-test search-test geo-test marketprice-test seller-test user-test

docker compose -f docker-compose.test.yml run --rm \
  -e TEST_PACKAGES="$packages" \
  test-runner-test
