import click

import pandas as pd
import os
import sys

from coral.utility import replace_special_with_dot


def r_correlation_matrix(
        file_path: str,
        output_folder: str,
        index_col: str,
        sample_cols: list[str],
        method: str = "pearson",
        min_value: float|None = None,
        order: str = "hclust",
        hclust_method: str = "ward.D",
        presenting_method: str = "ellipse",
        cor_shape: str = "upper",
        plot_only: bool = False,
        color_ramp_palette: str = "#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F"
):

    import rpy2.robjects as ro
    from rpy2.robjects.packages import importr
    from rpy2.robjects import pandas2ri
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"File {file_path} does not exist")
    if file_path.endswith(".csv"):
        df = pd.read_csv(file_path)
    elif file_path.endswith(".tsv") or file_path.endswith(".txt"):
        df = pd.read_csv(file_path, sep="\t")
    else:
        raise ValueError(f"File format not supported: {file_path}")
    r_index_col = replace_special_with_dot(index_col)
    r_sample_cols = [os.path.split(i)[1] for i in sample_cols]
    column_replacement = {sample_cols[i]: replace_special_with_dot(r_sample_cols[i]) for i in range(len(sample_cols))}
    column_replacement[index_col] = r_index_col
    with (ro.default_converter + pandas2ri.converter).context():
        df.rename(columns=column_replacement,
                  inplace=True)
        r_from_pd_df = ro.conversion.get_conversion().py2rpy(df)
        corrplot = importr("corrplot")


    ro.globalenv["data"] = r_from_pd_df
    ro.globalenv["selectedColumns"] = ro.StrVector(
        column_replacement.values())
    ro.globalenv["col"] = ro.StrVector(color_ramp_palette.split(","))
    ro.r(f"""
    data <- data[selectedColumns]
    rownames(data) <- data${r_index_col}
    data${r_index_col} <- NULL
    col <- colorRampPalette(col)(200)
    """)
    if not plot_only:
        ro.r(f"""
        data <- cor(data, method="{method}")
        """)

    ro.r("""
    data[is.na(data)] <- 1
    """)

    if min_value:
        ro.r(f"""
        min.value <- {min_value}
        """)
    else:
        ro.r("""
        min.value <- round(min(data), 1) - 0.1
        """)
    ro.r("""
    max.value <- 1
    """)

    output_file = os.path.join(output_folder, "correlation_matrix.txt").replace("\\", "/")
    output_plot = os.path.join(output_folder, "correlation_matrix.pdf").replace("\\", "/")


    if method:
        if order == "hclust":
            ro.r(f"""
            pdf("{output_plot}")
            cor.data <- corrplot(as.matrix(data), order="{order}", hclust.method="{hclust_method}", method="{presenting_method}", type="{cor_shape}", is.corr=FALSE, col.lim = c(min.value,max.value), col=col)
            dev.off()
            """)
        else:
            ro.r(f"""
            pdf("{output_plot}")
            cor.data <- corrplot(as.matrix(data), order="{order}", method="{presenting_method}", type="{cor_shape}", is.corr=FALSE, col.lim = c(min.value,max.value),  col=col)
            dev.off()
            """)
    else:
        if order == "hclust":
            ro.r(f"""
            pdf("{output_plot}")
            cor.data <- corrplot(as.matrix(data), order="{order}", hclust.method="{hclust_method}", method="{presenting_method}", type="{cor_shape}", is.corr=FALSE, col.lim = c(min.value,max.value),  col=col)
            dev.off()
            """)
        else:
            ro.r(f"""
            pdf("{output_plot}")
            cor.data <- corrplot(as.matrix(data), order="{order}", method="{presenting_method}", type="{cor_shape}", is.corr=FALSE, col.lim = c(min.value,max.value),  col=col)
            dev.off()
            """)
    result = ro.r("cor.data$corrPos")
    with (ro.default_converter + pandas2ri.converter).context():
        pd_from_r_df = ro.conversion.get_conversion().rpy2py(result)
        pd_from_r_df.to_csv(output_file, index=True, sep="\t")

def setup_r_home(r_home: str):
    if not os.path.exists(r_home):
        raise FileNotFoundError(f"R home directory {r_home} does not exist")
    os.environ["R_HOME"] = r_home


@click.command()
@click.option("--file_path", "-f", help="Path to the input file", type=str)
@click.option("--output_folder", "-o", help="Path to the output folder", type=str)
@click.option("--index_col", "-i", help="Name of the index column", type=str)
@click.option("--sample_cols", "-s", help="Name of the sample columns", type=str)
@click.option("--method", "-m", help="Method to calculate correlation", type=str, default="pearson")
@click.option("--min_value", "-min", help="Minimum value for the correlation color scale", type=float, default=None)
@click.option("--order", "-ord", help="Order of the correlation matrix", type=str, default=None)
@click.option("--hclust_method", "-hclust", help="Method for hierarchical clustering", type=str, default="ward.D")
@click.option("--presenting_method", "-pm", help="Method for presenting the correlation matrix", type=str, default="ellipse")
@click.option("--cor_shape", "-cs", help="Shape of the correlation matrix", type=str, default="upper")
@click.option("--r_home", "-rh", help="Path to the R home directory", type=str)
@click.option("--plot_only", "-po", help="Only plot the correlation matrix", is_flag=True)
@click.option("--color_ramp_palette", "-crp", help="Color ramp palette for the correlation matrix", type=str, default="#053061,#2166AC,#4393C3,#92C5DE,#D1E5F0,#FFFFFF,#FDDBC7,#F4A582,#D6604D,#B2182B,#67001F")
def main(file_path: str, output_folder: str, index_col: str, sample_cols: str, method: str, min_value: float, order: str, hclust_method: str, presenting_method: str, cor_shape: str, r_home: str, plot_only: bool, color_ramp_palette: str):
    setup_r_home(r_home)
    sample_cols = sample_cols.split(",")
    r_correlation_matrix(file_path, output_folder, index_col, sample_cols, method, min_value, order, hclust_method, presenting_method, cor_shape, plot_only, color_ramp_palette)
    sys.exit(0)

if __name__ == "__main__":
    main()