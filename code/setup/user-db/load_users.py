#!/usr/bin/env python3
"""Load prepared users CSV into the auth users table in chunks."""

import argparse
import os
import sys
from typing import Dict, List

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy import Integer, MetaData, Table, Text, Column, create_engine
from sqlalchemy.dialects.postgresql import insert as pg_insert

load_dotenv("../../.env")

EXPECTED_COLUMNS: List[str] = ["id", "name", "email"]

COLUMN_DTYPES: Dict[str, str] = {
	"id": "Int64",
	"name": "string",
	"email": "string",
}


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Load prepared mock users into PostgreSQL users table."
	)
	parser.add_argument("--dataset", default="users_prepared.csv")
	parser.add_argument("--table", default="users")
	parser.add_argument("--chunk-size", type=int, default=4096)
	parser.add_argument("--max-rows", type=int, default=None)
	return parser.parse_args()


def define_users_table(table_name: str, metadata: MetaData) -> Table:
	return Table(
		table_name,
		metadata,
		Column("id", Integer, primary_key=True),
		Column("name", Text),
		Column("email", Text, unique=True, index=True),
	)


def sanitize_chunk(chunk: pd.DataFrame) -> pd.DataFrame:
	df = chunk.copy()

	df["id"] = pd.to_numeric(df["id"], errors="coerce").astype("Int64")
	df = df[df["id"].notna()].copy()
	df["id"] = df["id"].astype(int)

	for col in ["name", "email"]:
		df[col] = df[col].astype("string").str.strip()

	# Email is unique in schema; skip rows without valid email.
	df = df[df["email"].notna() & (df["email"] != "")].copy()

	# Keep one row per id in this chunk and preserve the last entry.
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
				"name": None if pd.isna(row["name"]) else str(row["name"]),
				"email": None if pd.isna(row["email"]) else str(row["email"]),
			}
		)

	stmt = pg_insert(table).values(records)
	stmt = stmt.on_conflict_do_update(
		index_elements=["id"],
		set_={
			"name": stmt.excluded.name,
			"email": stmt.excluded.email,
		},
	)
	conn.execute(stmt)
	return len(records)


def main() -> int:
	args = parse_args()

	if args.chunk_size <= 0:
		print("--chunk-size must be greater than zero.", file=sys.stderr)
		return 1
	if args.max_rows is not None and args.max_rows <= 0:
		print("--max-rows must be a positive integer.", file=sys.stderr)
		return 1

	database_url = os.getenv("USER_PYTHON_DATABASE_URL")
	if not database_url:
		print("USER_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
		return 1

	try:
		header = pd.read_csv(args.dataset, nrows=0)
	except FileNotFoundError:
		print(f"Dataset not found: {args.dataset}", file=sys.stderr)
		return 1

	missing_cols = [c for c in EXPECTED_COLUMNS if c not in header.columns]
	if missing_cols:
		print(
			f"Input CSV is missing expected columns: {', '.join(missing_cols)}\n"
			"Run user-db/prepare_users.py first.",
			file=sys.stderr,
		)
		return 1

	engine = create_engine(database_url)
	metadata = MetaData()
	users_table = define_users_table(args.table, metadata)

	rows_processed = 0
	rows_upserted = 0

	chunk_iter = pd.read_csv(
		args.dataset,
		chunksize=args.chunk_size,
		dtype=COLUMN_DTYPES,
		low_memory=False,
	)

	with engine.begin() as conn:
		metadata.create_all(conn)

	for chunk in chunk_iter:
		if args.max_rows is not None:
			remaining = args.max_rows - rows_processed
			if remaining <= 0:
				break
			if len(chunk) > remaining:
				chunk = chunk.head(remaining)

		cleaned = sanitize_chunk(chunk)
		with engine.begin() as conn:
			rows_upserted += upsert_dataframe(cleaned, users_table, conn)

		rows_processed += len(chunk)
		print(
			f"Processed {rows_processed} rows, upserted {rows_upserted} users...",
			end="\r",
			flush=True,
		)

	print(f"\nDone. Total users upserted: {rows_upserted}")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())
