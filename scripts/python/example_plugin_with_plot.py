#!/usr/bin/env python3
import sys
import os
import argparse
import pandas as pd
import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from cauldron_plot_utils import PlotlyPlotBuilder, create_scatter_plot, create_multi_scatter_plot


def main():
    parser = argparse.ArgumentParser(description='Sample custom analysis with plot generation')
    parser.add_argument('--input_file', required=True, help='Input data file')
    parser.add_argument('--threshold', type=float, default=0.05, help='Threshold value')
    parser.add_argument('--method', required=True, help='Analysis method')
    parser.add_argument('--normalize', type=bool, default=True, help='Normalize data')
    parser.add_argument('--output', required=True, help='Output directory')

    args = parser.parse_args()

    print(f"Running sample analysis...")
    print(f"Input file: {args.input_file}")
    print(f"Threshold: {args.threshold}")
    print(f"Method: {args.method}")
    print(f"Normalize: {args.normalize}")
    print(f"Output: {args.output}")

    np.random.seed(42)
    n_points = 50
    x_data = np.random.randn(n_points)
    y_data = 2 * x_data + np.random.randn(n_points) * 0.5

    sample_names = [f"Sample_{i+1}" for i in range(n_points)]

    output_plot_path = os.path.join(args.output, 'plot_1.json')

    create_scatter_plot(
        x=x_data.tolist(),
        y=y_data.tolist(),
        title="Sample Analysis Results",
        xaxis_title="Feature X",
        yaxis_title="Feature Y",
        text=sample_names,
        color="#1976d2",
        output_file=output_plot_path
    )

    print(f"Generated plot: {output_plot_path}")

    groups = {
        'Group A': {'x': [], 'y': [], 'text': []},
        'Group B': {'x': [], 'y': [], 'text': []}
    }

    for i in range(n_points):
        group = 'Group A' if y_data[i] > 0 else 'Group B'
        groups[group]['x'].append(x_data[i])
        groups[group]['y'].append(y_data[i])
        groups[group]['text'].append(sample_names[i])

    data_series = [
        {
            'x': groups['Group A']['x'],
            'y': groups['Group A']['y'],
            'text': groups['Group A']['text'],
            'name': 'Group A',
            'color': '#388e3c'
        },
        {
            'x': groups['Group B']['x'],
            'y': groups['Group B']['y'],
            'text': groups['Group B']['text'],
            'name': 'Group B',
            'color': '#d32f2f'
        }
    ]

    output_plot_2_path = os.path.join(args.output, 'plot_2.json')

    create_multi_scatter_plot(
        data_series=data_series,
        title="Grouped Analysis Results",
        xaxis_title="Feature X",
        yaxis_title="Feature Y",
        output_file=output_plot_2_path
    )

    print(f"Generated grouped plot: {output_plot_2_path}")

    builder = PlotlyPlotBuilder(title="Custom Builder Example")
    builder.add_scatter(
        x=x_data.tolist(),
        y=y_data.tolist(),
        name="Data Points",
        text=sample_names,
        color="#7b1fa2",
        marker_size=12
    )

    threshold_line_x = [x_data.min(), x_data.max()]
    threshold_line_y = [args.threshold, args.threshold]
    builder.add_line(
        x=threshold_line_x,
        y=threshold_line_y,
        name=f"Threshold ({args.threshold})",
        color="#f57c00",
        line_width=2
    )

    builder.set_axis_titles("Feature X", "Feature Y")
    builder.set_legend(x=1.02, y=1.0)
    builder.set_margins(t=50, r=150, b=50, l=50)

    output_plot_3_path = os.path.join(args.output, 'plot_3.json')
    builder.save(output_plot_3_path)

    print(f"Generated custom plot: {output_plot_3_path}")

    results_path = os.path.join(args.output, 'results.txt')
    with open(results_path, 'w') as f:
        f.write("Sample\tX\tY\tGroup\n")
        for i in range(n_points):
            group = 'Group A' if y_data[i] > 0 else 'Group B'
            f.write(f"{sample_names[i]}\t{x_data[i]:.4f}\t{y_data[i]:.4f}\t{group}\n")

    print(f"Analysis completed successfully!")
    print(f"Results saved to: {results_path}")
    print(f"Generated {3} plots")

    return 0


if __name__ == '__main__':
    sys.exit(main())
