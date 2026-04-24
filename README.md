# computacao-nuvem-2025

This project uses Go microservices with PostgreSQL databases and a REST gateway.

## Setup

1. Clone the repository and enter it.
2. Download the dataset from Kaggle ([this link](https://www.kaggle.com/datasets/cisautomotiveapi/large-car-dataset)) and place it under code/setup. You can download the dataset from .

## Run locally
To run the application entirely on your local machine using Docker:

1. Prepare dataset artifacts:

```bash
./scripts/local/prepare.sh
```

1. Start services and databases:

```bash
./scripts/local/start.sh
```

1. Seed listing data:

```bash
./scripts/local/seed.sh
```

## Run on Kubernetes
Read the sections carefully, as some scripts only need to be run once.

1. Bootstrap Environment(Run once per machine)

```bash
./scripts/cloud/bootstrap_env.sh
```


1. Deploy all Kubernetes manifests:

```bash
./k8s.sh up
```

1. Deploy including ingress:

```bash
./k8s.sh up --with-ingress
```

1. Check deployment status:

```bash
./k8s.sh status
```

1. Remove deployment:

```bash
./k8s.sh down
```

## Gateway REST Endpoints

Base URL: `http://localhost:8080` (local) if in Cloud replace with your Ingress IP

- `GET /api/listings/search`
- `GET /api/listings/compare`
- `GET /api/listings/{id}`
- `POST /api/chat/open`
- `GET /api/chat/{chat_id}`
- `GET /api/market/insights/aggregates`
- `GET /api/market/price-comparison`
- `GET /api/listings/stats/by-location`
- `GET /api/market/average-price`
- `GET /api/auctions`
- `POST /api/auctions`
- `GET /api/auctions/{auction_id}`
- `DELETE /api/auctions/{auction_id}`
- `POST /api/auctions/{auction_id}/bid`
- `GET /api/auctions/{auction_id}/bids`
- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/users/me/favorites`
- `POST /api/users/me/favorites/{listing_id}`
- `DELETE /api/users/me/favorites/{listing_id}`
- `GET /api/sellers/{seller_id}`

## gRPC Endpoints

- Search: `localhost:50056` (or `localhost:50156` if `50056` is busy and `./start.sh` applies fallback)
- Listing: `localhost:50054`
- Auth: `localhost:50053`
- User: `localhost:50058`
- Seller: `localhost:50057`
- MarketPrice: `localhost:50055`
- Auction: `localhost:50051`

## WebSocket Endpoints

- Auction WS: `ws://localhost:8081/ws/auction/{auctionID}`
- Chat WS route (service route): `/ws/chat/{chatID}`

See full contract in `api/API.yaml`.
