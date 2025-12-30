import click
import pandas as pd
from uniprotparser.betaparser import UniprotParser, UniprotSequence
import io
import os


def get_uniprot_data(uniprot_ids: list[str]):
    """
    Get uniprot data from uniprot ids
    :param uniprot_ids: List of uniprot ids
    :return: pandas dataframe
    """
    parser = UniprotParser(include_isoform=True)
    data = []
    for p in parser.parse(uniprot_ids, 500):
        data.append(pd.read_csv(io.StringIO(p), sep="\t"))
    if len(data) == 0:
        return pd.DataFrame()
    elif len(data) == 1:
        return data[0]
    data = pd.concat(data)
    data = data[['Entry', 'Sequence']].drop_duplicates()
    return data


def remap_ptm(row: pd.Series, peptide_seq_col: str, position_in_peptide_col: str):
    peptide_seq = row[peptide_seq_col].upper()
    position_in_peptide = row[position_in_peptide_col]
    if pd.notna(position_in_peptide) and pd.notna(row["Sequence"]):

        peptide_position = -1
        try:
            if ";" in row["Sequence"]:
                peptide_position = row["Sequence"].split(";")[0].index(peptide_seq)
            else:
                peptide_position = row["Sequence"].index(peptide_seq)
        except ValueError:
            try:
                peptide_position = row["Sequence"].replace("I", "L").index(peptide_seq.replace("I", "L"))
                row["Comment"] = "I replaced by L"
            except ValueError:
                row["Comment"] = "Peptide not found"

        if peptide_position >= -1:
            position = int(peptide_position + position_in_peptide - 1)
            row["RemappedPosition"] = position + 1
            # row["Residue"] = row["Sequence"][position]
            # sequence_window = ""
            # if position - 1 - 10 >= 0:
            #     sequence_window += row["Sequence"][position - 1 - 10:position - 1]
            # else:
            #     sequence_window += row["Sequence"][:position - 1]
            #     if len(sequence_window) < 10:
            #         sequence_window = "_" * (10 - len(sequence_window)) + sequence_window
            # sequence_window += row["Sequence"][position - 1]
            # if position + 10 <= len(row["Sequence"]):
            #     sequence_window += row["Sequence"][position:position + 10]
            # else:
            #     sequence_window += row["Sequence"][position:]
            #     if len(sequence_window) < 21:
            #         sequence_window += "_" * (21 - len(sequence_window))
            #
            # row["Sequence.window"] = sequence_window
    return row

def remap(main_df: pd.DataFrame, peptide_column: str, uniprot_acc_column: str, position_in_peptide_column: str, seq_df: pd.DataFrame, output_folder: str):
    """
    Remap the peptides to the uniprot sequence
    :param main_df: Input tabular df
    :param peptide_column: Peptide column
    :param uniprot_acc_column: Uniprot accession column
    :param seq_df: Sequence dataframe
    :param output_folder: Output folder
    :return:
    """

    df = main_df.merge(seq_df, left_on=uniprot_acc_column, right_on="Entry", how="left")
    df = df.apply(lambda x: remap_ptm(x, peptide_column, position_in_peptide_column), axis=1)
    df.drop("Sequence", axis=1, inplace=True)
    return df



def load_fasta_library(fasta_file: str):
    """
    Load fasta library
    :param fasta_file: Fasta file
    :return: pd.DataFrame
    """
    df = []
    with open(fasta_file, 'rt') as f:
        fasta_data = ""
        current_label = ""
        for line in f:
            line = line.strip()
            if line.startswith(">"):
                if fasta_data != "":
                    df.append([current_label[:], fasta_data[:]])
                acc = UniprotSequence(line, parse_acc=True)
                if acc.accession:
                    current_label = acc.accession+acc.isoform
                else:
                    current_label = line[1:]
                fasta_data = ""
            else:
                fasta_data += line
        if fasta_data != "":
            df.append([current_label[:], fasta_data[:]])
    if len(df) == 0:
        return pd.DataFrame()
    return pd.DataFrame(df, columns=["Entry", "Sequence"]).drop_duplicates()

def process(fasta_file: str, input_file: str, peptide_column: str, uniprot_acc_column: str, position_in_peptide_column: str, output_folder: str):
    if input_file.endswith(".tsv") or input_file.endswith(".txt"):
        main_df = pd.read_csv(input_file, sep="\t")
    elif input_file.endswith(".csv"):
        main_df = pd.read_csv(input_file)
    else:
        raise ValueError("File format not supported")
    for i, r in main_df.iterrows():
        acc = UniprotSequence(r[uniprot_acc_column], parse_acc=True)
        if acc.accession:
            main_df.at[i, "MatchACC"] = acc.accession+acc.isoform
        else:
            main_df.at[i, "MatchACC"] = r[uniprot_acc_column]
    if fasta_file != "":
        seq_df = load_fasta_library(fasta_file)
    else:

        seq_df = get_uniprot_data(main_df["MatchACC"].unique().tolist())

    df = remap(main_df, peptide_column, "MatchACC", position_in_peptide_column, seq_df, output_folder)
    df.drop("MatchACC", axis=1, inplace=True)
    os.makedirs(output_folder, exist_ok=True)
    df.to_csv(os.path.join(output_folder, "remapped_peptides.txt"), sep="\t", index=False)


@click.command()
@click.option("--fasta_file", "-f", help="Path to the fasta file", default="")
@click.option("--input_file", "-i", help="Path to the input file")
@click.option("--peptide_column", "-p", help="Peptide column")
@click.option("--uniprot_acc_column", "-u", help="Uniprot accession column")
@click.option("--position_in_peptide_column", "-pos", help="Position in peptide column")
@click.option("--output_folder", "-o", help="Path to the output folder")
def main(fasta_file: str, input_file: str, peptide_column: str, uniprot_acc_column: str, position_in_peptide_column: str, output_folder: str):
    process(fasta_file, input_file, peptide_column, uniprot_acc_column, position_in_peptide_column, output_folder)

if __name__ == "__main__":
    main()