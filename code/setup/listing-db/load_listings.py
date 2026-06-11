#!/usr/bin/env python3
"""Load a prepared CSV (output of prepare_csv_for_load.py) into PostgreSQL in chunks."""

import argparse
import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Set, Tuple

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy import (
    BigInteger,
    Boolean,
    Column,
    DateTime,
    Float,
    ForeignKey,
    MetaData,
    SmallInteger,
    Table,
    Text,
    UniqueConstraint,
    create_engine,
    inspect,
    select,
    text,
)
from sqlalchemy.dialects.postgresql import insert as pg_insert

load_dotenv("../../.env")

# Columns the loader expects in the input CSV.
EXPECTED_COLUMNS = [
    "vin",
    "stock_num",
    "first_seen",
    "last_seen",
    "ask_price",
    "msrp",
    "mileage",
    "is_new",
    "is_sold",
    "color",
    "interior_color",
    "brand_name",
    "model_name",
    "dealer_id",
    "model_year",
    "body_class",
    "fuel_type_primary",
    "transmission_style",
    "transmission_speeds",
    "drive_type",
    "engine_cylinders",
    "engine_hp",
    "displacement_l",
    "doors",
    "trim",
    "turbo",
    "electrification_level",
    "district",
    "city",
    "country",
    "state"
]

# Columns that require a lookup table (normalised dimension).
LOOKUP_COLUMNS = {
    "brand_name":           "brand",
    "fuel_type_primary":    "fuel_type",
    "transmission_style":   "transmission",
    "drive_type":           "drive_type",
    "body_class":           "body_class",
    "electrification_level": "electrification_level",
}

COLUMN_DTYPES: Dict[str, str] = {
    "ask_price":          "Int64",
    "msrp":               "Int64",
    "mileage":            "Int64",
    "is_new":             "boolean",
    "is_sold":            "boolean",
    "dealer_id":          "Int64",
    "model_year":         "Int64",
    "engine_cylinders":   "Float64",
    "engine_hp":          "Float64",
    "displacement_l":     "Float64",
    "doors":              "Int64",
    "transmission_speeds": "Int64",
    # district, city, country are text, so no dtype needed (default is object)
}


@dataclass
class LookupCache:
    brand: Dict[str, int] = field(default_factory=dict)
    fuel_type: Dict[str, int] = field(default_factory=dict)
    transmission: Dict[str, int] = field(default_factory=dict)
    drive_type: Dict[str, int] = field(default_factory=dict)
    body_class: Dict[str, int] = field(default_factory=dict)
    electrification_level: Dict[str, int] = field(default_factory=dict)
    model: Dict[Tuple[int, str], int] = field(default_factory=dict)


def normalize_text(value: Any) -> "str | None":
    # Covers None, float NaN, pd.NA, pd.NaT, np.nan.
    try:
        if value is None or pd.isna(value):
            return None
    except (TypeError, ValueError):
        pass
    text = str(value).strip()
    return text or None


def ensure_lookup_values(
    conn,
    table: Table,
    value_col: Column,
    values: Set[str],
    cache: Dict[str, int],
) -> None:
    if not values:
        return
    missing = values - cache.keys()
    if not missing:
        return
    conn.execute(
        pg_insert(table)
        .values([{value_col.name: v} for v in sorted(missing)])
        .on_conflict_do_nothing(index_elements=[value_col.name])
    )
    rows = conn.execute(select(table.c.id, value_col).where(value_col.in_(missing))).all()
    cache.update({name: row_id for row_id, name in rows})


def ensure_model_values(
    conn,
    model_table: Table,
    pairs: Set[Tuple[int, str]],
    cache: Dict[Tuple[int, str], int],
) -> None:
    if not pairs:
        return
    valid = {(b, m) for b, m in pairs if isinstance(b, int) and isinstance(m, str)}
    if not valid:
        return
    missing = [(b, m) for b, m in valid if (b, m) not in cache]
    if not missing:
        return
    conn.execute(
        pg_insert(model_table)
        .values([{"brand_id": b, "name": m} for b, m in sorted(missing)])
        .on_conflict_do_nothing(index_elements=["brand_id", "name"])
    )
    brand_ids = sorted({b for b, _ in missing})
    model_names = sorted({m for _, m in missing})
    rows = conn.execute(
        select(model_table.c.id, model_table.c.brand_id, model_table.c.name).where(
            model_table.c.brand_id.in_(brand_ids),
            model_table.c.name.in_(model_names),
        )
    ).all()
    cache.update({(brand_id, name): row_id for row_id, brand_id, name in rows})


