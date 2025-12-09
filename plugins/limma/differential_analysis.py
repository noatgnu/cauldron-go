import pandas as pd
import click
import os


def set_up_R_HOME(value: str):
    os.environ["R_HOME"] = value


def diff_analysis(input_file: str, output_folder: str, annotation_file: str, comparison_file: str, index_col: str,
                  log2: bool = False, aggregate_column: str = "", aggregate_method: str = "MsCoreUtils::robustSummary",
                  col_filter: float = 0.7, row_filter: float = 0.7, impute: str = "knn",
                  normalize: str = "quantiles.robust"):
    from coral.data import Coral
    from coral.utility import detect_delimiter_from_extension
    coral = Coral()
    coral.load_unproccessed_file(input_file, sep=detect_delimiter_from_extension(input_file))
    annotation_df = pd.read_csv(annotation_file, sep=detect_delimiter_from_extension(annotation_file))
    comparison_df = pd.read_csv(comparison_file, sep=detect_delimiter_from_extension(comparison_file))
    index = index_col.split(",")
    for i, r in annotation_df.iterrows():
        coral.add_sample(r["Sample"])
        if r["Condition"] not in coral.conditions:
            coral.add_condition(r["Condition"])
        coral.add_condition_map(r["Condition"], [r["Sample"]])

    for i, r in comparison_df.iterrows():
        coral.add_comparison(r["condition_A"], r["condition_B"], r["comparison_label"])
    coral.index_columns = index
    #count missing for each column in unprocessed_df
    print(coral.unprocessed_df.isnull().sum())
    #count proportion of missing values for each column in unprocessed_df
    print(coral.unprocessed_df.isnull().sum() / len(coral.unprocessed_df))


    coral.filter_missing_columns(col_filter)
    coral.prepare()
    coral.filter_missing_rows(row_filter)
    #if impute != "knn":
    if impute != "":
        coral.impute(impute)
        coral.export_df_from_R(os.path.join(output_folder, "imputed.txt").replace("\\", "/"))
    if log2:
        coral.log_transform()
    if aggregate_column:
        coral.aggregate_features(aggregate_column, aggregate_method)
        coral.export_df_from_R(os.path.join(output_folder, "aggregated.txt").replace("\\", "/"))
    if normalize:
        coral.normalize(normalize)
        coral.export_df_from_R(os.path.join(output_folder, "normalized.txt").replace("\\", "/"))
    #if impute == "knn":
    #    coral.impute(impute)
    #    coral.export_df_from_R(os.path.join(output_folder, "imputed.txt").replace("\\", "/"))
    coral.prepare_for_limma()
    result = []
    for d in coral.run_limma():
        result.append(d)
    if len(result) > 1:
        result = pd.concat(result)
    else:
        result = result[0]
    os.makedirs(output_folder, exist_ok=True)
    result.to_csv(os.path.join(output_folder, "differential_analysis.txt"), sep="\t", index=False)

@click.command()
@click.option("--input_file", "-i", help="Path to the input file")
@click.option("--output_folder", "-o", help="Path to the output folder")
@click.option("--annotation_file", "-a", help="Path to the annotation file")
@click.option("--comparison_file", "-c", help="Path to the comparison file")
@click.option("--index_col", "-x", help="Name of the index column")
@click.option("--log2", "-l", help="Log2 transform the data", is_flag=True)
@click.option("--aggregate_column", "-g", help="Name of the column to aggregate")
@click.option("--aggregate_method", "-m", help="Method to aggregate the data", default="MsCoreUtils::robustSummary")
@click.option("--col_filter", "-f", help="Filter unprocessed data columns with missing values more than threshold", default=0.7)
@click.option("--row_filter", "-r", help="Filter unprocessed data rows with missing values more than threshold", default=0.7)
@click.option("--impute", "-p", help="Impute missing values with method", default="")
@click.option("--normalize", "-n", help="Normalize data with method", default="")
@click.option("--r_home", "-r", help="Path to the R home")
def main(input_file: str, output_folder: str, annotation_file: str, comparison_file: str, index_col: str, log2: bool = False,
         aggregate_column: str = "", aggregate_method: str = "MsCoreUtils::robustSummary",  col_filter: float = 0.7,
         row_filter: float = 0.7, impute: str = "", normalize: str = "",r_home: str = ""):
    if r_home:
        set_up_R_HOME(r_home)
    diff_analysis(input_file, output_folder, annotation_file, comparison_file, index_col, log2, aggregate_column,
                  aggregate_method, col_filter, row_filter, impute, normalize)

if __name__ == "__main__":
    main()

