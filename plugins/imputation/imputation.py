import os

import click
import numpy as np
import pandas as pd
from sklearn.experimental import enable_iterative_imputer
from sklearn.impute import IterativeImputer, KNNImputer, SimpleImputer


def impute(file_path: str, output_folder: str, columns_name: str, imputer_type: str = "knn", n_neighbors: int = 5, max_iter: int = 10, simple_strategy: str = "mean", fill_value: float = 0.0):
    if file_path.endswith(".tsv") or file_path.endswith(".txt"):
        df = pd.read_csv(file_path, sep="\t")
    elif file_path.endswith(".csv"):
        df = pd.read_csv(file_path, sep=",")
    else:
        raise ValueError("Invalid file extension")
    for i in ["NA", "NaN", "nan", "#VALUE!"]:
        df.replace(i, np.nan, inplace=True)
    columns = columns_name.split(",")
    sample_df = df[columns]
    if imputer_type == "knn":
        imputer = KNNImputer(n_neighbors=n_neighbors)
    elif imputer_type == "simple":
        if simple_strategy == "constant":
            imputer = SimpleImputer(strategy=simple_strategy, fill_value=fill_value)
        else:
            imputer = SimpleImputer(strategy=simple_strategy)
    elif imputer_type == "iterative":
        imputer = IterativeImputer(max_iter=max_iter)
    else:
        raise ValueError("Invalid imputer type")
    sample_df = pd.DataFrame(imputer.fit_transform(sample_df), columns=columns)
    df[columns] = sample_df[columns]
    os.makedirs(output_folder, exist_ok=True)
    df.to_csv(os.path.join(output_folder, "imputed.data.txt"), sep="\t", index=False)

@click.command()
@click.option("--file_path", "-f", help="Path to the file to be processed")
@click.option("--output_file", "-o", help="Path to the output file")
@click.option("--columns_name", "-c", help="Name of the columns to be imputed")
@click.option("--imputer_type", "-i", help="Type of imputer", default="knn")
@click.option("--n_neighbors", "-n", help="Number of neighbors", default=5)
@click.option("--max_iter", "-m", help="Maximum number of iterations", default=10)
@click.option("--simple_strategy", "-s", help="Simple imputer strategy", default="mean")
@click.option("--fill_value", "-v", help="Fill value for constant strategy", default=0.0)
def main(file_path: str, output_file: str, columns_name: str, imputer_type: str, n_neighbors: int, max_iter: int, simple_strategy: str, fill_value: float):
    impute(file_path, output_file, columns_name, imputer_type, n_neighbors, max_iter, simple_strategy, fill_value)

if __name__ == "__main__":
    main()