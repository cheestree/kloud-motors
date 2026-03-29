#!/usr/bin/env python3
"""Prepare a CSV for load_chunks_to_db by deduplicating VINs and limiting rows."""

import argparse
from pathlib import Path
import random
import sys
from typing import Dict, List, Optional, Set

import pandas as pd


def prompt_text(prompt: str, default: Optional[str] = None) -> str:
    suffix = f" [{default}]" if default else ""
    value = input(f"{prompt}{suffix}: ").strip()
    if value:
        return value
    return default or ""


def prompt_int(prompt: str, default: Optional[int] = None) -> Optional[int]:
    default_text = str(default) if default is not None else None
    while True:
        raw_value = prompt_text(prompt, default_text)
        if raw_value == "":
            return None
        try:
            value = int(raw_value)
        except ValueError:
            print("Please enter a valid integer.", file=sys.stderr)
            continue
        if value <= 0:
            print("Please enter an integer greater than zero.", file=sys.stderr)
            continue
        return value


def build_output_path(dataset_path: str, rows: Optional[int]) -> str:
    source = Path(dataset_path)
    suffix = f"_{rows}_rows" if rows is not None else "_all_rows"
    return str(source.with_name(f"{source.stem}_deduped{suffix}.csv"))


def clean_chunk(chunk: pd.DataFrame, vin_column: str) -> pd.DataFrame:
    filtered = chunk[chunk[vin_column].notna()].copy()
    filtered[vin_column] = filtered[vin_column].astype(str).str.strip()
    return filtered[filtered[vin_column] != ""]


def write_rows(
    output_path: str,
    rows_df: pd.DataFrame,
    header_written: bool,
) -> bool:
    if rows_df.empty:
        return header_written
    rows_df.to_csv(output_path, index=False, mode="a" if header_written else "w", header=not header_written)
    return True


def append_reservoir_sample(
    reservoir: List[dict],
    candidate_rows: pd.DataFrame,
    sample_size: int,
    seen_count: int,
    rng: random.Random,
) -> int:
    for _, row in candidate_rows.iterrows():
        seen_count += 1
        row_dict = row.to_dict()
        if len(reservoir) < sample_size:
            reservoir.append(row_dict)
            continue
        replacement_idx = rng.randint(1, seen_count)
        if replacement_idx <= sample_size:
            reservoir[replacement_idx - 1] = row_dict
    return seen_count


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Deduplicate VINs in a CSV and export a limited-size output file."
    )
    parser.add_argument(
        "--dataset",
        default="CIS_Automotive_Kaggle_Sample.csv",
        help="Path to the source CSV dataset.",
    )
    parser.add_argument(
        "--output",
        default=None,
        help="Output CSV path. Defaults to <dataset>_deduped_<rows>.csv.",
    )
    parser.add_argument(
        "--rows",
        type=int,
        default=None,
        help="Maximum number of rows in output after deduplication.",
    )
    parser.add_argument(
        "--vin-column",
        default="vin",
        help="Column name that identifies each vehicle.",
    )
    parser.add_argument(
        "--dedupe-keep",
        choices=["first", "last"],
        default="last",
        help="Which duplicate VIN row to keep.",
    )
    parser.add_argument(
        "--selection",
        choices=["head", "random"],
        default="head",
        help="How to choose rows when --rows is provided.",
    )
    parser.add_argument(
        "--random-seed",
        type=int,
        default=42,
        help="Random seed used when --selection random.",
    )
    parser.add_argument(
        "--chunk-size",
        type=int,
        default=50000,
        help="Number of rows to process per chunk when reading the source CSV.",
    )
    parser.add_argument(
        "--interactive",
        action="store_true",
        help="Prompt for options with a simple interactive interface.",
    )
    return parser.parse_args()


def apply_interactive_overrides(args: argparse.Namespace) -> argparse.Namespace:
    args.dataset = prompt_text("Dataset path", args.dataset)
    args.vin_column = prompt_text("VIN column", args.vin_column)

    print("Duplicate handling options: first, last")
    args.dedupe_keep = prompt_text("Keep duplicate", args.dedupe_keep)
    if args.dedupe_keep not in {"first", "last"}:
        print("Invalid option selected. Falling back to 'last'.")
        args.dedupe_keep = "last"

    args.rows = prompt_int("Rows to export (blank for all)", args.rows)

    print("Row selection options: head, random")
    args.selection = prompt_text("Selection method", args.selection)
    if args.selection not in {"head", "random"}:
        print("Invalid option selected. Falling back to 'head'.")
        args.selection = "head"

    if args.selection == "random":
        random_seed = prompt_int("Random seed", args.random_seed)
        args.random_seed = random_seed if random_seed is not None else args.random_seed

    default_output = build_output_path(args.dataset, args.rows)
    args.output = prompt_text("Output CSV path", args.output or default_output)
    return args


