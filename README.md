# Auction & MarketPrice Services

Microserviços de Leilão e Análise de Preços implementados em Go com PostgreSQL.

## 1. Setup Inicial
```bash
cp .env.example .env
python3 -m venv venv && source venv/bin/activate
pip install -r requirements.txt
```

## 2. Base de Dados
Inicie os contentores de DB e execute os scripts de inicialização:
```bash
docker compose up -d auction-db listing-db
python code/setup/auction-db/init_auction_db.py
cd code/setup && python listing-db/load_listings.py --dataset data/dataset_prepared.csv
```

## 3. Execução
```bash
docker compose up --build auction marketprice listing
```

**Portas:**
- **Auction**: 50056 (gRPC) / 8080 (WS)
- **MarketPrice**: 50055 (gRPC)
- **Listing**: 50052 (gRPC)