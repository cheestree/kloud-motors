#!/usr/bin/env python3
"""Initialize the auth_db with required tables."""

import os
import sys
from dotenv import load_dotenv
from sqlalchemy import create_engine, Column, Integer, Text, MetaData, Table

# Load environment variables from .env file
base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
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

    auth_users = Table(
        "auth_users",
        metadata,
        Column("id", Integer, primary_key=True, autoincrement=True),
        Column("email", Text, unique=True, index=True),
        Column("password", Text),
    )

    try:
        print("Dropping existing auth tables...")
        metadata.drop_all(engine)
        print("Creating auth tables...")
        metadata.create_all(engine)
        print("Tables created successfully!")
    except Exception as e:
        print(f"Error creating tables: {e}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    sys.exit(main())
