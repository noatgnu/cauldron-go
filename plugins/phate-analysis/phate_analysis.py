import os.path

import click
import numpy as np
import pandas as pd
import phate


def phate_analysis(df: pd.DataFrame, n_components: int):
    ph = phate.PHATE(n_components=n_components)
    phate_op = ph.fit_transform(df)
    return phate_op

def phate_(input_file: str, output_folder: str, columns_name: list[str], n_components: int = 2, log2: bool = False):
    assert n_components in [2, 3], "Invalid number of components"
    if input_file.endswith(".tsv") or input_file.endswith(".txt"):
        df = pd.read_csv(input_file, sep="\t")
    elif input_file.endswith(".csv"):
        df = pd.read_csv(input_file, sep=",")
    else:
        raise ValueError("Invalid file extension")
    data = np.log2(df[columns_name].transpose()) if log2 else df[columns_name].transpose()
    data.replace([np.inf, -np.inf], 0, inplace=True)
    phate_op = phate_analysis(data, n_components)
    phate_df = pd.DataFrame(phate_op)
    if n_components == 2:
        phate_df.rename(columns={0: "x_phate", 1: "y_phate"}, inplace=True)
    else:
        phate_df.rename(columns={0: "x_phate", 1: "y_phate", 2: "z_phate"}, inplace=True)
    phate_df["sample"] = columns_name
    os.makedirs(output_folder, exist_ok=True)
    phate_df.to_csv(os.path.join(output_folder, "phate_output.txt"), sep="\t", index=False)
    return phate_df

@click.command()
@click.option("--input_file", "-i", help="Path to the input file")
@click.option("--output_folder", "-o", help="Path to the output folder")
@click.option("--columns_name", "-c", help="Name of the columns to be included in the analysis")
@click.option("--n_components", "-n", help="Number of components", default=2)
@click.option("--log2", "-l", help="Log2 transform the data", default=False)
def main(input_file: str, output_folder: str, columns_name: str, n_components: int, log2: bool):
    phate_(input_file, output_folder, columns_name.split(","), n_components, log2)

if __name__ == '__main__':
    main()