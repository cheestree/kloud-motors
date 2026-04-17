#!/usr/bin/env python3
"""Prepare a users CSV from unique seller/dealer IDs in the prepared listings dataset."""

import argparse
import hashlib
import sys
from pathlib import Path
from typing import Dict, List, Set

import pandas as pd

EXPECTED_SOURCE_COLUMN = "dealer_id"
OUTPUT_COLUMNS: List[str] = ["id", "name", "email", "password"]


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Create mock users from unique dealer IDs in a prepared listings CSV."
	)
	parser.add_argument("--dataset", default="dataset_prepared.csv")
	parser.add_argument("--output", default="users_prepared.csv")
	parser.add_argument("--chunk-size", type=int, default=16384)
	parser.add_argument("--max-users", type=int, default=None)
	return parser.parse_args()


def stable_password_for_id(user_id: int) -> str:
	# Deterministic mock password so repeated runs are reproducible.
	digest = hashlib.sha256(f"mock-user-{user_id}".encode("utf-8")).hexdigest()
	return f"pw_{digest[:16]}"


def normalize_id(value) -> int | None:
	if pd.isna(value):
		return None
	text = str(value).strip()
	if not text:
		return None
	try:
		return int(float(text))
	except ValueError:
		return None


def to_user_row(user_id: int) -> Dict[str, object]:
	return {
		"id": user_id,
		"name": f"Seller {user_id}",
		"email": f"seller{user_id}@mock.local",
		"password": stable_password_for_id(user_id),
	}


def main() -> int:
	args = parse_args()

	if args.chunk_size <= 0:
		print("--chunk-size must be greater than zero.", file=sys.stderr)
		return 1
	if args.max_users is not None and args.max_users <= 0:
		print("--max-users must be a positive integer.", file=sys.stderr)
		return 1

	try:
		header_df = pd.read_csv(args.dataset, nrows=0)
	except FileNotFoundError:
		print(f"Dataset not found: {args.dataset}", file=sys.stderr)
		return 1

	if EXPECTED_SOURCE_COLUMN not in header_df.columns:
		print(
			f"Input CSV must contain '{EXPECTED_SOURCE_COLUMN}'. Run prepare_listings.py first.",
			file=sys.stderr,
		)
		return 1

	seen_ids: Set[int] = set()
	prepared_rows: List[Dict[str, object]] = []

	for chunk in pd.read_csv(args.dataset, chunksize=args.chunk_size, low_memory=False):
		if args.max_users is not None and len(seen_ids) >= args.max_users:
			break

		for raw in chunk[EXPECTED_SOURCE_COLUMN]:
			user_id = normalize_id(raw)
			if user_id is None:
				continue
			if user_id in seen_ids:
				continue

			seen_ids.add(user_id)
			prepared_rows.append(to_user_row(user_id))

			if args.max_users is not None and len(seen_ids) >= args.max_users:
				break

	output_path = Path(args.output)
	out_df = pd.DataFrame(prepared_rows, columns=OUTPUT_COLUMNS)
	out_df.to_csv(output_path, index=False)

	print(f"Prepared {len(out_df)} users -> {output_path}")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())
