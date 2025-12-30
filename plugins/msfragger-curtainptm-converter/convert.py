#!/usr/bin/env python3
"""
MSFragger to CurtainPTM Converter
Converts MSFragger PTM output to CurtainPTM upload format.
"""

import argparse
import sys
import os
from pathlib import Path

try:
    from curtainutils.msfragger import process_msfragger_ptm_single_site
except ImportError:
    print("Error: curtainutils package not found. Please install it using: pip install curtainutils")
    sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Convert MSFragger PTM output to CurtainPTM format"
    )

    parser.add_argument(
        "--input_file",
        required=True,
        help="Path to MSFragger PTM output file"
    )

    parser.add_argument(
        "--index_col",
        default="Index",
        help="Column name containing site information (default: Index)"
    )

    parser.add_argument(
        "--peptide_col",
        default="Peptide",
        help="Column name containing peptide sequences (default: Peptide)"
    )

    parser.add_argument(
        "--fasta_file",
        default="",
        help="Path to FASTA file (optional, will fetch from UniProt if not provided)"
    )

    parser.add_argument(
        "--get_position_from_peptide",
        action="store_true",
        help="Parse position from lowercase residues in peptide column"
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

    output_path = Path(args.output_folder)
    output_path.mkdir(parents=True, exist_ok=True)
    output_file = output_path / args.output_filename

    print(f"Processing MSFragger PTM data...")
    print(f"  Input file: {args.input_file}")
    print(f"  Index column: {args.index_col}")
    print(f"  Peptide column: {args.peptide_col}")
    print(f"  FASTA file: {args.fasta_file if args.fasta_file else 'Not provided (will fetch from UniProt)'}")
    print(f"  Get position from peptide: {args.get_position_from_peptide}")
    print(f"  Sequence window size: {args.sequence_window_size}")
    print(f"  Output file: {output_file}")

    try:
        process_msfragger_ptm_single_site(
            file_path=args.input_file,
            index_col=args.index_col,
            peptide_col=args.peptide_col,
            output_file=str(output_file),
            fasta_file=args.fasta_file,
            get_position_from_peptide_column=args.get_position_from_peptide,
            columns=args.uniprot_columns,
            sequence_window_size=args.sequence_window_size
        )

        print(f"\nConversion completed successfully!")
        print(f"Output file: {output_file}")

    except Exception as e:
        print(f"\nError during conversion: {str(e)}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
