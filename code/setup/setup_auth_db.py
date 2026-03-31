#!/usr/bin/env python3
"""Initialize the user and seller databases with required tables."""

import os
import sys
from dotenv import load_dotenv
from sqlalchemy import create_engine, Column, String, Integer, Text, UniqueConstraint, MetaData, Table, Float

# Load environment variables from .env file
base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
dotenv_path = os.path.join(base_dir, ".env")
load_dotenv(dotenv_path)

def main():
    user_database_url = os.getenv("USER_PYTHON_DATABASE_URL")
    seller_database_url = os.getenv("SELLER_PYTHON_DATABASE_URL")
    if not user_database_url:
        print("USER_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1
    if not seller_database_url:
        print("SELLER_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    print(f"Connecting to user database: {user_database_url}")
    user_engine = create_engine(user_database_url)
    user_metadata = MetaData()

    # Define User table
    users = Table(
        "users",
        user_metadata,
        Column("id", Integer, primary_key=True, autoincrement=True),
        Column("name", Text),
        Column("email", Text, unique=True, index=True),
        Column("password", Text),
    )

    # Define Favorite table
    favorites = Table(
        "favorites",
        user_metadata,
        Column("id", Integer, primary_key=True, autoincrement=True),
        Column("user_id", Integer),
        Column("listing_id", Integer),
        UniqueConstraint("user_id", "listing_id", name="idx_user_listing"),
    )

    print(f"Connecting to seller database: {seller_database_url}")
    seller_engine = create_engine(seller_database_url)
    seller_metadata = MetaData()

    # Define Seller table
    sellers = Table(
        "sellers",
        seller_metadata,
        Column("id", Integer, primary_key=True, autoincrement=True),
        Column("name", Text),
        Column("seller_type", String(50)),
        Column("contact_info", Text),
        Column("rating", Float),
    )

    try:
        print("Dropping existing user tables...")
        user_metadata.drop_all(user_engine)
        print("Creating user tables...")
        user_metadata.create_all(user_engine)

        print("Dropping existing seller tables...")
        seller_metadata.drop_all(seller_engine)
        print("Creating seller tables...")
        seller_metadata.create_all(seller_engine)
        print("Tables created successfully!")
    except Exception as e:
        print(f"Error creating tables: {e}", file=sys.stderr)
        return 1

    return 0

if __name__ == "__main__":
    sys.exit(main())