def main() -> int:
    args = parse_args()

    if args.interactive:
        args = apply_interactive_overrides(args)

    if args.rows is not None and args.rows <= 0:
        print("--rows must be greater than zero.", file=sys.stderr)
        return 1
    if args.chunk_size <= 0:
        print("--chunk-size must be greater than zero.", file=sys.stderr)
        return 1

    output_path = args.output or build_output_path(args.dataset, args.rows)

    try:
        first_chunk = pd.read_csv(args.dataset, chunksize=args.chunk_size)
        first_chunk_df = next(first_chunk)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1
    except StopIteration:
        pd.DataFrame().to_csv(output_path, index=False)
        print("Input rows: 0")
        print("Rows with VIN: 0")
        print("Unique VIN rows: 0")
        print("Exported rows: 0")
        print(f"Output file: {output_path}")
        return 0

    if args.vin_column not in first_chunk_df.columns:
        print(
            f"VIN column '{args.vin_column}' was not found in dataset columns.",
            file=sys.stderr,
        )
        return 1

    reservoir: List[dict] = []
    reservoir_seen = 0
    rng = random.Random(args.random_seed)
    exported_rows = 0
    header_written = False
    input_rows = 0
    rows_with_vin = 0

    if args.dedupe_keep == "first":
        seen_vins: Set[str] = set()

        for chunk in pd.read_csv(args.dataset, chunksize=args.chunk_size):
            input_rows += len(chunk)
            cleaned = clean_chunk(chunk, args.vin_column)
            rows_with_vin += len(cleaned)

            is_new_vin = ~cleaned[args.vin_column].isin(seen_vins)
            new_rows = cleaned[is_new_vin]
            seen_vins.update(new_rows[args.vin_column].tolist())

            if args.selection == "random" and args.rows is not None:
                reservoir_seen = append_reservoir_sample(
                    reservoir,
                    new_rows,
                    args.rows,
                    reservoir_seen,
                    rng,
                )
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

        deduped_rows = len(seen_vins)

    else:
        last_valid_position_by_vin: Dict[str, int] = {}
        valid_position = 0

        for chunk in pd.read_csv(args.dataset, chunksize=args.chunk_size):
            input_rows += len(chunk)
            cleaned = clean_chunk(chunk, args.vin_column)
            rows_with_vin += len(cleaned)

            for vin in cleaned[args.vin_column].tolist():
                valid_position += 1
                last_valid_position_by_vin[vin] = valid_position

        deduped_rows = len(last_valid_position_by_vin)
        valid_position = 0

        for chunk in pd.read_csv(args.dataset, chunksize=args.chunk_size):
            cleaned = clean_chunk(chunk, args.vin_column)
            keep_mask: List[bool] = []
            for vin in cleaned[args.vin_column].tolist():
                valid_position += 1
                keep_mask.append(last_valid_position_by_vin.get(vin) == valid_position)

            deduped_chunk = cleaned[keep_mask]

            if args.selection == "random" and args.rows is not None:
                reservoir_seen = append_reservoir_sample(
                    reservoir,
                    deduped_chunk,
                    args.rows,
                    reservoir_seen,
                    rng,
                )
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
            random_output = pd.DataFrame(reservoir)
            random_output = random_output.reindex(columns=first_chunk_df.columns)
            random_output.to_csv(output_path, index=False)
            exported_rows = len(random_output)
            header_written = True
        else:
            header_written = write_rows(output_path, first_chunk_df.head(0), header_written)
            exported_rows = 0

    if not header_written:
        write_rows(output_path, first_chunk_df.head(0), header_written)

    print(f"Input rows: {input_rows}")
    print(f"Rows with VIN: {rows_with_vin}")
    print(f"Unique VIN rows: {deduped_rows}")
    print(f"Exported rows: {exported_rows}")
    print(f"Output file: {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
