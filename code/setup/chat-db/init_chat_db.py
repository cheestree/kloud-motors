#!/usr/bin/env python3
import os
import sys
from dotenv import load_dotenv
from sqlalchemy import create_engine, text

# Load environment variables from .env file
base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
dotenv_path = os.path.join(base_dir, ".env")
load_dotenv(dotenv_path)

def main():
    database_url = os.getenv("CHAT_PYTHON_DATABASE_URL")
    if not database_url:
        print("CHAT_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    print("Connecting to chat database...")
    engine = create_engine(database_url)
    
    with engine.begin() as conn:
        # Force drop to ensure the schema is updated (brand instead of make)
        conn.execute(text("DROP TABLE IF EXISTS chat CASCADE;"))
        conn.execute(text("""
            CREATE TABLE chat (
                user_id BIGINT,
                listing_id BIGINT,
                brand VARCHAR(255) NOT NULL,
                model VARCHAR(255) NOT NULL,
                chat_id UUID NOT NULL DEFAULT gen_random_uuid(),
                PRIMARY KEY (user_id, listing_id, chat_id)
            );
        """))
        print("Table 'chat' verified/created successfully in chat_db!")

    return 0

if __name__ == "__main__":
    sys.exit(main())
