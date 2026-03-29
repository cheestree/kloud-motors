import os
import pandas as pd
from sqlalchemy import create_engine, text
from datetime import datetime, timedelta
import uuid

def main():
    db_url = os.environ.get("AUCTION_PYTHON_DATABASE_URL", "postgresql://auction_user:auction_password@localhost:5436/auction_db")
    engine = create_engine(db_url)

    now = datetime.utcnow()

    print(f"A ligar à Base de Dados: {db_url}")

    # Drop and recreate tables
    with engine.connect() as conn:
        print("A apagar tabelas existentes...")
        conn.execute(text("DROP TABLE IF EXISTS bids CASCADE"))
        conn.execute(text("DROP TABLE IF EXISTS auctions CASCADE"))

        print("A criar tabela 'auctions'...")
        conn.execute(text("""
            CREATE TABLE auctions (
                id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                listing_id    TEXT NOT NULL,
                seller_id     TEXT NOT NULL,
                starting_price DOUBLE PRECISION NOT NULL,
                current_price  DOUBLE PRECISION,
                reserve_price  DOUBLE PRECISION,
                status        TEXT NOT NULL DEFAULT 'ACTIVE',
                end_time      TIMESTAMPTZ NOT NULL,
                winner_user_id TEXT,
                created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                reserve_met   BOOLEAN NOT NULL DEFAULT FALSE,
                total_bids    INTEGER NOT NULL DEFAULT 0
            )
        """))

        print("A criar tabela 'bids'...")
        conn.execute(text("""
            CREATE TABLE bids (
                id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                auction_id  UUID NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
                bidder_id   TEXT NOT NULL,
                bid_amount  DOUBLE PRECISION NOT NULL,
                timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )
        """))

        conn.commit()
        print("Tabelas criadas com sucesso!")

    # Generate auction IDs
    auction_ids = [str(uuid.uuid4()) for _ in range(3)]

    # Mock Data: Auctions
    mock_auctions = [
        {
            "id": auction_ids[0],
            "listing_id": "1",
            "seller_id": "100",
            "starting_price": 1000.0,
            "current_price": None,
            "reserve_price": 1500.0,
            "status": "ACTIVE",
            "end_time": now + timedelta(days=5),
            "winner_user_id": None,
            "created_at": now,
            "reserve_met": False,
            "total_bids": 0
        },
        {
            "id": auction_ids[1],
            "listing_id": "2",
            "seller_id": "101",
            "starting_price": 500.0,
            "current_price": 600.0,
            "reserve_price": 550.0,
            "status": "CLOSED",
            "end_time": now - timedelta(days=1),
            "winner_user_id": "200",
            "created_at": now - timedelta(days=10),
            "reserve_met": True,
            "total_bids": 2
        },
        {
            "id": auction_ids[2],
            "listing_id": "3",
            "seller_id": "102",
            "starting_price": 50000.0,
            "current_price": 51000.0,
            "reserve_price": 60000.0,
            "status": "ACTIVE",
            "end_time": now + timedelta(days=10),
            "winner_user_id": None,
            "created_at": now - timedelta(days=2),
            "reserve_met": False,
            "total_bids": 1
        }
    ]

    # Mock Data: Bids
    mock_bids = [
        # Two bids for Auction #2
        {
            "id": str(uuid.uuid4()),
            "auction_id": auction_ids[1],
            "bidder_id": "105",
            "bid_amount": 550.0,
            "timestamp": now - timedelta(days=5)
        },
        {
            "id": str(uuid.uuid4()),
            "auction_id": auction_ids[1],
            "bidder_id": "200",  # Winner of auction #2
            "bid_amount": 600.0,
            "timestamp": now - timedelta(days=1, hours=2)
        },
        # One bid for Auction #3
        {
            "id": str(uuid.uuid4()),
            "auction_id": auction_ids[2],
            "bidder_id": "300",
            "bid_amount": 51000.0,
            "timestamp": now - timedelta(days=1)
        }
    ]

    df_auctions = pd.DataFrame(mock_auctions)
    df_bids = pd.DataFrame(mock_bids)

    print("A inserir mock data na tabela 'auctions'...")
    df_auctions.to_sql('auctions', engine, if_exists='append', index=False)

    print("A inserir mock data na tabela 'bids'...")
    df_bids.to_sql('bids', engine, if_exists='append', index=False)

    print("Sucesso! Tabelas recriadas e dados mock inseridos.")
    print(f"  Auctions: {len(mock_auctions)}")
    print(f"  Bids: {len(mock_bids)}")

if __name__ == "__main__":
    main()
