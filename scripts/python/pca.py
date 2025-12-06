import json
import os

import click
import numpy as np
import pandas as pd
from sklearn.decomposition import PCA


def pca_analysis(df: pd.DataFrame, n_components: int):
    pca = PCA(n_components=n_components)
    pca_op = pca.fit_transform(df)
    return pca_op, pca.explained_variance_ratio_

def pca_(input_file: str, output_folder: str, columns_name: list[str], n_components: int = 2, log2: bool = False):
    if input_file.endswith(".tsv") or input_file.endswith(".txt"):
        df = pd.read_csv(input_file, sep="\t")
    elif input_file.endswith(".csv"):
        df = pd.read_csv(input_file, sep=",")
    else:
        raise ValueError("Invalid file extension")
    data = np.log2(df[columns_name].transpose()) if log2 else df[columns_name].transpose()
    data.replace([np.inf, -np.inf], 0, inplace=True)
    pca_op, explained_ratio = pca_analysis(data, n_components)
    pca_df = pd.DataFrame(pca_op)
    if n_components == 2:
        pca_df.rename(columns={0: "x_pca", 1: "y_pca"}, inplace=True)
    else:
        pca_df.rename(columns={0: "x_pca", 1: "y_pca", 2: "z_pca"}, inplace=True)
    pca_df["sample"] = columns_name
    print(pca_df)
    os.makedirs(output_folder, exist_ok=True)
    pca_df.to_csv(os.path.join(output_folder, "pca_output.txt"), sep="\t", index=False)
    with open(os.path.join(output_folder, "explained_variance_ratio.json"), "w") as f:
        f.write(json.dumps(explained_ratio.tolist()))
    return pca_df

@click.command()
@click.option("--input_file", "-i", help="Path to the input file")
@click.option("--output_folder", "-o", help="Path to the output folder")
@click.option("--columns_name", "-c", help="Name of the columns to be included in the analysis")
@click.option("--n_components", "-n", help="Number of components", default=2)
@click.option("--log2", "-l", help="Log2 transform the data", default=False)
def main(input_file: str, output_folder: str, columns_name: str, n_components: int, log2: bool):
    pca_(input_file, output_folder, columns_name.split(","), n_components, log2)

if __name__ == '__main__':
    main()