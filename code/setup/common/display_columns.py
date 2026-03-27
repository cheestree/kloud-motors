#!/usr/bin/env python3
"""Display dataset columns."""

import argparse
import sys

import pandas as pd


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Display CSV columns.")
    parser.add_argument(
        "--dataset",
        default="CIS_Automotive_Kaggle_Sample.csv",
        help="Path to the CSV dataset.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        df = pd.read_csv(args.dataset, nrows=0)
    except FileNotFoundError:
        print(f"Dataset not found: {args.dataset}", file=sys.stderr)
        return 1

    for col in df.columns:
        print(col)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
