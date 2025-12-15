import matplotlib
matplotlib.use('Agg')

import numpy as np
import pandas as pd
import click
import os
import dabest
from matplotlib import pyplot as plt
import re

def est_plot(file_path: str, index_col: str, selected_protein: list[str], sample_annotation: str, output_folder: str, log2: bool = True, condition_order: list[str] = []):
    print(f"[Estimation Plot] Starting with parameters:")
    print(f"  file_path: {file_path}")
    print(f"  index_col: {index_col}")
    print(f"  selected_protein: {selected_protein}")
    print(f"  sample_annotation: {sample_annotation}")
    print(f"  output_folder: {output_folder}")
    print(f"  log2: {log2}")
    print(f"  condition_order: {condition_order}")

    if file_path.endswith(".txt") or file_path.endswith(".tsv"):
        df = pd.read_csv(file_path, sep="\t")
    elif file_path.endswith(".csv"):
        df = pd.read_csv(file_path)
    else:
        raise ValueError("File format not supported")

    print(f"[Estimation Plot] Loaded data with shape: {df.shape}")
    print(f"[Estimation Plot] Columns: {df.columns.tolist()}")
    print(f"[Estimation Plot] Index column '{index_col}' unique values: {df[index_col].nunique()}")

    if index_col in df.columns:
        print(f"[Estimation Plot] Sample values from '{index_col}' column:")
        sample_values = df[index_col].head(20).tolist()
        for i, val in enumerate(sample_values, 1):
            print(f"  {i}. {val}")

    print(f"[Estimation Plot] Looking for these proteins: {selected_protein}")

    original_df = df.copy()
    df = df[df[index_col].isin(selected_protein)]
    print(f"[Estimation Plot] After filtering for selected proteins, shape: {df.shape}")

    if df.empty:
        print(f"[Estimation Plot] ERROR: No data left after filtering! Check if selected proteins exist in data.")
        print(f"[Estimation Plot] Selected proteins that were searched for: {selected_protein}")
        print(f"[Estimation Plot] None of the selected proteins were found in the '{index_col}' column.")
        print(f"[Estimation Plot] First 20 actual values in '{index_col}':")
        for i, val in enumerate(original_df[index_col].head(20).tolist(), 1):
            print(f"  {i}. {val}")
        return

    if sample_annotation.endswith(".txt") or sample_annotation.endswith(".tsv"):
        sample_annotation = pd.read_csv(sample_annotation, sep="\t")
    elif sample_annotation.endswith(".csv"):
        sample_annotation = pd.read_csv(sample_annotation)
    else:
        raise ValueError("File format not supported")
    sample_annotation["Sample"] = sample_annotation["Sample"].astype(str)
    sample_annotation["Condition"] = sample_annotation["Condition"].astype(str)

    sample_annotation = sample_annotation.set_index("Sample").to_dict()["Condition"]
    print(f"[Estimation Plot] Sample annotation loaded: {len(sample_annotation)} samples")

    melted_df = df.melt(id_vars=[index_col], value_vars=sample_annotation.keys(), var_name="Sample", value_name="Value")
    melted_df["Condition"] = melted_df["Sample"].map(sample_annotation)

    os.makedirs(output_folder, exist_ok=True)
    print(f"[Estimation Plot] Created output folder: {output_folder}")

    plot_count = 0
    for g, d in melted_df.groupby(index_col):
        print(f"[Estimation Plot] Processing protein: {g}")
        g = re.sub(r'[^\w\s]', '_', g)
        for c in d["Condition"].unique():
            if d[d["Condition"] == c]["Value"].isna().all():
                d = d[d["Condition"] != c]

        if condition_order:
            d["Condition"] = pd.Categorical(d["Condition"], [i for i in condition_order if i in d["Condition"].values], ordered=True)
        else:
            d["Condition"] = pd.Categorical(d["Condition"], d["Condition"].unique(), ordered=True)
        if log2:
            d["Value"] = np.log2(d["Value"])
        print(f"[Estimation Plot] Conditions for {g}: {d['Condition'].cat.categories.tolist()}")

        try:
            dabest_obj = dabest.load(data=d, x="Condition", y="Value", idx=d["Condition"].cat.categories)
            plt.cla()
            dabest_obj.mean_diff.plot(fig_size=(20,5))
            plt.tight_layout()
            output_path = os.path.join(output_folder, g+".svg")
            plt.savefig(output_path)
            plt.close()
            print(f"[Estimation Plot] Saved plot: {output_path}")

            stats_path = os.path.join(output_folder, g+"_stats.tsv")
            dabest_obj.mean_diff.statistical_tests.to_csv(stats_path, index=False, sep="\t")
            print(f"[Estimation Plot] Saved stats: {stats_path}")
            plot_count += 1
        except Exception as e:
            print(f"[Estimation Plot] ERROR generating plot for {g}: {e}")
            import traceback
            traceback.print_exc()

    print(f"[Estimation Plot] Completed! Generated {plot_count} plots in {output_folder}")


@click.command()
@click.option("--file_path", "-f", help="Path to the input file", required=True)
@click.option("--index_col", "-i", help="Name of the index column", required=True)
@click.option("--selected_protein", "-p", help="Comma-separated list of selected proteins", required=True)
@click.option("--sample_annotation", "-s", help="Path to the sample annotation file", required=True)
@click.option("--output_folder", "-o", help="Path to the output folder", required=True)
@click.option("--log2", "-l", help="Log2 transform the data", is_flag=True)
@click.option("--condition_order", "-c", help="Order of the conditions", default="")
def main(file_path: str, index_col: str, selected_protein: str, sample_annotation: str, output_folder: str, log2: bool, condition_order: str):
    print(f"[Main] Received arguments:")
    print(f"  file_path: {file_path}")
    print(f"  index_col: {index_col}")
    print(f"  selected_protein: '{selected_protein}'")
    print(f"  sample_annotation: {sample_annotation}")
    print(f"  output_folder: {output_folder}")
    print(f"  log2: {log2}")
    print(f"  condition_order: '{condition_order}'")

    if not selected_protein:
        print("[Main] ERROR: No proteins selected!")
        return

    protein_list = [p.strip() for p in selected_protein.split(",") if p.strip()]
    print(f"[Main] Parsed protein list: {protein_list}")

    condition_list = [c.strip() for c in condition_order.split(",") if c.strip()] if condition_order else []
    print(f"[Main] Parsed condition list: {condition_list}")

    est_plot(file_path, index_col, protein_list, sample_annotation, output_folder, log2, condition_list)


if __name__ == "__main__":
    main()