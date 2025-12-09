"""
Dose-response curve analysis using the dra package.
"""

import os

import click
import pandas as pd
import numpy as np
from dra import DoseResponseAnalyzer, DoseResponsePlotter


def create_overlay_plot(
    results: dict,
    analyzer: DoseResponseAnalyzer,
    df: pd.DataFrame,
    compounds_to_overlay: list[str],
    compound_colors: dict,
    output_folder: str,
    show_ic50_lines: bool = True,
    file_format: str = "svg",
    add_timestamp: bool = True,
):
    """
    Create overlay plot with multiple compounds on the same axes.

    Args:
        results: Results from analyzer.fit_best_models()
        analyzer: DoseResponseAnalyzer instance
        df: Original data
        compounds_to_overlay: List of compound names to overlay
        compound_colors: Dictionary mapping compound names to colors
        output_folder: Output directory
        show_ic50_lines: Whether to show IC50 reference lines
        file_format: Output file format
        add_timestamp: Whether to add timestamp to filename
    """
    try:
        import matplotlib.pyplot as plt
        from datetime import datetime
        from pathlib import Path

        concentration_col = analyzer.columns["concentration"]
        response_col = analyzer.columns["response"]
        compound_col = analyzer.columns["compound"]

        data_filtered = df[df[concentration_col] > 0].copy()

        default_colors = [
            "#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd",
            "#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf"
        ]

        marker_styles = ["o", "s", "^", "D", "v", "<", ">", "p", "*", "h"]

        fig, ax = plt.subplots(figsize=(10, 7))

        x_min_global = float('inf')
        x_max_global = float('-inf')

        legend_elements = []
        ic50_values = []

        for idx, compound in enumerate(compounds_to_overlay):
            if compound not in results["best_fitted_models"]:
                print(f"Warning: Compound '{compound}' not found in results. Skipping.")
                continue

            model_data = results["best_fitted_models"][compound]
            compound_data = data_filtered[data_filtered[compound_col] == compound].copy()
            model_result = model_data["model_result"]

            color = compound_colors.get(compound, default_colors[idx % len(default_colors)])
            marker = marker_styles[idx % len(marker_styles)]

            x_min = compound_data[concentration_col].min()
            x_max = compound_data[concentration_col].max()
            x_min_global = min(x_min_global, x_min)
            x_max_global = max(x_max_global, x_max)

            conc_smooth, response_smooth = analyzer.predict_curve(
                model_data, concentration_range=(x_min, x_max), n_points=200
            )

            ax.scatter(
                compound_data[concentration_col],
                compound_data[response_col],
                color=color,
                s=50,
                alpha=0.6,
                marker=marker,
                zorder=3,
                label=f"{compound} (data)"
            )

            ax.plot(
                conc_smooth,
                response_smooth,
                color=color,
                linewidth=2.5,
                label=f"{compound} ({model_result['model_name']})",
                zorder=2,
            )

            plotter = DoseResponsePlotter()
            params = plotter._extract_model_parameters(model_result)
            ic50 = params["ic50"]

            if show_ic50_lines and not np.isnan(ic50):
                ic50_values.append((compound, ic50, color))

        xlim_extended = [x_min_global / 10, x_max_global * 10]
        log_min = int(np.floor(np.log10(xlim_extended[0])))
        log_max = int(np.ceil(np.log10(xlim_extended[1])))

        if show_ic50_lines and ic50_values:
            for compound, ic50, color in ic50_values:
                ax.axvline(
                    x=ic50,
                    color=color,
                    linestyle="--",
                    linewidth=1.5,
                    alpha=0.7,
                    zorder=1,
                )

                y_position = ax.get_ylim()[1] * (0.95 - 0.05 * ic50_values.index((compound, ic50, color)))
                ax.text(
                    ic50 * 1.1,
                    y_position,
                    f"{compound} IC₅₀ = {ic50:.1f}",
                    color=color,
                    fontsize=9,
                    fontweight="bold",
                    bbox=dict(boxstyle="round,pad=0.3", facecolor="white", alpha=0.8),
                )

        ax.set_xscale("log")
        ax.set_xlim(xlim_extended)

        log_range = range(log_min, log_max + 1)
        x_ticks = [
            10**i for i in log_range
            if 10**i >= xlim_extended[0] and 10**i <= xlim_extended[1]
        ]
        ax.set_xticks(x_ticks)
        ax.set_xticklabels([f"{tick:g}" for tick in x_ticks])

        ax.set_xlabel(f"{concentration_col}", fontsize=12)
        ax.set_ylabel(f"{response_col}", fontsize=12)
        ax.set_title("Dose-Response Curves Overlay", fontsize=14, fontweight="bold")
        ax.grid(True, alpha=0.3, which="both")
        ax.legend(fontsize=9, loc="best", framealpha=0.9)

        plt.tight_layout()

        output_path = Path(output_folder)
        output_path.mkdir(parents=True, exist_ok=True)

        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S") if add_timestamp else ""
        timestamp_suffix = f"_{timestamp}" if timestamp else ""

        overlay_filename = f"dose_response_overlay{timestamp_suffix}.{file_format}"
        overlay_path = output_path / overlay_filename

        plt.savefig(overlay_path, dpi=300, bbox_inches="tight")
        plt.close()

        print(f"Overlay plot saved as: {overlay_path}")

    except ImportError:
        print("Matplotlib not available. Install it to generate plots:")
        print("pip install matplotlib")
    except Exception as e:
        print(f"Error creating overlay plot: {str(e)}")


