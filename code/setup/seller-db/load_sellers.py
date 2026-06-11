#!/usr/bin/env python3
"""Load prepared users CSV into the sellers table in chunks."""

import argparse
import os
import sys
from typing import Dict, List

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy import BigInteger, MetaData, Table, Text, Column, String, Float, create_engine
from sqlalchemy.dialects.postgresql import insert as pg_insert

load_dotenv("../../.env")

EXPECTED_COLUMNS: List[str] = ["id", "name"]

COLUMN_DTYPES: Dict[str, str] = {
    "id": "Int64",
    "name": "string",
}

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Load prepared mock sellers into PostgreSQL sellers table."
    )
    parser.add_argument("--dataset", default="users_prepared.csv")
    parser.add_argument("--table", default="sellers")
    parser.add_argument("--chunk-size", type=int, default=4096)
    parser.add_argument("--max-rows", type=int, default=None)
    return parser.parse_args()

def define_sellers_table(table_name: str, metadata: MetaData) -> Table:
    return Table(
        table_name,
        metadata,
        Column("id", BigInteger, primary_key=True),
        Column("name", Text),
        Column("seller_type", String(50)),
        Column("contact_info", Text),
        Column("rating", Float),
    )

def sanitize_chunk(chunk: pd.DataFrame) -> pd.DataFrame:
    df = chunk.copy()
    df["id"] = pd.to_numeric(df["id"], errors="coerce").astype("Int64")
    df = df[df["id"].notna()].copy()
    df["id"] = df["id"].astype(int)
    df["name"] = df["name"].astype("string").str.strip()
    df = df.drop_duplicates(subset=["id"], keep="last")
    return df

def upsert_dataframe(df: pd.DataFrame, table: Table, conn) -> int:
    if df.empty:
        return 0

    records = []
    for row in df.to_dict(orient="records"):
        records.append(
            {
                "id": int(row["id"]),
                "name": str(row["name"]),
                "seller_type": "private_seller",
                "contact_info": f"Contact for {row['name']}",
                "rating": 5.0,
            }
        )

    stmt = pg_insert(table).values(records)
    stmt = stmt.on_conflict_do_update(
        index_elements=["id"],
        set_={
            "name": stmt.excluded.name,
            "seller_type": stmt.excluded.seller_type,
            "contact_info": stmt.excluded.contact_info,
            "rating": stmt.excluded.rating,
        },
    )
    conn.execute(stmt)
    return len(records)

def main() -> int:
    args = parse_args()
    database_url = os.getenv("SELLER_PYTHON_DATABASE_URL")
    if not database_url:
        print("SELLER_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    try:
        header = pd.read_csv(args.dataset, nrows=0)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1

    engine = create_engine(database_url)
    metadata = MetaData()
    sellers_table = define_sellers_table(args.table, metadata)

    chunk_iter = pd.read_csv(
        args.dataset,
        chunksize=args.chunk_size,
        dtype=COLUMN_DTYPES,
        low_memory=False,
    )

    with engine.begin() as conn:
        metadata.create_all(conn)

    rows_processed = 0
    rows_upserted = 0
    for chunk in chunk_iter:
        if args.max_rows is not None and rows_processed >= args.max_rows:
            break
        cleaned = sanitize_chunk(chunk)
        with engine.begin() as conn:
            rows_upserted += upsert_dataframe(cleaned, sellers_table, conn)
        rows_processed += len(chunk)
        print(f"Processed {rows_processed} rows, upserted {rows_upserted} sellers...", end="\r", flush=True)

    print(f"\nDone. Total sellers upserted: {rows_upserted}")
    return 0

if __name__ == "__main__":
    sys.exit(main())
