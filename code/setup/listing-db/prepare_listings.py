#!/usr/bin/env python3
"""Prepare a CSV for load_chunks_to_db by renaming columns, deduplicating VINs, and limiting rows."""

import argparse
import random
import sys
from pathlib import Path
from typing import Dict, List, Optional, Set

import pandas as pd
from faker import Faker

# Source column to snake_case output column.
# Only columns in this map will be kept in the output.
COLUMN_MAP: Dict[str, str] = {
    "vin":                    "vin",
    "stockNum":               "stock_num",
    "firstSeen":              "first_seen",
    "lastSeen":               "last_seen",
    "askPrice":               "ask_price",
    "msrp":                   "msrp",
    "mileage":                "mileage",
    "isNew":                  "is_new",
    "isSold":                 "is_sold",
    "color":                  "color",
    "interiorColor":          "interior_color",
    "brandName":              "brand_name",
    "modelName":              "model_name",
    "dealerID":               "dealer_id",
    "vf_ModelYear":           "model_year",
    "vf_BodyClass":           "body_class",
    "vf_FuelTypePrimary":     "fuel_type_primary",
    "vf_TransmissionStyle":   "transmission_style",
    "vf_TransmissionSpeeds":  "transmission_speeds",
    "vf_DriveType":           "drive_type",
    "vf_EngineCylinders":     "engine_cylinders",
    "vf_EngineHP":            "engine_hp",
    "vf_DisplacementL":       "displacement_l",
    "vf_Doors":               "doors",
    "vf_Trim":                "trim",
    "vf_Turbo":               "turbo",
    "vf_ElectrificationLevel": "electrification_level",
}

# Canonical output column order
OUTPUT_COLUMNS: List[str] = list(COLUMN_MAP.values()) + ["district", "city", "country", "state"]


def build_output_path(dataset_path: str, rows: Optional[int]) -> str:
    source = Path(dataset_path)
    suffix = f"_{rows}_rows" if rows is not None else "_all_rows"
    return str(source.with_name(f"{source.stem}_prepared{suffix}.csv"))


def rename_and_select(df: pd.DataFrame, faker: Optional[Faker] = None) -> pd.DataFrame:
    """Keep only mapped source columns and rename them to snake_case output names. Add US district, city, country."""
    available = {src: dst for src, dst in COLUMN_MAP.items() if src in df.columns}
    if not available:
        out = pd.DataFrame(columns=OUTPUT_COLUMNS)
    else:
        out = df[list(available.keys())].rename(columns=available)
        for col in COLUMN_MAP.values():
            if col not in out.columns:
                out[col] = pd.NA
        out = out[list(COLUMN_MAP.values())]

    # is_sold defaults to false when not provided in the source CSV.
    if "is_sold" not in out.columns:
        out["is_sold"] = False
    else:
        out["is_sold"] = out["is_sold"].map(
            lambda v: str(v).strip().lower() in {"1", "true", "t", "yes", "y"}
            if pd.notna(v)
            else False
        )
    if faker is not None:
        districts = []
        cities = []
        states = []
        countries = []

        for _ in range(len(out)):
            loc = faker.local_latlng(country_code="US")
            if loc is None:
                districts.append(pd.NA)
                cities.append(pd.NA)
                states.append(pd.NA)
                countries.append(pd.NA)
                continue

            _, _, city, _, state = loc
            cities.append(city)
            states.append(state)
            districts.append(pd.NA)  # Always null
            countries.append("United States")

        out["district"] = districts
        out["city"] = cities
        out["state"] = states
        out["country"] = countries
    else:
        out["district"] = pd.NA
        out["city"] = pd.NA
        out["state"] = pd.NA
        out["country"] = pd.NA
    return out[OUTPUT_COLUMNS]


def clean_chunk(df: pd.DataFrame, vin_col: str) -> pd.DataFrame:
    mask = df[vin_col].notna()
    df = df[mask].copy()
    df[vin_col] = df[vin_col].astype(str).str.strip()
    return df[df[vin_col] != ""]


def write_rows(path: str, df: pd.DataFrame, header_written: bool) -> bool:
    if df.empty:
        return header_written
    df.to_csv(path, index=False, mode="a" if header_written else "w", header=not header_written)
    return True


