import re
import click
import numpy as np
from scipy.stats import variation
from glob import glob
import os
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

@click.command()
@click.option("--log_file_path", "-f", help="Path to the file to be processed", default="")
@click.option("--report_pr_file_path", "-r", help="Path to the file to be processed", default="")
@click.option("--report_pg_file_path", "-g", help="Path to the file to be processed", default="")
@click.option("--intensity_col", "-i", help="Name of the intensity column", default="Intensity")
@click.option("--annotation_file", "-a", help="Path to the annotation file", default="")
@click.option("--output_file", "-o", help="Path to the output file", default="./diann_output.txt")
@click.option("--sample_names", "-s", help="Sample names delimited with ','", default="")
def diann_main(log_file_path: str, report_pr_file_path: str, report_pg_file_path: str, intensity_col: str, annotation_file: str, output_file: str, sample_names: str):
    """
    Main function to process DIA-NN output files and generate coefficient of variation (CV) plots.

    :param log_file_path: Path to the log file to extract sample names.
    :param report_pr_file_path: Path to the protein report file.
    :param report_pg_file_path: Path to the protein group report file.
    :param intensity_col: Name of the intensity column.
    :param annotation_file: Path to the annotation file.
    :param output_file: Path to the output file.
    :param sample_names: Comma-delimited sample names.
    """
    main(log_file_path, report_pr_file_path, report_pg_file_path, intensity_col, annotation_file, output_file, sample_names)

def main(log_file_path: str, report_pr_file_path: str, report_pg_file_path: str, intensity_col: str, annotation_file: str, output_file: str, sample_names: str):
    """
    Core function to process DIA-NN output files and generate CV plots.

    :param log_file_path: Path to the log file to extract sample names.
    :param report_pr_file_path: Path to the protein report file.
    :param report_pg_file_path: Path to the protein group report file.
    :param intensity_col: Name of the intensity column.
    :param annotation_file: Path to the annotation file.
    :param output_file: Path to the output file.
    :param sample_names: Comma-delimited sample names.
    """
    if log_file_path == "":
        samples = sample_names.split(",")
    else:
        samples = extract_filename(log_file_path)
    assert len(samples) > 0, "Please provide sample names"
    annotation = pd.read_csv(annotation_file, sep="\t")
    os.makedirs(output_file, exist_ok=True)
    if report_pr_file_path != "":
        pr_df = pd.read_csv(report_pr_file_path, sep="\t")
        pr_df = pr_df.melt(id_vars=[k for k in pr_df.columns if k not in samples], value_vars=samples, var_name="Sample", value_name=intensity_col)
        pr_df["Sample"] = pd.Categorical(pr_df["Sample"].values, samples)
        pr_df = pr_df.merge(annotation, on="Sample", how="left")
        pr_df = pr_df.groupby(["Condition", "Protein.Group", "Modified.Sequence"]).apply(lambda x: variation(x[intensity_col], nan_policy="omit")).reset_index()
        pr_df.rename(columns={0: "CV"}, inplace=True)
        draw_cv_intensity(pr_df,  os.path.join(output_file, "pr_cv.svg"))
    if report_pg_file_path != "":
        pg_df = pd.read_csv(report_pg_file_path, sep="\t")
        pg_df = pg_df.melt(id_vars=[k for k in pg_df.columns if k not in samples], value_vars=samples, var_name="Sample", value_name=intensity_col)
        pg_df["Sample"] = pd.Categorical(pg_df["Sample"].values, samples)
        pg_df = pg_df.merge(annotation, on="Sample", how="left")
        pg_df = pg_df.groupby(["Condition", "Protein.Group"]).apply(lambda x: variation(x[intensity_col], nan_policy="omit")).reset_index()
        pg_df.rename(columns={0: "CV"}, inplace=True)
        draw_cv_intensity(pg_df, os.path.join(output_file, "pg_cv.svg"))

def extract_filename(log_file_path):
    """
    Extract sample names from the log file.

    :param log_file_path: Path to the log file.
    :return: List of sample names.
    """
    samples = []
    with open(log_file_path, "rt") as log_stuff:
        for line in log_stuff:
            line = line.strip()
            if "Loading run" in line:
                match = re.search("Loading run (.+)", line)
                if match:
                    if match.group(1) not in samples:
                        samples.append(match.group(1))
    return samples

def draw_cv_intensity(df, output_file: str):
    """
    Draw and save the coefficient of variation (CV) intensity plot.

    :param df: DataFrame containing CV values.
    :param output_file: Path to the output file to save the plot.
    """
    custom_legend = {}
    for label, group_data in df.groupby(["Condition"]):
        label = label[0]
        fig, ax = plt.subplots()
        title = label + " ({})".format(np.round(group_data["CV"].median(), 2))
        custom_legend[label] = title
        sns.kdeplot(data=group_data, x="CV", linewidth=5, ax=ax).set(title=title)

    fig, ax = plt.subplots()
    sns.kdeplot(data=df.assign(Label=df["Condition"].map(custom_legend)), hue="Label", x="CV", linewidth=5, ax=ax)
    fig.savefig(output_file, dpi=300)

if __name__ == '__main__':
    diann_main()