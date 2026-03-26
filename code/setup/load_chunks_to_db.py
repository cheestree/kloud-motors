#!/usr/bin/env python3
"""Load a CSV file in chunks and insert selected columns into PostgreSQL."""

import argparse
from dataclasses import dataclass, field
import os
import sys
from typing import Any, Dict, List, Set, Tuple
from dotenv import load_dotenv
load_dotenv("../../.env")

import pandas as pd
from sqlalchemy import (
    BigInteger,
    Boolean,
    Integer,
    DateTime,
    ForeignKey,
    Float,
    MetaData,
    Table,
    Text,
    Column,
    UniqueConstraint,
    create_engine,
    inspect,
    select,
)
from sqlalchemy.dialects.postgresql import insert as pg_insert

DEFAULT_COLUMNS = [
    "vin",
    "stockNum",
    "firstSeen",
    "lastSeen",
    "askPrice",
    "msrp",
    "mileage",
    "isNew",
    "brandName",
    "modelName",
    "vf_ModelYear",
    "vf_BodyClass",
    "vf_FuelTypePrimary",
    "vf_TransmissionStyle",
    "vf_DriveType",
    "vf_EngineCylinders",
    "vf_EngineHP",
    "color",
    "interiorColor",
    "dealerID",
]

# Explicit dtypes to prevent pandas from guessing wrong types across chunks.
COLUMN_DTYPES: Dict[str, str] = {
    "askPrice": "Int64",
    "msrp": "Int64",
    "mileage": "Int64",
    "isNew": "boolean",
    "vf_ModelYear": "Int64",
    "vf_EngineCylinders": "Float64",
    "vf_EngineHP": "Float64",
    "dealerID": "Int64",
}

BRAND_COLUMN = "brandName"
MODEL_COLUMN = "modelName"
FUEL_COLUMN = "vf_FuelTypePrimary"
TRANSMISSION_COLUMN = "vf_TransmissionStyle"
DRIVE_TYPE_COLUMN = "vf_DriveType"
BODY_CLASS_COLUMN = "vf_BodyClass"

# Caches to avoid redundant database queries for lookup values.
@dataclass
class LookupCache:
    brands: Dict[str, int] = field(default_factory=dict)
    fuel_types: Dict[str, int] = field(default_factory=dict)
    transmissions: Dict[str, int] = field(default_factory=dict)
    drive_types: Dict[str, int] = field(default_factory=dict)
    body_classes: Dict[str, int] = field(default_factory=dict)
    models: Dict[Tuple[int, str], int] = field(default_factory=dict)

def normalize_text(value: Any) -> str | None:
    if value is None:
        return None
    if pd.isna(value):
        return None
    text = str(value).strip()
    return text if text else None

def ensure_lookup_values(
    conn,
    table: Table,
    value_column: Column,
    values: Set[str],
    cache: Dict[str, int],
) -> Dict[str, int]:
    if not values:
        return cache

    missing = values - cache.keys()
    if not missing:
        return cache

    conn.execute(
        pg_insert(table)
        .values([{value_column.name: value} for value in sorted(missing)])
        .on_conflict_do_nothing(index_elements=[value_column.name])
    )

    rows = conn.execute(
        select(table.c.id, value_column).where(value_column.in_(missing))
    ).all()
    cache.update({value: item_id for item_id, value in rows})
    return cache


def ensure_model_values(
    conn,
    model_table: Table,
    model_pairs: Set[Tuple[int, str]],
    cache: Dict[Tuple[int, str], int],
) -> Dict[Tuple[int, str], int]:
    if not model_pairs:
        return cache

    valid_pairs = [
        (brand_id, model_name)
        for brand_id, model_name in model_pairs
        if isinstance(brand_id, int) and isinstance(model_name, str)
    ]
    if not valid_pairs:
        return cache

    missing_pairs = [(b, m) for b, m in valid_pairs if (b, m) not in cache]
    if not missing_pairs:
        return cache

    conn.execute(
        pg_insert(model_table)
        .values(
            [
                {"brand_id": brand_id, "name": model_name}
                for brand_id, model_name in sorted(missing_pairs)
            ]
        )
        .on_conflict_do_nothing(index_elements=["brand_id", "name"])
    )

    missing_brand_ids = sorted({b for b, _ in missing_pairs})
    missing_model_names = sorted({m for _, m in missing_pairs})
    rows = conn.execute(
        select(model_table.c.id, model_table.c.brand_id, model_table.c.name).where(
            model_table.c.brand_id.in_(missing_brand_ids),
            model_table.c.name.in_(missing_model_names),
        )
    ).all()
    cache.update({(brand_id, model_name): item_id for item_id, brand_id, model_name in rows})
    return cache


