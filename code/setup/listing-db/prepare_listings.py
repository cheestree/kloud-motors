#!/usr/bin/env python3
"""Prepare a CSV for load_chunks_to_db by renaming columns, deduplicating VINs, and limiting rows."""

import argparse
import random
import sys
from pathlib import Path
from typing import Dict, List, Optional, Set

import pandas as pd

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
OUTPUT_COLUMNS: List[str] = list(COLUMN_MAP.values())


def build_output_path(dataset_path: str, rows: Optional[int]) -> str:
    source = Path(dataset_path)
    suffix = f"_{rows}_rows" if rows is not None else "_all_rows"
    return str(source.with_name(f"{source.stem}_prepared{suffix}.csv"))


def rename_and_select(df: pd.DataFrame) -> pd.DataFrame:
    """Keep only mapped source columns and rename them to snake_case output names."""
    available = {src: dst for src, dst in COLUMN_MAP.items() if src in df.columns}
    if not available:
        return pd.DataFrame(columns=OUTPUT_COLUMNS)
    df = df[list(available.keys())].rename(columns=available)
    for col in OUTPUT_COLUMNS:
        if col not in df.columns:
            df[col] = pd.NA
    return df[OUTPUT_COLUMNS]


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
    parser.add_argument("--dataset", default="dataset.csv")
    parser.add_argument("--output", default="dataset_prepared.csv")
    parser.add_argument("--rows", type=int, default=None, help="Max output rows after dedup.")
    parser.add_argument("--vin-column", default="vin", help="Source VIN column name.")
    parser.add_argument("--dedupe-keep", choices=["first", "last"], default="last")
    parser.add_argument("--selection", choices=["head", "random"], default="head")
    parser.add_argument("--random-seed", type=int, default=42)
    parser.add_argument("--chunk-size", type=int, default=50_000)
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

    def iter_chunks():
        return pd.read_csv(args.dataset, chunksize=args.chunk_size, low_memory=False)

    if args.dedupe_keep == "first":
        seen_vins: Set[str] = set()
        deduped_rows = 0

        for chunk in iter_chunks():
            input_rows += len(chunk)
            chunk = rename_and_select(chunk)
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
            chunk = rename_and_select(chunk)
            chunk = clean_chunk(chunk, output_vin_col)
            rows_with_vin += len(chunk)
            for vin in chunk[output_vin_col]:
                pos += 1
                last_pos[vin] = pos
        deduped_rows = len(last_pos)

        # Pass 2: emit only the last-seen row per VIN.
        pos = 0
        for chunk in iter_chunks():
            chunk = rename_and_select(chunk)
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
            out_df.to_csv(output_path, index=False)
            exported_rows = len(out_df)
        else:
            pd.DataFrame(columns=OUTPUT_COLUMNS).to_csv(output_path, index=False)
            exported_rows = 0
        header_written = True

    if not header_written:
        pd.DataFrame(columns=OUTPUT_COLUMNS).to_csv(output_path, index=False)

    print(f"Input rows:      {input_rows}")
    print(f"Rows with VIN:   {rows_with_vin}")
    print(f"Unique VIN rows: {deduped_rows}")
    print(f"Exported rows:   {exported_rows}")
    print(f"Output file:     {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())