def dose_response_analysis(
    input_file: str,
    output_folder: str,
    compound_col: str = "Compound",
    concentration_col: str = "Conc",
    response_col: str = "Rab10",
    show_ic50_lines: bool = True,
    show_dmax_lines: bool = True,
    file_format: str = "svg",
    add_timestamp: bool = True,
    enable_custom_models: bool = True,
    selection_metric: str = "rmse",
    overlay_compounds: str = "",
    compound_colors: str = "",
):
    """
    Perform dose-response curve analysis on input data.

    Args:
        input_file: Path to input file (CSV or TSV)
        output_folder: Path to output folder
        compound_col: Column name for compound identifiers
        concentration_col: Column name for concentration values
        response_col: Column name for response values
        show_ic50_lines: Whether to show IC50 reference lines
        show_dmax_lines: Whether to show Dmax reference lines
        file_format: Output file format (svg, png, pdf)
        add_timestamp: Whether to add timestamp to filenames
        enable_custom_models: Whether to include extended model set
        selection_metric: Metric for model selection (rmse, aic, bic, r2)
        overlay_compounds: Comma-separated list of compounds to overlay
        compound_colors: Semicolon-separated compound:color pairs (e.g., "CompoundA:#ff0000;CompoundB:#00ff00")
    """
    if input_file.endswith(".tsv") or input_file.endswith(".txt"):
        df = pd.read_csv(input_file, sep="\t")
    elif input_file.endswith(".csv"):
        df = pd.read_csv(input_file, sep=",")
    else:
        raise ValueError("Invalid file extension. Use .csv, .tsv, or .txt")

    column_mapping = {
        "compound": compound_col,
        "concentration": concentration_col,
        "response": response_col,
    }

    analyzer = DoseResponseAnalyzer(
        column_mapping=column_mapping,
        enable_custom_models=enable_custom_models,
        selection_metric=selection_metric,
    )

    print("Fitting dose-response models...")
    results = analyzer.fit_best_models(df)

    os.makedirs(output_folder, exist_ok=True)

    results["summary_table"].to_csv(
        os.path.join(output_folder, "summary_table.txt"), sep="\t", index=False
    )
    results["best_models"].to_csv(
        os.path.join(output_folder, "best_models.txt"), sep="\t", index=False
    )

    print("\n=== BEST MODELS ===")
    print(results["best_models"])

    print("\n=== IC50 VALUES ===")
    for _, row in results["best_models"].iterrows():
        print(
            f"{row['Compound']}: IC50 = {row['IC50']:.2f} (Model: {row['Model']}, RMSE: {row['RMSE']:.4f})"
        )

    plotter = DoseResponsePlotter()
    plotter.plot_dose_response_curves(
        results,
        analyzer,
        df,
        show_ic50_lines=show_ic50_lines,
        show_dmax_lines=show_dmax_lines,
        output_dir=output_folder,
        filename_prefix="dose_response",
        add_timestamp=add_timestamp,
        file_format=file_format,
    )

    if overlay_compounds:
        compounds_to_overlay = [c.strip() for c in overlay_compounds.split(",")]

        color_mapping = {}
        if compound_colors:
            for pair in compound_colors.split(";"):
                if ":" in pair:
                    compound, color = pair.split(":", 1)
                    color_mapping[compound.strip()] = color.strip()

        create_overlay_plot(
            results,
            analyzer,
            df,
            compounds_to_overlay,
            color_mapping,
            output_folder,
            show_ic50_lines,
            file_format,
            add_timestamp,
        )

    print(f"\nResults saved to: {output_folder}")


