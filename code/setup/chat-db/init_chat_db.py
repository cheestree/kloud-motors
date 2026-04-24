#!/usr/bin/env python3
import os
import sys
from sqlalchemy import create_engine, text

def main():
    database_url = os.getenv("CHAT_PYTHON_DATABASE_URL")
    if not database_url:
        print("CHAT_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    print("Connecting to chat database...")
    engine = create_engine(database_url)
    
    with engine.begin() as conn:
        conn.execute(text("""
            CREATE TABLE IF NOT EXISTS chat (
                user_id INTEGER,
                listing_id INTEGER,
                make VARCHAR(255) NOT NULL,
                model VARCHAR(255) NOT NULL,
                chat_id UUID NOT NULL DEFAULT gen_random_uuid(),
                PRIMARY KEY (user_id, listing_id, chat_id)
            );
        """))
        print("Table 'chat' verified/created successfully in chat_db!")

    return 0

if __name__ == "__main__":
    sys.exit(main())