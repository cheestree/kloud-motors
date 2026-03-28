#!/usr/bin/env python3
"""Initialize the auth-db with required tables."""

import os
import sys
from dotenv import load_dotenv
from sqlalchemy import create_engine, Column, String, Integer, Text, BigInteger, DateTime, Boolean, ForeignKey, UniqueConstraint, MetaData, Table, Float

# Load environment variables from .env file
base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
dotenv_path = os.path.join(base_dir, ".env")
load_dotenv(dotenv_path)

def main():
    database_url = os.getenv("AUTH_PYTHON_DATABASE_URL")
    if not database_url:
        print("AUTH_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    print(f"Connecting to database: {database_url}")
    engine = create_engine(database_url)
    metadata = MetaData()

    # Define User table (consistent with GORM model in main.go)
    users = Table(
        "users",
        metadata,
        Column("id", String, primary_key=True),
        Column("name", Text),
        Column("email", Text, unique=True, index=True),
        Column("password", Text),
        Column("is_seller", Boolean, default=False),
        Column("seller_type", String(50)),
        Column("contact_info", Text),
        Column("rating", Float),
    )

    # Define Favorite table
    favorites = Table(
        "favorites",
        metadata,
        Column("id", Integer, primary_key=True, autoincrement=True),
        Column("user_id", String), # In a real app, this would be a ForeignKey
        Column("listing_id", Text),
        UniqueConstraint("user_id", "listing_id", name="idx_user_listing"),
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