def transform_chunk(
    chunk: pd.DataFrame,
    conn,
    brand_table: Table,
    model_table: Table,
    fuel_type_table: Table,
    transmission_table: Table,
    drive_type_table: Table,
    body_class_table: Table,
    cache: LookupCache,
) -> pd.DataFrame:
    transformed = chunk.copy()

    brand_series = transformed[BRAND_COLUMN].map(normalize_text)
    fuel_series = transformed[FUEL_COLUMN].map(normalize_text)
    model_series = transformed[MODEL_COLUMN].map(normalize_text)
    transmission_series = transformed[TRANSMISSION_COLUMN].map(normalize_text)
    drive_type_series = transformed[DRIVE_TYPE_COLUMN].map(normalize_text)
    body_class_series = transformed[BODY_CLASS_COLUMN].map(normalize_text)

    ensure_lookup_values(
        conn, brand_table, brand_table.c.name,
        {v for v in brand_series if isinstance(v, str) and v},
        cache.brands,
    )
    ensure_lookup_values(
        conn, fuel_type_table, fuel_type_table.c.name,
        {v for v in fuel_series if isinstance(v, str) and v},
        cache.fuel_types,
    )
    ensure_lookup_values(
        conn, transmission_table, transmission_table.c.name,
        {v for v in transmission_series if isinstance(v, str) and v},
        cache.transmissions,
    )
    ensure_lookup_values(
        conn, drive_type_table, drive_type_table.c.name,
        {v for v in drive_type_series if isinstance(v, str) and v},
        cache.drive_types,
    )
    ensure_lookup_values(
        conn, body_class_table, body_class_table.c.name,
        {v for v in body_class_series if isinstance(v, str) and v},
        cache.body_classes,
    )

    transformed["brand_id"] = brand_series.map(cache.brands)
    transformed["fuel_type_id"] = fuel_series.map(cache.fuel_types)
    transformed["transmission_id"] = transmission_series.map(cache.transmissions)
    transformed["drive_type_id"] = drive_type_series.map(cache.drive_types)
    transformed["body_class_id"] = body_class_series.map(cache.body_classes)

    model_pairs: Set[Tuple[int, str]] = set()
    for brand_name, model_name in zip(brand_series, model_series):
        if brand_name is None or model_name is None:
            continue
        brand_id = cache.brands.get(brand_name)
        if brand_id is not None:
            model_pairs.add((brand_id, model_name))

    ensure_model_values(conn, model_table, model_pairs, cache.models)

    model_ids: List[int | None] = []
    for brand_name, model_name in zip(brand_series, model_series):
        if brand_name is None or model_name is None:
            model_ids.append(None)
            continue
        brand_id = cache.brands.get(brand_name)
        if brand_id is None:
            model_ids.append(None)
            continue
        model_ids.append(cache.models.get((brand_id, model_name)))

    transformed["model_id"] = pd.Series(model_ids, index=transformed.index)
    transformed["brand_id"] = transformed["brand_id"].astype("Int64")
    transformed["model_id"] = transformed["model_id"].astype("Int64")
    transformed["fuel_type_id"] = transformed["fuel_type_id"].astype("Int64")
    transformed["transmission_id"] = transformed["transmission_id"].astype("Int64")
    transformed["drive_type_id"] = transformed["drive_type_id"].astype("Int64")
    transformed["body_class_id"] = transformed["body_class_id"].astype("Int64")

    transformed = transformed.drop(
        columns=[
            BRAND_COLUMN,
            MODEL_COLUMN,
            FUEL_COLUMN,
            TRANSMISSION_COLUMN,
            DRIVE_TYPE_COLUMN,
            BODY_CLASS_COLUMN,
        ]
    )
    return transformed


def create_fact_table(table_name: str, metadata: MetaData) -> Table:
    # Hardcoded schema, avoids dtype instability across chunks.
    return Table(
        table_name,
        metadata,
        Column("vin", Text, primary_key=True),
        Column("stockNum", Text),
        Column("firstSeen", DateTime),
        Column("lastSeen", DateTime),
        Column("askPrice", BigInteger),
        Column("msrp", BigInteger),
        Column("mileage", BigInteger),
        Column("isNew", Boolean),
        Column("vf_ModelYear", BigInteger),
        Column("vf_EngineCylinders", Float),
        Column("vf_EngineHP", Float),
        Column("color", Text),
        Column("interiorColor", Text),
        Column("dealerID", BigInteger),
        Column("brand_id", Integer, ForeignKey("brand.id")),
        Column("model_id", Integer, ForeignKey("model.id")),
        Column("fuel_type_id", Integer, ForeignKey("fuel_type.id")),
        Column("transmission_id", Integer, ForeignKey("transmission.id")),
        Column("drive_type_id", Integer, ForeignKey("drive_type.id")),
        Column("body_class_id", Integer, ForeignKey("body_class.id")),
    )


