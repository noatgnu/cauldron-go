import numpy as np
import pandas as pd
import click
import os
import dabest
from matplotlib import pyplot as plt
import re

def est_plot(file_path: str, index_col: str, selected_protein: list[str], sample_annotation: str, out_folder: str, log2: bool = True, condition_order: list[str] = []):
    if file_path.endswith(".txt") or file_path.endswith(".tsv"):
        df = pd.read_csv(file_path, sep="\t")
    elif file_path.endswith(".csv"):
        df = pd.read_csv(file_path)
    else:
        raise ValueError("File format not supported")
    df = df[df[index_col].isin(selected_protein)]

    if sample_annotation.endswith(".txt") or sample_annotation.endswith(".tsv"):
        sample_annotation = pd.read_csv(sample_annotation, sep="\t")
    elif sample_annotation.endswith(".csv"):
        sample_annotation = pd.read_csv(sample_annotation)
    else:
        raise ValueError("File format not supported")
    sample_annotation["Sample"] = sample_annotation["Sample"].astype(str)
    sample_annotation["Condition"] = sample_annotation["Condition"].astype(str)

    sample_annotation = sample_annotation.set_index("Sample").to_dict()["Condition"]

    melted_df = df.melt(id_vars=[index_col], value_vars=sample_annotation.keys(), var_name="Sample", value_name="Value")
    melted_df["Condition"] = melted_df["Sample"].map(sample_annotation)

    for g, d in melted_df.groupby(index_col):
        # replace special characters
        g = re.sub(r'[^\w\s]', '_', g)
        # remove missing values if no values was detected in condition
        for c in d["Condition"].unique():
            if d[d["Condition"] == c]["Value"].isna().all():
                d = d[d["Condition"] != c]

        if condition_order:
            d["Condition"] = pd.Categorical(d["Condition"], [i for i in condition_order if i in d["Condition"].values], ordered=True)
        else:
            d["Condition"] = pd.Categorical(d["Condition"], d["Condition"].unique(), ordered=True)
        if log2:
            d["Value"] = np.log2(d["Value"])
        print(d["Condition"])
        dabest_obj = dabest.load(data=d, x="Condition", y="Value", idx=d["Condition"].cat.categories)
        plt.cla()
        dabest_obj.mean_diff.plot(fig_size=(20,5))
        plt.tight_layout()
        plt.savefig(os.path.join(out_folder, g+".svg"))
        dabest_obj.mean_diff.statistical_tests.to_csv(os.path.join(out_folder, g+"_stats.tsv"), index=False, sep="\t")


@click.command()
@click.option("--file_path", "-f", help="Path to the input file")
@click.option("--index_col", "-i", help="Name of the index column")
@click.option("--selected_protein", "-p", help="List of selected proteins", multiple=True)
@click.option("--sample_annotation", "-s", help="Path to the sample annotation file")
@click.option("--out_folder", "-o", help="Path to the output folder")
@click.option("--log2", "-l", help="Log2 transform the data", is_flag=True)
@click.option("--condition_order", "-c", help="Order of the conditions")
def main(file_path: str, index_col: str, selected_protein: list[str], sample_annotation: str, out_folder: str, log2: bool, condition_order: str):
    est_plot(file_path, index_col, selected_protein, sample_annotation, out_folder, log2, condition_order.split(","))


if __name__ == "__main__":
    main()