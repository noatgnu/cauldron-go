import os

import click
import pandas as pd


def load_fasta_library(fasta_file: str, miss_cleavage: int = 2, min_length: int = 5):
    """
    Load fasta file into a pandas dataframe
    :param fasta_file: Path to the fasta file
    :param miss_cleavage: Number of miss cleavage
    :param min_length: Minimum length of the peptide
    :return: pandas dataframe
    """
    seq_df = []
    with open(fasta_file, 'rt') as f:
        current_label = ""
        current_seq = ""
        for i in f:
            if i.startswith(">"):
                if current_label != "":
                    fragments = {}
                    # get all tryptic fragments
                    last_pos = 0
                    for pos, aa in enumerate(current_seq):
                        if aa in ["K", "R"]:
                            fragments[last_pos] = current_seq[last_pos:pos + 1]
                            last_pos = pos + 1
                    if last_pos < len(current_seq):
                        fragments[last_pos] = current_seq[last_pos:]
                    # write all tryptic fragments with two miss cleavages
                    all_pos = list(fragments.keys())
                    for i2 in range(len(all_pos)):
                        if i2 + miss_cleavage < len(all_pos):
                            combo = ""
                            for i3 in range(miss_cleavage + 1):
                                combo += fragments[all_pos[i2 + i3]]
                            if len(combo) > min_length:
                                seq_df.append([f"{current_label}_{all_pos[i2]}_missed_cleavage", combo])
                current_label = i.strip()
                current_seq = ""
            else:
                current_seq += i.strip()
        if current_label != "":
            fragments = {}
            # get all tryptic fragments
            last_pos = 0
            for pos, aa in enumerate(current_seq):
                if aa in ["K", "R"]:
                    fragments[last_pos] = current_seq[last_pos:pos + 1]
                    last_pos = pos + 1
            if last_pos < len(current_seq):
                fragments[last_pos] = current_seq[last_pos:]
            # write all tryptic fragments with one miss cleavage
            all_pos = list(fragments.keys())
            for i2 in range(len(all_pos)):
                if i2 + miss_cleavage < len(all_pos):
                    combo = ""
                    for i3 in range(miss_cleavage + 1):
                        combo += fragments[all_pos[i2 + i3]]
                    if len(combo) > min_length:
                        seq_df.append(
                            [f"{current_label}_{all_pos[i2]}_missed_cleavage", combo])
    seq_df = pd.DataFrame(seq_df, columns=["Entry", "Sequence"])
    return seq_df

def check_data_for_peptide_in_library(file_path: str, peptide_column: str, seq_df: pd.DataFrame, output_folder: str):
    """
    Check if the peptide in the file is in the library
    :param file_path:
    :param peptide_column:
    :param seq_df:
    :return:
    """
    unique_seq_df = []
    for i, g in seq_df.groupby("Sequence"):
        r = g.iloc[0]
        unique_seq_df.append([r["Entry"], r["Sequence"]])
    unique_seq_df = pd.DataFrame(unique_seq_df, columns=["Entry", "Sequence"])
    unique_seq_df["Search.Col"] = unique_seq_df["Sequence"].apply(lambda x: x.replace("I", "L"))
    if file_path.endswith(".tsv") or file_path.endswith(".txt"):
        data = pd.read_csv(file_path, sep="\t")
    elif file_path.endswith(".csv"):
        data = pd.read_csv(file_path)
    else:
        raise ValueError("File format not supported")
    data["Search.Col"] = data[peptide_column].apply(lambda x: x.replace("I", "L"))
    data["Found"] = False
    content = "\t".join(unique_seq_df["Search.Col"].tolist())
    for i, row in data.iterrows():
        if row["Search.Col"] in content:
            data.at[i, "Found"] = True

    os.makedirs(output_folder, exist_ok=True)
    data.to_csv(os.path.join(output_folder, "peptide_in_library.txt"), sep="\t", index=False)


@click.command()
@click.option("--fasta_file", "-f", help="Path to the fasta file")
@click.option("--miss_cleavage", "-m", help="Number of miss cleavage", default=2)
@click.option("--min_length", "-l", help="Minimum length of the peptide", default=5)
@click.option("--file_path", "-i", help="Path to the input file")
@click.option("--peptide_column", "-p", help="Name of the peptide column")
@click.option("--output_folder", "-o", help="Path to the output folder")
def main(fasta_file: str, miss_cleavage: int, min_length: int, file_path: str, peptide_column: str, output_folder: str):
    seq_df = load_fasta_library(fasta_file, miss_cleavage, min_length)
    check_data_for_peptide_in_library(file_path, peptide_column, seq_df, output_folder)

if __name__ == "__main__":
    main()
