import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.patches import Circle
from matplotlib.colors import Normalize, LinearSegmentedColormap
import argparse
import sys


def load_and_process_data(file_path, separator='\t'):
    """Load and process the data file.

    Args:
        file_path: Path to input file
        separator: Column separator (default: tab)

    Returns:
        tuple: (genes, cell_types, processed_data) or (None, None, None) if error
    """
    try:
        df = pd.read_csv(file_path, sep=separator, na_values=['NA', '#VALUE!'])
        df = df.dropna(axis=1, how='all')

        if df.empty or len(df.columns) < 2:
            print("Error: Data file must have at least 2 columns (genes and at least one cell type)", file=sys.stderr)
            return None, None, None

        genes = df.iloc[:, 0].values
        cell_types = df.columns[1:]
        data = df[cell_types].values.astype(float)

        return genes, cell_types, data

    except Exception as e:
        print(f"Error processing file: {str(e)}", file=sys.stderr)
        return None, None, None


def create_bubble_plot(genes, cell_types, data, log_transform=True,
                      colormap_name='yellow_red', min_radius=0.2, max_radius=0.5):
    """Create bubble plot visualization.

    Args:
        genes: Array of gene names
        cell_types: Array of cell type names
        data: 2D array of intensity values
        log_transform: Whether to apply log10 transformation
        colormap_name: Color scheme name
        min_radius: Minimum bubble radius
        max_radius: Maximum bubble radius

    Returns:
        matplotlib.figure.Figure: The created figure
    """
    processed_data = data.copy()
    if log_transform:
        processed_data = np.log10(processed_data + 1e-10)

    colormaps = {
        'yellow_red': LinearSegmentedColormap.from_list('yellow_red', ['yellow', 'red']),
        'blue_red': LinearSegmentedColormap.from_list('blue_red', ['blue', 'red']),
        'green_purple': LinearSegmentedColormap.from_list('green_purple', ['green', 'purple']),
        'viridis': plt.cm.viridis,
        'plasma': plt.cm.plasma
    }
    cmap = colormaps.get(colormap_name, colormaps['yellow_red'])

    norm = Normalize(vmin=np.nanmin(processed_data), vmax=np.nanmax(processed_data))

    fig_width = max(8, len(genes) * 0.5)
    fig_height = max(6, len(cell_types) * 0.5)
    fig, ax = plt.subplots(figsize=(fig_width, fig_height))

    for i, gene in enumerate(genes):
        for j, cell in enumerate(cell_types):
            value = processed_data[i, j]
            if np.isnan(value) or np.isinf(value):
                continue

            norm_value = norm(value)
            radius = min_radius + (max_radius - min_radius) * norm_value
            color = cmap(norm_value)
            circle = Circle((i, j), radius=radius, color=color)
            ax.add_patch(circle)

    ax.set_xticks(np.arange(len(genes)))
    ax.set_xticklabels(genes, rotation=90, fontsize=8)
    ax.set_yticks(np.arange(len(cell_types)))
    ax.set_yticklabels(cell_types, fontsize=8)
    ax.set_xlim(-0.5, len(genes) - 0.5)
    ax.set_ylim(-0.5, len(cell_types) - 0.5)
    ax.invert_yaxis()
    ax.set_aspect('equal')

    sm = plt.cm.ScalarMappable(cmap=cmap, norm=norm)
    sm.set_array([])
    cbar = plt.colorbar(sm, ax=ax, fraction=0.046, pad=0.04, shrink=0.3, aspect=20)
    cbar.set_label('Intensity' + (' (log10)' if log_transform else ''),
                   rotation=270, labelpad=15)
    cbar.ax.tick_params(labelsize=8)

    plt.tight_layout()
    return fig


def main():
    parser = argparse.ArgumentParser(
        description='Create bubble plot heat map visualization from tabular data'
    )
    parser.add_argument('--input_file', required=True, help='Input data file path')
    parser.add_argument('--output_folder', required=True, help='Output folder path')
    parser.add_argument('--separator', default='\t',
                       choices=['\t', ',', ';'],
                       help='Column separator (tab, comma, semicolon)')
    parser.add_argument('--log_transform', action='store_true', default=True,
                       help='Apply log10 transformation (default: True)')
    parser.add_argument('--no_log_transform', dest='log_transform', action='store_false',
                       help='Do not apply log10 transformation')
    parser.add_argument('--colormap', default='yellow_red',
                       choices=['yellow_red', 'blue_red', 'green_purple', 'viridis', 'plasma'],
                       help='Color scheme for the plot')
    parser.add_argument('--min_radius', type=float, default=0.2,
                       help='Minimum bubble radius (default: 0.2)')
    parser.add_argument('--max_radius', type=float, default=0.5,
                       help='Maximum bubble radius (default: 0.5)')
    parser.add_argument('--output_format', default='both',
                       choices=['png', 'pdf', 'both'],
                       help='Output format (default: both)')
    parser.add_argument('--dpi', type=int, default=300,
                       help='Output resolution in DPI (default: 300)')

    args = parser.parse_args()

    print(f"Loading data from {args.input_file}...")
    genes, cell_types, data = load_and_process_data(args.input_file, args.separator)

    if genes is None:
        sys.exit(1)

    print(f"Data loaded successfully: {len(genes)} features × {len(cell_types)} samples")

    print("Generating bubble plot...")
    fig = create_bubble_plot(
        genes, cell_types, data,
        log_transform=args.log_transform,
        colormap_name=args.colormap,
        min_radius=args.min_radius,
        max_radius=args.max_radius
    )

    import os
    os.makedirs(args.output_folder, exist_ok=True)

    if args.output_format in ['png', 'both']:
        png_path = os.path.join(args.output_folder, 'bubble_plot.png')
        print(f"Saving PNG to {png_path}...")
        fig.savefig(png_path, format='png', dpi=args.dpi, bbox_inches='tight')
        print(f"PNG saved successfully")

    if args.output_format in ['pdf', 'both']:
        pdf_path = os.path.join(args.output_folder, 'bubble_plot.pdf')
        print(f"Saving PDF to {pdf_path}...")
        fig.savefig(pdf_path, format='pdf', dpi=args.dpi, bbox_inches='tight')
        print(f"PDF saved successfully")

    plt.close(fig)
    print("Bubble plot generation completed successfully")


if __name__ == "__main__":
    main()