def append_reservoir_sample(
    reservoir: List[dict],
    rows: pd.DataFrame,
    sample_size: int,
    seen_count: int,
    rng: random.Random,
) -> int:
    for _, row in rows.iterrows():
        seen_count += 1
        d = row.to_dict()
        if len(reservoir) < sample_size:
            reservoir.append(d)
        else:
            idx = rng.randint(1, seen_count)
            if idx <= sample_size:
                reservoir[idx - 1] = d
    return seen_count


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Rename columns, deduplicate VINs, and export a prepared CSV."
    )
    parser.add_argument("--dataset", default="CIS_Automotive_Kaggle_Sample.csv")
    parser.add_argument("--output", default="dataset_prepared.csv")
    parser.add_argument("--rows", type=int, default=50000, help="Max output rows after dedup.")
    parser.add_argument("--vin-column", default="vin", help="Source VIN column name.")
    parser.add_argument("--dedupe-keep", choices=["first", "last"], default="last")
    parser.add_argument("--selection", choices=["head", "random"], default="head")
    parser.add_argument("--random-seed", type=int, default=42)
    parser.add_argument("--chunk-size", type=int, default=8192, help="Number of rows to process at a time.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    if args.rows is not None and args.rows <= 0:
        print("--rows must be greater than zero.", file=sys.stderr)
        return 1
    if args.chunk_size <= 0:
        print("--chunk-size must be greater than zero.", file=sys.stderr)
        return 1

    output_path = args.output or build_output_path(args.dataset, args.rows)

    try:
        header_df = pd.read_csv(args.dataset, nrows=0)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1

    source_vin_col = args.vin_column
    if source_vin_col not in header_df.columns:
        print(f"VIN column '{source_vin_col}' not found in dataset.", file=sys.stderr)
        return 1

    output_vin_col = COLUMN_MAP.get(source_vin_col, source_vin_col)

    rng = random.Random(args.random_seed)
    reservoir: List[dict] = []
    reservoir_seen = 0
    header_written = False
    exported_rows = 0
    input_rows = 0
    rows_with_vin = 0


    faker = Faker("en_US")

    def iter_chunks():
        return pd.read_csv(args.dataset, chunksize=args.chunk_size, low_memory=False)

    if args.dedupe_keep == "first":
        seen_vins: Set[str] = set()
        deduped_rows = 0

        for chunk in iter_chunks():
            input_rows += len(chunk)
            chunk = rename_and_select(chunk, faker)
            chunk = clean_chunk(chunk, output_vin_col)
            rows_with_vin += len(chunk)

            is_new = ~chunk[output_vin_col].isin(seen_vins)
            new_rows = chunk[is_new]
            seen_vins.update(new_rows[output_vin_col].tolist())
            deduped_rows += len(new_rows)

            if args.selection == "random" and args.rows is not None:
                reservoir_seen = append_reservoir_sample(reservoir, new_rows, args.rows, reservoir_seen, rng)
                continue

            if args.rows is None:
                header_written = write_rows(output_path, new_rows, header_written)
                exported_rows += len(new_rows)
            else:
                remaining = args.rows - exported_rows
                if remaining <= 0:
                    continue
                to_write = new_rows.head(remaining)
                header_written = write_rows(output_path, to_write, header_written)
                exported_rows += len(to_write)

    else:  # keep last
        # Pass 1: record the last valid position of each VIN.
        last_pos: Dict[str, int] = {}
        pos = 0
        for chunk in iter_chunks():
            input_rows += len(chunk)
            chunk = rename_and_select(chunk, faker)
            chunk = clean_chunk(chunk, output_vin_col)
            rows_with_vin += len(chunk)
            for vin in chunk[output_vin_col]:
                pos += 1
                last_pos[vin] = pos
        deduped_rows = len(last_pos)

        # Pass 2: emit only the last-seen row per VIN.
        pos = 0
        for chunk in iter_chunks():
            chunk = rename_and_select(chunk, faker)
            chunk = clean_chunk(chunk, output_vin_col)
            keep: List[bool] = []
            for vin in chunk[output_vin_col]:
                pos += 1
                keep.append(last_pos.get(vin) == pos)
            deduped_chunk = chunk[keep]

            if args.selection == "random" and args.rows is not None:
                reservoir_seen = append_reservoir_sample(reservoir, deduped_chunk, args.rows, reservoir_seen, rng)
                continue

            if args.rows is None or args.rows >= deduped_rows:
                header_written = write_rows(output_path, deduped_chunk, header_written)
                exported_rows += len(deduped_chunk)
            else:
                remaining = args.rows - exported_rows
                if remaining <= 0:
                    continue
                to_write = deduped_chunk.head(remaining)
                header_written = write_rows(output_path, to_write, header_written)
                exported_rows += len(to_write)

    if args.selection == "random" and args.rows is not None:
        if reservoir:
            out_df = pd.DataFrame(reservoir).reindex(columns=OUTPUT_COLUMNS)
            out_df["district"] = [faker.county() for _ in range(len(out_df))]
            out_df["city"] = [faker.city() for _ in range(len(out_df))]
            out_df["country"] = ["United States" for _ in range(len(out_df))]
            # Chunked deduplication (keep last or first), with early stop if enough rows
            vin_to_row = {}  # vin -> row dict
            stop_early = False
            for chunk in iter_chunks():
                if stop_early:
                    break
                input_rows += len(chunk)
                chunk = rename_and_select(chunk, faker)
                chunk = clean_chunk(chunk, output_vin_col)
                rows_with_vin += len(chunk)
                for _, row in chunk.iterrows():
                    vin = row[output_vin_col]
                    if args.dedupe_keep == "first":
                        if vin not in vin_to_row:
                            vin_to_row[vin] = row
                    else:  # keep last
                        vin_to_row[vin] = row
                    # Early stop if enough unique VINs collected
                    if args.rows is not None and len(vin_to_row) >= args.rows:
                        stop_early = True
                        break
            deduped_rows = len(vin_to_row)

            # Convert dict to DataFrame
            deduped_df = pd.DataFrame(list(vin_to_row.values()))

            # Selection (random or head)
            if args.selection == "random" and args.rows is not None:
                deduped_df = deduped_df.sample(n=min(args.rows, len(deduped_df)), random_state=args.random_seed)
            elif args.rows is not None:
                deduped_df = deduped_df.head(args.rows)

            exported_rows = len(deduped_df)
            if not deduped_df.empty:
                deduped_df.to_csv(output_path, index=False)
                header_written = True
            else:
                pd.DataFrame(columns=OUTPUT_COLUMNS).to_csv(output_path, index=False)
                header_written = True

if __name__ == "__main__":
    main()