"""
Example plugin demonstrating how to generate pre-generated JSON plots
that can be displayed by CauldronGO's PluginPlot component.

The JSON format follows the Plotly.js structure with data, layout, and config sections.
"""

import argparse
import json
import numpy as np
import pandas as pd


def create_scatter_plot_json(x_data, y_data, labels, output_path):
    """
    Create a Plotly JSON file for a scatter plot.

    Args:
        x_data: Array of x coordinates
        y_data: Array of y coordinates
        labels: Array of point labels
        output_path: Path to save the JSON file
    """
    plotly_json = {
        "data": [
            {
                "x": x_data.tolist(),
                "y": y_data.tolist(),
                "mode": "markers",
                "type": "scatter",
                "text": labels.tolist(),
                "hoverinfo": "text",
                "marker": {
                    "size": 10,
                    "color": "#1976d2",
                    "line": {
                        "color": "#fff",
                        "width": 1
                    }
                }
            }
        ],
        "layout": {
            "title": {
                "text": "Example Scatter Plot"
            },
            "xaxis": {
                "title": {
                    "text": "X Axis"
                },
                "zeroline": True,
                "gridcolor": "#e0e0e0",
                "showgrid": True
            },
            "yaxis": {
                "title": {
                    "text": "Y Axis"
                },
                "zeroline": True,
                "gridcolor": "#e0e0e0",
                "showgrid": True
            },
            "plot_bgcolor": "#fafafa",
            "paper_bgcolor": "#ffffff",
            "hovermode": "closest",
            "width": 800,
            "height": 600
        },
        "config": {
            "responsive": True,
            "displayModeBar": True,
            "displaylogo": False,
            "modeBarButtonsToRemove": ["toImage"]
        }
    }

    with open(output_path, 'w') as f:
        json.dump(plotly_json, f, indent=2)


def create_bar_chart_json(categories, values, output_path):
    """
    Create a Plotly JSON file for a bar chart.

    Args:
        categories: Array of category names
        values: Array of values for each category
        output_path: Path to save the JSON file
    """
    plotly_json = {
        "data": [
            {
                "x": categories.tolist(),
                "y": values.tolist(),
                "type": "bar",
                "marker": {
                    "color": "#388e3c"
                }
            }
        ],
        "layout": {
            "title": {
                "text": "Example Bar Chart"
            },
            "xaxis": {
                "title": {
                    "text": "Categories"
                }
            },
            "yaxis": {
                "title": {
                    "text": "Values"
                }
            },
            "plot_bgcolor": "#fafafa",
            "paper_bgcolor": "#ffffff"
        }
    }

    with open(output_path, 'w') as f:
        json.dump(plotly_json, f, indent=2)


def create_3d_scatter_json(x_data, y_data, z_data, labels, output_path):
    """
    Create a Plotly JSON file for a 3D scatter plot.

    Args:
        x_data: Array of x coordinates
        y_data: Array of y coordinates
        z_data: Array of z coordinates
        labels: Array of point labels
        output_path: Path to save the JSON file
    """
    plotly_json = {
        "data": [
            {
                "x": x_data.tolist(),
                "y": y_data.tolist(),
                "z": z_data.tolist(),
                "mode": "markers",
                "type": "scatter3d",
                "text": labels.tolist(),
                "hoverinfo": "text",
                "marker": {
                    "size": 5,
                    "color": "#d32f2f",
                    "line": {
                        "color": "#fff",
                        "width": 0.5
                    }
                }
            }
        ],
        "layout": {
            "title": {
                "text": "Example 3D Scatter Plot"
            },
            "scene": {
                "xaxis": {
                    "title": {
                        "text": "X Axis"
                    },
                    "gridcolor": "#e0e0e0",
                    "backgroundcolor": "#fafafa",
                    "showgrid": True
                },
                "yaxis": {
                    "title": {
                        "text": "Y Axis"
                    },
                    "gridcolor": "#e0e0e0",
                    "backgroundcolor": "#fafafa",
                    "showgrid": True
                },
                "zaxis": {
                    "title": {
                        "text": "Z Axis"
                    },
                    "gridcolor": "#e0e0e0",
                    "backgroundcolor": "#fafafa",
                    "showgrid": True
                }
            },
            "paper_bgcolor": "#ffffff",
            "hovermode": "closest",
            "width": 800,
            "height": 600
        }
    }

    with open(output_path, 'w') as f:
        json.dump(plotly_json, f, indent=2)


def main():
    parser = argparse.ArgumentParser(description='Example plugin with JSON plot generation')
    parser.add_argument('--input_file', required=True, help='Input data file')
    parser.add_argument('--output_folder', required=True, help='Output folder')

    args = parser.parse_args()

    df = pd.read_csv(args.input_file, sep='\t')

    x_data = np.random.randn(50)
    y_data = np.random.randn(50)
    z_data = np.random.randn(50)
    labels = [f'Point {i+1}' for i in range(50)]

    create_scatter_plot_json(
        x_data,
        y_data,
        np.array(labels),
        f'{args.output_folder}/plot_1.json'
    )

    categories = np.array(['A', 'B', 'C', 'D', 'E'])
    values = np.random.randint(10, 100, size=5)

    create_bar_chart_json(
        categories,
        values,
        f'{args.output_folder}/plot_2.json'
    )

    create_3d_scatter_json(
        x_data[:30],
        y_data[:30],
        z_data[:30],
        np.array(labels[:30]),
        f'{args.output_folder}/plot_3.json'
    )

    print("Generated 3 JSON plots: plot_1.json (scatter), plot_2.json (bar), plot_3.json (3D scatter)")


if __name__ == '__main__':
    main()