def upsert_dataframe(df: pd.DataFrame, table: Table, conn) -> None:
    deduped = df.drop_duplicates(subset=["vin"], keep="last")
    records = deduped.where(pd.notna(deduped), None).to_dict(orient="records")
    if not records:
        return
    stmt = pg_insert(table).values(records)
    stmt = stmt.on_conflict_do_update(
        index_elements=["vin"],
        set_={col: stmt.excluded[col] for col in deduped.columns if col != "vin"},
    )
    conn.execute(stmt)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Load CSV data in chunks and insert into PostgreSQL."
    )
    parser.add_argument(
        "--dataset",
        default="CIS_Automotive_Kaggle_Sample.csv",
        help="Path to the CSV dataset.",
    )
    parser.add_argument(
        "--table",
        default="automotive_data",
        help="Target table name.",
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=16384,
        help="Number of rows per chunk.",
    )
    parser.add_argument(
        "--num-columns",
        type=int,
        default=None,
        help="Number of columns to load from DEFAULT_COLUMNS.",
    )
    parser.add_argument(
        "--max-rows",
        type=int,
        default=None,
        help="Maximum number of rows to process.",
    )
    return parser.parse_args()


def resolve_columns(dataset: str, num_columns: int | None) -> List[str]:
    header = pd.read_csv(dataset, nrows=0)
    available = list(header.columns)

    requested = DEFAULT_COLUMNS
    if num_columns is not None:
        requested = requested[:num_columns]

    selected = [col for col in requested if col in available]
    if selected:
        return selected

    if num_columns is not None:
        return available[:num_columns]
    return available


def main() -> int:
    args = parse_args()
    database_url = os.getenv("PYTHON_DATABASE_URL")
    if not database_url:
        print("PYTHON_DATABASE_URL is not set.", file=sys.stderr)
        return 1

    try:
        selected_columns = resolve_columns(args.dataset, args.num_columns)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1

    if not selected_columns:
        print("No columns selected.", file=sys.stderr)
        return 1

    # Filter COLUMN_DTYPES to only columns actually being loaded.
    active_dtypes = {k: v for k, v in COLUMN_DTYPES.items() if k in selected_columns}

    engine = create_engine(database_url)

    try:
        chunk_iter = pd.read_csv(
            args.dataset,
            usecols=selected_columns,
            chunksize=args.chunk_size,
            dtype=active_dtypes,
        )
    except ValueError as exc:
        print(f"Invalid column selection: {exc}", file=sys.stderr)
        return 1

    try:
        first_chunk = next(chunk_iter)
    except StopIteration:
        print("Dataset is empty.", file=sys.stderr)
        return 1

    remaining_rows = args.max_rows
    if remaining_rows is not None and remaining_rows <= 0:
        print("--max-rows must be a positive integer.", file=sys.stderr)
        return 1

    if remaining_rows is not None and len(first_chunk) > remaining_rows:
        first_chunk = first_chunk.head(remaining_rows)
        remaining_rows = 0
    elif remaining_rows is not None:
        remaining_rows -= len(first_chunk)

    metadata = MetaData()
    brand_table = Table(
        "brand",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("name", Text, nullable=False, unique=True),
    )
    fuel_type_table = Table(
        "fuel_type",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("name", Text, nullable=False, unique=True),
    )
    transmission_table = Table(
        "transmission",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("name", Text, nullable=False, unique=True),
    )
    drive_type_table = Table(
        "drive_type",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("name", Text, nullable=False, unique=True),
    )
    body_class_table = Table(
        "body_class",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("name", Text, nullable=False, unique=True),
    )
    model_table = Table(
        "model",
        metadata,
        Column("id", Integer, primary_key=True),
        Column("brand_id", Integer, ForeignKey("brand.id"), nullable=False),
        Column("name", Text, nullable=False),
        UniqueConstraint("brand_id", "name", name="uq_model_brand_name"),
    )

    with engine.begin() as conn:
        metadata.create_all(conn)

        cache = LookupCache()

        first_transformed = transform_chunk(
            first_chunk,
            conn,
            brand_table,
            model_table,
            fuel_type_table,
            transmission_table,
            drive_type_table,
            body_class_table,
            cache,
        )

        inspector = inspect(conn)
        if not inspector.has_table(args.table):
            fact_table = create_fact_table(args.table, metadata)
            metadata.create_all(conn)
        else:
            fact_table = Table(args.table, MetaData(), autoload_with=conn)
            existing_columns = {column.name for column in fact_table.columns}
            missing_columns = [
                col for col in first_transformed.columns if col not in existing_columns
            ]
            if missing_columns:
                print(
                    "Existing table schema is incompatible. Missing columns: "
                    + ", ".join(missing_columns),
                    file=sys.stderr,
                )
                return 1

        upsert_dataframe(first_transformed, fact_table, conn)

    for chunk in chunk_iter:
        if remaining_rows is not None and remaining_rows <= 0:
            break
        if remaining_rows is not None and len(chunk) > remaining_rows:
            chunk = chunk.head(remaining_rows)
            remaining_rows = 0
        elif remaining_rows is not None:
            remaining_rows -= len(chunk)

        with engine.begin() as conn:
            transformed = transform_chunk(
                chunk,
                conn,
                brand_table,
                model_table,
                fuel_type_table,
                transmission_table,
                drive_type_table,
                body_class_table,
                cache,
            )
            upsert_dataframe(transformed, fact_table, conn)

    return 0

if __name__ == "__main__":
    raise SystemExit(main())
