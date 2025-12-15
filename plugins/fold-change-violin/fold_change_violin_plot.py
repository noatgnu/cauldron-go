import matplotlib
matplotlib.use('Agg')

import os
import click
import seaborn as sns
import pandas as pd
import matplotlib.pyplot as plt


def load_data(file_path):
    return pd.read_csv(file_path, sep="\t")


def process_data(df, columns_prefix, categories, match_value, fold_enrichment_col, organelle_col, comparison_col=None):
    result = []
    for i in df.columns:
        if columns_prefix:
            if i.startswith(columns_prefix):
                for i2, r in df.iterrows():
                    for j in categories:
                        if r[j] == match_value:
                            row = [r[i], j]
                            if comparison_col:
                                row.append(i)
                            result.append(row)
                            break
        else:
            for i2, r in df.iterrows():
                for j in categories:
                    if r[j] == match_value:
                        row = [r[i], j]
                        if comparison_col:
                            row.append(i)
                        result.append(row)
                        break

    columns = [fold_enrichment_col, organelle_col]
    if comparison_col:
        columns.append(comparison_col)

    result = pd.DataFrame(result, columns=columns)
    result[organelle_col] = pd.Categorical(result[organelle_col], categories, ordered=True)
    return result


def plot_data(result, fold_enrichment_col, organelle_col, comparison_col, colors, figsize, output_folder=None):
    sns.set(font="Arial")
    sns.set(rc={'figure.figsize': figsize})
    sns.set_style("white")

    if output_folder:
        os.makedirs(output_folder, exist_ok=True)

    if comparison_col and comparison_col in result.columns:
        grouped = result.groupby(comparison_col)
    else:
        grouped = [(None, result)]

    for i, data in grouped:
        plt.cla()
        g = sns.violinplot(data=data, y=organelle_col, x=fold_enrichment_col, orient="h", hue=organelle_col,
                           palette=colors, linewidth=1, linecolor="black")

        sns.stripplot(data=data, y=organelle_col, x=fold_enrichment_col, orient="h", linewidth=1, color="#ebebeb",
                      alpha=0.5)
        fig = g.get_figure()
        fig.tight_layout()
        if i:
            base_filename = f"{i.replace(':', '')}"
        else:
            base_filename = "plot"

        if output_folder:
            filepath_pdf = os.path.join(output_folder, f"{base_filename}.pdf")
            filepath_svg = os.path.join(output_folder, f"{base_filename}.svg")
        else:
            filepath_pdf = f"{base_filename}.pdf"
            filepath_svg = f"{base_filename}.svg"

        fig.savefig(filepath_pdf)
        print(f"Saved plot as {filepath_pdf}")
        fig.savefig(filepath_svg)
        print(f"Saved plot as {filepath_svg}")
        plt.close(fig)


@click.command()
@click.option('--file_path', prompt='File path', help='The path to the input CSV file.')
@click.option('--columns_prefix', default='Difference', help='Prefix of the columns to be included.')
@click.option('--categories', default='C..Lysosome,C..Mitochondria,Endosome,Golgi,ER,Ribosome,Nuclear',
              help='Comma-separated list of categories to be used for filtering.')
@click.option('--match_value', default='+', help='The symbol or string to match in the data.')
@click.option('--fold_enrichment_col', default='Fold enrichment', help='Name of the column for fold enrichment.')
@click.option('--organelle_col', default='Organelle', help='Name of the column for organelle.')
@click.option('--comparison_col', default='Comparison',
              help='Name of the column for comparison. Set to "" to make it optional.', required=False)
@click.option('--colors',
              default='C..Lysosome:#ffb3ba,C..Mitochondria:#ffdfba,Endosome:#ffffba,Golgi:#baffc9,ER:#bae1ff,Ribosome:#d7baff,Nuclear:#ffbaff',
              help='Comma-separated list of colors for each category in the format category:color.')
@click.option('--figsize', default='6,10', help='Figure size in inches, in the format width,height.')
@click.option('--output_folder', default=None, help='Output folder for saving plots.')
def main(file_path, columns_prefix, categories, match_value, fold_enrichment_col, organelle_col, comparison_col, colors,
         figsize, output_folder):
    categories_list = categories.split(',')
    comparison_col = comparison_col if comparison_col else None
    colors_dict = dict(item.split(":") for item in colors.split(","))
    figsize_tuple = tuple(map(int, figsize.split(',')))
    df = load_data(file_path)
    result = process_data(df, columns_prefix, categories_list, match_value, fold_enrichment_col, organelle_col,
                          comparison_col)
    plot_data(result, fold_enrichment_col, organelle_col, comparison_col, colors_dict, figsize_tuple, output_folder)


if __name__ == '__main__':
    main()