def build_dimension_tables(metadata: MetaData) -> Dict[str, Table]:
    def simple_lookup(table_name: str) -> Table:
        return Table(
            table_name,
            metadata,
            Column("id", BigInteger, primary_key=True),
            Column("name", Text, nullable=False, unique=True),
        )

    tables = {name: simple_lookup(name) for name in LOOKUP_COLUMNS.values()}
    tables["model"] = Table(
        "model",
        metadata,
        Column("id", BigInteger, primary_key=True),
        Column("brand_id", BigInteger, ForeignKey("brand.id"), nullable=False),
        Column("name", Text, nullable=False),
        UniqueConstraint("brand_id", "name", name="uq_model_brand_name"),
    )
    return tables


def create_fact_table(table_name: str, metadata: MetaData) -> Table:
    return Table(
        table_name,
        metadata,
        Column("id", BigInteger, primary_key=True, autoincrement=True),
        Column("vin",                  Text,       unique=True, nullable=False),
        Column("stock_num",            Text),
        Column("first_seen",           DateTime),
        Column("last_seen",            DateTime),
        Column("ask_price",            BigInteger),
        Column("msrp",                 BigInteger),
        Column("mileage",              BigInteger),
        Column("is_new",               Boolean),
        Column("is_sold",              Boolean, nullable=False, server_default=text("false")),
        Column("color",                Text),
        Column("interior_color",       Text),
        Column("dealer_id",            BigInteger),
        Column("model_year",           SmallInteger),
        Column("engine_cylinders",     Float),
        Column("engine_hp",            Float),
        Column("displacement_l",       Float),
        Column("doors",                SmallInteger),
        Column("transmission_speeds",  SmallInteger),
        Column("trim",                 Text),
        Column("turbo",                Text),
        Column("brand_id",             BigInteger, ForeignKey("brand.id")),
        Column("model_id",             BigInteger, ForeignKey("model.id")),
        Column("fuel_type_id",         BigInteger, ForeignKey("fuel_type.id")),
        Column("transmission_id",      BigInteger, ForeignKey("transmission.id")),
        Column("drive_type_id",        BigInteger, ForeignKey("drive_type.id")),
        Column("body_class_id",        BigInteger, ForeignKey("body_class.id")),
        Column("electrification_level_id", BigInteger, ForeignKey("electrification_level.id")),
        Column("district",             Text),
        Column("city",                 Text),
        Column("country",              Text),
        Column("state",                Text),
    )


def transform_chunk(
    chunk: pd.DataFrame,
    conn,
    dim_tables: Dict[str, Table],
    cache: LookupCache,
) -> pd.DataFrame:
    df = chunk.copy()

    if "is_sold" not in df.columns:
        df["is_sold"] = False
    else:
        df["is_sold"] = df["is_sold"].fillna(False)

    # Normalise all lookup source columns up front.
    normalised: Dict[str, pd.Series] = {
        src_col: df[src_col].map(normalize_text)
        for src_col in LOOKUP_COLUMNS
        if src_col in df.columns
    }

    # Populate dimension tables and caches.
    for src_col, table_name in LOOKUP_COLUMNS.items():
        if src_col not in normalised:
            continue
        table = dim_tables[table_name]
        values: Set[str] = {v for v in normalised[src_col] if isinstance(v, str)}
        cache_dict: Dict[str, int] = getattr(cache, table_name)
        ensure_lookup_values(conn, table, table.c.name, values, cache_dict)

    model_pairs: Set[Tuple[int, str]] = set()
    if "brand_name" in normalised and "model_name" in df.columns:
        model_series = df["model_name"].map(normalize_text)
        for brand_name, model_name in zip(normalised["brand_name"], model_series):
            if not isinstance(brand_name, str) or not isinstance(model_name, str):
                continue
            brand_id = cache.brand.get(brand_name)
            if isinstance(brand_id, int):
                model_pairs.add((brand_id, model_name))
        ensure_model_values(conn, dim_tables["model"], model_pairs, cache.model)

    # Replace source columns with FK id columns.
    for src_col, table_name in LOOKUP_COLUMNS.items():
        fk_col = f"{table_name}_id"
        if src_col in normalised:
            cache_dict = getattr(cache, table_name)
            df[fk_col] = normalised[src_col].map(cache_dict).astype("Int64")
        df = df.drop(columns=[src_col], errors="ignore")

    if "brand_name" in normalised and "model_name" in df.columns:
        model_ids: List[int | None] = []
        for brand_name, model_name in zip(normalised["brand_name"], df.get("model_name", pd.Series())):
            if brand_name is None or model_name is None:
                model_ids.append(None)
                continue
            brand_id = cache.brand.get(brand_name)
            model_ids.append(cache.model.get((brand_id, normalize_text(model_name))) if brand_id else None)
        df["model_id"] = pd.array(model_ids, dtype="Int64")
    df = df.drop(columns=["model_name"], errors="ignore")

    return df


