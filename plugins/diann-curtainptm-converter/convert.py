#!/usr/bin/env python3
"""
DIA-NN to CurtainPTM Converter
Converts DIA-NN PTM differential analysis output to CurtainPTM upload format.
"""

import argparse
import sys
import os
from pathlib import Path

try:
    from curtainutils.diann import process_diann_ptm
except ImportError:
    print("Error: curtainutils package not found. Please install it using: pip install curtainutils")
    sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Convert DIA-NN PTM output to CurtainPTM format"
    )

    parser.add_argument(
        "--pr_file",
        required=True,
        help="Path to DIA-NN PR file (differential analysis)"
    )

    parser.add_argument(
        "--report_file",
        default=None,
        help="Path to DIA-NN report file (optional)"
    )

    parser.add_argument(
        "--modification_type",
        default="UniMod:21",
        help="Modification type to process (default: UniMod:21 for Phosphorylation)"
    )

    parser.add_argument(
        "--custom_modification",
        default=None,
        help="Custom modification identifier (overrides modification_type)"
    )

    parser.add_argument(
        "--fasta_file",
        default=None,
        help="Path to FASTA file (optional, will fetch from UniProt if not provided)"
    )

    parser.add_argument(
        "--localization_score_col",
        default="PTM.Site.Confidence",
        help="Column name for localization score (default: PTM.Site.Confidence)"
    )

    parser.add_argument(
        "--multiple_site",
        action="store_true",
        help="Process multiple sites instead of single site"
    )

    parser.add_argument(
        "--uniprot_columns",
        default="accession,id,sequence,protein_name",
        help="UniProt columns to retrieve (default: accession,id,sequence,protein_name)"
    )

    parser.add_argument(
        "--output_filename",
        default="curtainptm_input.txt",
        help="Output filename (default: curtainptm_input.txt)"
    )

    parser.add_argument(
        "--output_meta",
        action="store_true",
        help="Generate additional metadata file with intensity information"
    )

    parser.add_argument(
        "--sequence_window_size",
        type=int,
        default=21,
        help="Size of sequence window around modification sites (default: 21)"
    )

    parser.add_argument(
        "--output_folder",
        required=True,
        help="Output folder for converted file"
    )

    args = parser.parse_args()

    modification = args.custom_modification if args.custom_modification else args.modification_type

    output_path = Path(args.output_folder)
    output_path.mkdir(parents=True, exist_ok=True)
    output_file = output_path / args.output_filename

    print(f"Processing DIA-NN PTM data...")
    print(f"  PR file: {args.pr_file}")
    print(f"  Report file: {args.report_file if args.report_file else 'Not provided (using PR file only)'}")
    print(f"  Modification: {modification}")
    print(f"  Multiple sites: {args.multiple_site}")
    print(f"  Output metadata: {args.output_meta}")
    print(f"  Sequence window size: {args.sequence_window_size}")
    print(f"  Output file: {output_file}")

    try:
        process_diann_ptm(
            pr_file_path=args.pr_file,
            report_file_path=args.report_file,
            output_file=str(output_file),
            modification_of_interests=modification,
            columns=args.uniprot_columns,
            fasta_file=args.fasta_file,
            localization_score_col=args.localization_score_col,
            output_meta=args.output_meta,
            multiple_site=args.multiple_site,
            sequence_window_size=args.sequence_window_size
        )

        print(f"\nConversion completed successfully!")
        print(f"Output file: {output_file}")

    except Exception as e:
        print(f"\nError during conversion: {str(e)}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
