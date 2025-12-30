#!/usr/bin/env python3
"""
Spectronaut to CurtainPTM Converter
Converts Spectronaut PTM output to CurtainPTM upload format.
"""

import argparse
import sys
import os
from pathlib import Path

try:
    from curtainutils.spectronaut import process_spectronaut_ptm
except ImportError:
    print("Error: curtainutils package not found. Please install it using: pip install curtainutils")
    sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Convert Spectronaut PTM output to CurtainPTM format"
    )

    parser.add_argument(
        "--input_file",
        required=True,
        help="Path to Spectronaut PTM output file"
    )

    parser.add_argument(
        "--index_col",
        default="PTM_collapse_key",
        help="Column name containing position information (default: PTM_collapse_key)"
    )

    parser.add_argument(
        "--peptide_col",
        default="PEP.StrippedSequence",
        help="Column name containing peptide sequences (default: PEP.StrippedSequence)"
    )

    parser.add_argument(
        "--uniprot_id_col",
        default="UniprotID",
        help="Column name containing UniProt IDs (default: UniprotID)"
    )

    parser.add_argument(
        "--processing_mode",
        default="1",
        choices=["1", "2", "3"],
        help="Processing mode: 1=basic, 2=PTM group, 3=ProForma (default: 1)"
    )

    parser.add_argument(
        "--modification_type",
        default="Phospho (STY)",
        help="Modification type for modes 2 and 3 (default: Phospho (STY))"
    )

    parser.add_argument(
        "--fasta_file",
        default="",
        help="Path to FASTA file (optional, will fetch from UniProt if not provided)"
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

    mode_descriptions = {
        "1": "Basic PTM processing",
        "2": "PTM group processing with site probabilities",
        "3": "ProForma sequence processing with position validation"
    }

    print(f"Processing Spectronaut PTM data...")
    print(f"  Input file: {args.input_file}")
    print(f"  Index column: {args.index_col}")
    print(f"  Peptide column: {args.peptide_col}")
    print(f"  UniProt ID column: {args.uniprot_id_col}")
    print(f"  Processing mode: {args.processing_mode} ({mode_descriptions[args.processing_mode]})")
    if args.processing_mode in ["2", "3"]:
        print(f"  Modification type: {args.modification_type}")
    print(f"  FASTA file: {args.fasta_file if args.fasta_file else 'Not provided (will fetch from UniProt)'}")
    print(f"  Sequence window size: {args.sequence_window_size}")
    print(f"  Output file: {output_file}")

    try:
        process_spectronaut_ptm(
            file_path=args.input_file,
            index_col=args.index_col,
            peptide_col=args.peptide_col,
            output_file=str(output_file),
            fasta_file=args.fasta_file,
            uniprot_id_col=args.uniprot_id_col,
            mode=args.processing_mode,
            modification=args.modification_type,
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