def upsert_dataframe(df: pd.DataFrame, table: Table, conn) -> None:
    records = []
    for row in df.to_dict(orient="records"):
        records.append({
            str(k): (None if isinstance(v, float) and pd.isna(v) else
                     None if hasattr(v, "_value") and pd.isna(v) else v)
            for k, v in row.items()
        })
    if not records:
        return
    stmt = pg_insert(table).values(records)
    stmt = stmt.on_conflict_do_update(
        index_elements=["vin"],
        set_={col: stmt.excluded[col] for col in df.columns if col != "vin"},
    )
    conn.execute(stmt)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Load a prepared CSV into PostgreSQL in chunks."
    )
    parser.add_argument("--dataset", default="dataset_prepared.csv")
    parser.add_argument("--table", default="automotive_data")
    parser.add_argument("--chunk-size", type=int, default=16_384)
    parser.add_argument("--max-rows", type=int, default=None)
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    database_url = os.getenv("LISTING_PYTHON_DATABASE_URL")
    if not database_url:
        database_url = "postgresql://listing_user:listing_password@localhost:5432/listing_db"
    if not database_url:
        print("LISTING_PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    if args.max_rows is not None and args.max_rows <= 0:
        print("--max-rows must be a positive integer.", file=sys.stderr)
        return 1

    # Validate that the CSV has the expected columns before doing any DB work.
    try:
        header = pd.read_csv(args.dataset, nrows=0)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1

    missing_cols = [c for c in EXPECTED_COLUMNS if c not in header.columns]
    if missing_cols:
        print(
            f"Input CSV is missing expected columns: {', '.join(missing_cols)}\n"
            "Run prepare_listings.py first.",
            file=sys.stderr,
        )
        return 1

    engine = create_engine(database_url)
    metadata = MetaData()
    dim_tables = build_dimension_tables(metadata)

    chunk_iter = pd.read_csv(
        args.dataset,
        chunksize=args.chunk_size,
        dtype=COLUMN_DTYPES,
        low_memory=False,
    )

    rows_processed = 0

    with engine.begin() as conn:
        metadata.create_all(conn)  # creates dim tables
        if not inspect(conn).has_table(args.table):
            fact_table = create_fact_table(args.table, metadata)
            metadata.create_all(conn)  # creates fact table
        else:
            fact_table = Table(args.table, MetaData(), autoload_with=conn)
            conn.execute(text(f"ALTER TABLE {args.table} ADD COLUMN IF NOT EXISTS is_sold BOOLEAN NOT NULL DEFAULT false"))

    cache = LookupCache()

    for chunk in chunk_iter:
        if args.max_rows is not None:
            remaining = args.max_rows - rows_processed
            if remaining <= 0:
                break
            if len(chunk) > remaining:
                chunk = chunk.head(remaining)

        with engine.begin() as conn:
            transformed = transform_chunk(chunk, conn, dim_tables, cache)
            upsert_dataframe(transformed, fact_table, conn)

        rows_processed += len(chunk)
        print(f"Processed {rows_processed} rows...", end="\r", flush=True)

    print(f"\nDone. Total rows upserted: {rows_processed}")

    sql_path = Path(__file__).parent / "create_indexes.sql"
    if sql_path.exists():
        print("Creating performance indexes...")
        with engine.begin() as conn:
            conn.execute(text(sql_path.read_text()))
        print("Indexes created successfully.")
    else:
        print(f"Warning: index script not found at {sql_path}", file=sys.stderr)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
