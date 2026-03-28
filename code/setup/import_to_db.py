import os
import pandas as pd
from sqlalchemy import create_engine
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dataset", required=True, help="The final deduped CSV file")
    args = parser.parse_args()

    db_url = "postgresql://listing_user:listing_password@localhost:5432/listing_db"
    engine = create_engine(db_url)

    print(f"Loading the file '{args.dataset}' into memory...")
    df = pd.read_csv(args.dataset, low_memory=False)

    print("Importing data to the 'listings' table in the Postgres Database...")
    df.to_sql('listings', engine, if_exists='replace', index=False)
    
    print("Success! The 'listings' table has been created and populated.")

if __name__ == "__main__":
    main()
