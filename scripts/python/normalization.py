import os

import click
import numpy as np
import pandas as pd
from sklearn.preprocessing import MinMaxScaler, StandardScaler, RobustScaler, MaxAbsScaler, QuantileTransformer, PowerTransformer, Normalizer

def normalize(file_path: str, output_folder: str, columns_name: str, scaler_type: str = "minmax", with_centering: bool = True, with_scaling: bool = True, n_quantiles: int = 1000, output_distribution: str = "uniform", copy: bool = True, norm: str = "l2", power_method: str = "yeo-johnson"):
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
    if scaler_type == "minmax":
        scaler = MinMaxScaler(copy=copy)
    elif scaler_type == "standard":
        scaler = StandardScaler(copy=copy, with_mean=with_centering, with_std=with_scaling)
    elif scaler_type == "robust":
        scaler = RobustScaler(copy=copy, with_centering=with_centering, with_scaling=with_scaling)
    elif scaler_type == "maxabs":
        scaler = MaxAbsScaler(copy=copy)
    elif scaler_type == "quantile":
        scaler = QuantileTransformer(n_quantiles=n_quantiles, output_distribution=output_distribution, copy=copy)
    elif scaler_type == "power":
        scaler = PowerTransformer(method=power_method, standardize=with_scaling, copy=copy)
    elif scaler_type == "normalizer":
        scaler = Normalizer(norm=norm, copy=copy)
    else:
        raise ValueError("Invalid scaler type")
    sample_df = pd.DataFrame(scaler.fit_transform(sample_df), columns=columns)
    df[columns] = sample_df[columns]
    os.makedirs(output_folder, exist_ok=True)
    df.to_csv(os.path.join(output_folder, "normalized.data.txt"), sep="\t", index=False)

@click.command()
@click.option("--file_path", "-f", help="Path to the file to be processed")
@click.option("--output_folder", "-o", help="Path to the output folder")
@click.option("--columns_name", "-c", help="Name of the columns to be normalized")
@click.option("--scaler_type", "-s", help="Type of scaler", default="minmax")
@click.option("--with_centering", "-w", help="With centering", default=True)
@click.option("--with_scaling", "-t", help="With scaling", default=True)
@click.option("--n_quantiles", "-n", help="Number of quantiles", default=1000)
@click.option("--output_distribution", "-d", help="Output distribution", default="uniform")
@click.option("--copy", "-p", help="Copy", default=True)
@click.option("--norm", "-m", help="Norm", default="l2")
@click.option("--power_method", "-e", help="Power method", default="yeo-johnson")
def main(file_path: str, output_folder: str, columns_name: str, scaler_type: str, with_centering: bool, with_scaling: bool, n_quantiles: int, output_distribution: str, copy: bool, norm: str, power_method: str):
    normalize(file_path, output_folder, columns_name, scaler_type, with_centering, with_scaling, n_quantiles, output_distribution, copy, norm, power_method)

if __name__ == '__main__':
    main()