def run_example():
    """
    Run example analysis with synthetic dose-response data.

    Creates synthetic data for two compounds (Compound_A and Compound_B) with different
    dose-response characteristics and demonstrates the complete analysis workflow including
    individual plots and overlay visualization.
    """
    print("=" * 60)
    print("Running Example Dose-Response Analysis")
    print("=" * 60)

    np.random.seed(42)
    compounds = ["Compound_A", "Compound_B", "Compound_C"]
    concentrations = [0.1, 1, 10, 100, 1000, 10000]

    data_list = []
    for compound in compounds:
        for conc in concentrations:
            if compound == "Compound_A":
                true_response = 0.1 + (1.0 - 0.1) / (1 + (conc / 100) ** 1.5)
            elif compound == "Compound_B":
                true_response = 0.05 + (0.9 - 0.05) / (1 + (conc / 500) ** 2.0)
            else:
                true_response = 0.15 + (0.95 - 0.15) / (1 + (conc / 250) ** 1.8)

            for rep in range(3):
                noisy_response = true_response + np.random.normal(0, 0.05)
                data_list.append(
                    {"Compound": compound, "Conc": conc, "Rab10": max(0, noisy_response)}
                )

    data = pd.DataFrame(data_list)

    output_folder = "example_dose_response_output"
    os.makedirs(output_folder, exist_ok=True)

    data.to_csv(os.path.join(output_folder, "example_data.txt"), sep="\t", index=False)
    print(f"\nExample data saved to: {os.path.join(output_folder, 'example_data.txt')}")

    dose_response_analysis(
        input_file=os.path.join(output_folder, "example_data.txt"),
        output_folder=output_folder,
        compound_col="Compound",
        concentration_col="Conc",
        response_col="Rab10",
        show_ic50_lines=True,
        show_dmax_lines=True,
        file_format="png",
        add_timestamp=True,
        enable_custom_models=True,
        selection_metric="rmse",
        overlay_compounds="Compound_A,Compound_B,Compound_C",
        compound_colors="Compound_A:#e74c3c;Compound_B:#3498db;Compound_C:#2ecc71",
    )

    print("\n" + "=" * 60)
    print("Example analysis complete!")
    print(f"Check the '{output_folder}' directory for results.")
    print("=" * 60)


@click.command()
@click.option("--input_file", "-i", help="Path to the input file (CSV or TSV)")
@click.option("--output_folder", "-o", help="Path to the output folder")
@click.option("--compound_col", "-c", help="Column name for compound identifiers", default="Compound")
@click.option("--concentration_col", "-n", help="Column name for concentration values", default="Conc")
@click.option("--response_col", "-r", help="Column name for response values", default="Rab10")
@click.option("--show_ic50_lines", "-ic50", help="Show IC50 reference lines", is_flag=True, default=True)
@click.option("--show_dmax_lines", "-dmax", help="Show Dmax reference lines", is_flag=True, default=True)
@click.option("--file_format", "-f", help="Output file format (svg, png, pdf)", default="svg")
@click.option("--add_timestamp", "-t", help="Add timestamp to filenames", is_flag=True, default=True)
@click.option("--enable_custom_models", "-m", help="Include extended model set", is_flag=True, default=True)
@click.option("--selection_metric", "-s", help="Metric for model selection (rmse, aic, bic, r2)", default="rmse")
@click.option("--overlay_compounds", "-y", help="Comma-separated list of compounds to overlay on one plot (e.g., 'CompoundA,CompoundB,CompoundC')", default="")
@click.option("--compound_colors", "-l", help="Semicolon-separated compound:color pairs (e.g., 'CompoundA:#ff0000;CompoundB:#00ff00')", default="")
@click.option("--example", "-e", help="Run example analysis with synthetic data", is_flag=True, default=False)
def main(
    input_file: str,
    output_folder: str,
    compound_col: str,
    concentration_col: str,
    response_col: str,
    show_ic50_lines: bool,
    show_dmax_lines: bool,
    file_format: str,
    add_timestamp: bool,
    enable_custom_models: bool,
    selection_metric: str,
    overlay_compounds: str,
    compound_colors: str,
    example: bool,
):
    """
    Dose-response curve analysis command-line tool.

    Fits multiple dose-response models to input data and generates visualization plots.

    Examples:

    Run example with synthetic data:
        python dose_response_analyzer.py -e

    Basic usage:
        python dose_response_analyzer.py -i data.csv -o results/

    Overlay specific compounds with custom colors:
        python dose_response_analyzer.py -i data.csv -o results/ -y "CompoundA,CompoundB,CompoundC" -l "CompoundA:#ff0000;CompoundB:#00ff00;CompoundC:#0000ff"

    Overlay compounds with default colors:
        python dose_response_analyzer.py -i data.csv -o results/ -y "CompoundA,CompoundB"
    """
    if example:
        run_example()
    else:
        if not input_file or not output_folder:
            print("Error: --input_file and --output_folder are required when not running example mode.")
            print("Use -e or --example to run with synthetic example data.")
            return

        dose_response_analysis(
            input_file,
            output_folder,
            compound_col,
            concentration_col,
            response_col,
            show_ic50_lines,
            show_dmax_lines,
            file_format,
            add_timestamp,
            enable_custom_models,
            selection_metric,
            overlay_compounds,
            compound_colors,
        )


if __name__ == "__main__":
    main()
