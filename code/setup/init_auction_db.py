#!/usr/bin/env python3
"""Initialize the auction-db with required tables."""

import os
import sys
from dotenv import load_dotenv
from sqlalchemy import create_engine, Column, String, Float, Integer, DateTime, Boolean, ForeignKey, MetaData, Table, func

# Load environment variables from .env file
base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
dotenv_path = os.path.join(base_dir, ".env")
load_dotenv(dotenv_path)

def main():
    database_url = os.getenv("AUCTION_PYTHON_DATABASE_URL")
    if not database_url:
        print("AUCTION_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    print(f"Connecting to database: {database_url}")
    engine = create_engine(database_url)
    metadata = MetaData()

    auctions = Table(
        "auctions",
        metadata,
        Column("id", String, primary_key=True),
        Column("listing_id", String, nullable=False, index=True),
        Column("seller_id", String, nullable=False, index=True),
        Column("starting_price", Float, nullable=False),
        Column("current_price", Float, nullable=True),
        Column("reserve_price", Float, nullable=True),
        Column("status", String(20), default="ACTIVE", index=True), # ACTIVE, COMPLETED, CANCELLED
        Column("end_time", DateTime, nullable=False),
        Column("winner_user_id", String, nullable=True),
        Column("created_at", DateTime, server_default=func.now()),
        Column("reserve_met", Boolean, default=False),
        Column("total_bids", Integer, default=0),
    )

    bids = Table(
        "bids",
        metadata,
        Column("id", String, primary_key=True),
        Column("auction_id", String, ForeignKey("auctions.id", ondelete="CASCADE"), nullable=False, index=True),
        Column("bidder_id", String, nullable=False, index=True),
        Column("bid_amount", Float, nullable=False),
        Column("timestamp", DateTime, server_default=func.now()),
    )

    try:
        print("Dropping existing tables...")
        metadata.drop_all(engine)
        print("Creating tables...")
        metadata.create_all(engine)
        print("Tables created successfully!")
    except Exception as e:
        print(f"Error creating tables: {e}", file=sys.stderr)
        return 1

    return 0

if __name__ == "__main__":
    sys.exit(main())
