import json
import os
from typing import List, Dict, Any, Optional, Union


class PlotlyPlotBuilder:
    def __init__(self, title: str = ""):
        self.data: List[Dict[str, Any]] = []
        self.layout: Dict[str, Any] = {
            "title": title,
            "paper_bgcolor": "#ffffff",
            "hovermode": "closest"
        }
        self.config: Dict[str, Any] = {
            "responsive": True,
            "displayModeBar": True,
            "displaylogo": False
        }

    def add_scatter(
        self,
        x: List[Union[float, int]],
        y: List[Union[float, int]],
        name: Optional[str] = None,
        text: Optional[List[str]] = None,
        color: str = "#1976d2",
        marker_size: int = 10,
        show_legend: bool = True
    ):
        trace = {
            "x": x,
            "y": y,
            "mode": "markers",
            "type": "scatter",
            "marker": {
                "size": marker_size,
                "color": color,
                "line": {
                    "color": "#fff",
                    "width": 1
                }
            }
        }

        if name:
            trace["name"] = name

        if text:
            trace["text"] = text
            trace["hoverinfo"] = "text+name" if name else "text"
        else:
            trace["hoverinfo"] = "x+y+name" if name else "x+y"

        if not show_legend:
            trace["showlegend"] = False

        self.data.append(trace)
        return self

    def add_scatter_3d(
        self,
        x: List[Union[float, int]],
        y: List[Union[float, int]],
        z: List[Union[float, int]],
        name: Optional[str] = None,
        text: Optional[List[str]] = None,
        color: str = "#1976d2",
        marker_size: int = 6,
        show_legend: bool = True
    ):
        trace = {
            "x": x,
            "y": y,
            "z": z,
            "mode": "markers",
            "type": "scatter3d",
            "marker": {
                "size": marker_size,
                "color": color,
                "line": {
                    "color": "#fff",
                    "width": 0.5
                }
            }
        }

        if name:
            trace["name"] = name

        if text:
            trace["text"] = text
            trace["hoverinfo"] = "text+name" if name else "text"
        else:
            trace["hoverinfo"] = "x+y+z+name" if name else "x+y+z"

        if not show_legend:
            trace["showlegend"] = False

        self.data.append(trace)
        return self

    def add_line(
        self,
        x: List[Union[float, int]],
        y: List[Union[float, int]],
        name: Optional[str] = None,
        color: str = "#1976d2",
        line_width: int = 2,
        show_legend: bool = True
    ):
        trace = {
            "x": x,
            "y": y,
            "mode": "lines",
            "type": "scatter",
            "line": {
                "color": color,
                "width": line_width
            }
        }

        if name:
            trace["name"] = name

        if not show_legend:
            trace["showlegend"] = False

        self.data.append(trace)
        return self

    def add_bar(
        self,
        x: List[Union[str, float, int]],
        y: List[Union[float, int]],
        name: Optional[str] = None,
        color: str = "#1976d2",
        show_legend: bool = True
    ):
        trace = {
            "x": x,
            "y": y,
            "type": "bar",
            "marker": {
                "color": color
            }
        }

        if name:
            trace["name"] = name

        if not show_legend:
            trace["showlegend"] = False

        self.data.append(trace)
        return self

    def set_layout(self, **kwargs):
        self.layout.update(kwargs)
        return self

    def set_axis_titles(self, xaxis_title: str, yaxis_title: str, zaxis_title: Optional[str] = None):
        self.layout["xaxis"] = self.layout.get("xaxis", {})
        self.layout["xaxis"]["title"] = xaxis_title

        self.layout["yaxis"] = self.layout.get("yaxis", {})
        self.layout["yaxis"]["title"] = yaxis_title

        if zaxis_title:
            if "scene" not in self.layout:
                self.layout["scene"] = {}
            self.layout["scene"]["zaxis"] = {"title": zaxis_title}

        return self

    def set_3d_axis_titles(self, xaxis_title: str, yaxis_title: str, zaxis_title: str):
        self.layout["scene"] = {
            "xaxis": {"title": xaxis_title, "gridcolor": "#e0e0e0", "backgroundcolor": "#fafafa"},
            "yaxis": {"title": yaxis_title, "gridcolor": "#e0e0e0", "backgroundcolor": "#fafafa"},
            "zaxis": {"title": zaxis_title, "gridcolor": "#e0e0e0", "backgroundcolor": "#fafafa"}
        }
        return self

    def set_legend(self, x: float = 1.02, y: float = 1.0, show: bool = True):
        self.layout["showlegend"] = show
        if show:
            self.layout["legend"] = {"x": x, "y": y}
        return self

    def set_margins(self, t: int = 50, r: int = 50, b: int = 50, l: int = 50):
        self.layout["margin"] = {"t": t, "r": r, "b": b, "l": l}
        return self

    def build(self) -> Dict[str, Any]:
        return {
            "data": self.data,
            "layout": self.layout,
            "config": self.config
        }

    def save(self, output_file: str):
        plot_json = self.build()
        with open(output_file, 'w') as f:
            json.dump(plot_json, f, indent=2)


def create_scatter_plot(
    x: List[Union[float, int]],
    y: List[Union[float, int]],
    title: str = "Scatter Plot",
    xaxis_title: str = "X",
    yaxis_title: str = "Y",
    text: Optional[List[str]] = None,
    color: str = "#1976d2",
    output_file: Optional[str] = None
) -> Dict[str, Any]:
    builder = PlotlyPlotBuilder(title=title)
    builder.add_scatter(x, y, text=text, color=color)
    builder.set_axis_titles(xaxis_title, yaxis_title)

    if output_file:
        builder.save(output_file)

    return builder.build()


def create_scatter_3d_plot(
    x: List[Union[float, int]],
    y: List[Union[float, int]],
    z: List[Union[float, int]],
    title: str = "3D Scatter Plot",
    xaxis_title: str = "X",
    yaxis_title: str = "Y",
    zaxis_title: str = "Z",
    text: Optional[List[str]] = None,
    color: str = "#1976d2",
    output_file: Optional[str] = None
) -> Dict[str, Any]:
    builder = PlotlyPlotBuilder(title=title)
    builder.add_scatter_3d(x, y, z, text=text, color=color)
    builder.set_3d_axis_titles(xaxis_title, yaxis_title, zaxis_title)

    if output_file:
        builder.save(output_file)

    return builder.build()


def create_multi_scatter_plot(
    data_series: List[Dict[str, Any]],
    title: str = "Multi-Series Scatter Plot",
    xaxis_title: str = "X",
    yaxis_title: str = "Y",
    output_file: Optional[str] = None
) -> Dict[str, Any]:
    builder = PlotlyPlotBuilder(title=title)

    default_colors = ['#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2',
                      '#0097a7', '#c2185b', '#5d4037', '#fbc02d', '#00796b']

    for i, series in enumerate(data_series):
        color = series.get('color', default_colors[i % len(default_colors)])
        builder.add_scatter(
            x=series['x'],
            y=series['y'],
            name=series.get('name'),
            text=series.get('text'),
            color=color,
            marker_size=series.get('marker_size', 10)
        )

    builder.set_axis_titles(xaxis_title, yaxis_title)
    builder.set_legend(x=1.02, y=1.0)
    builder.set_margins(t=50, r=150, b=50, l=50)

    if output_file:
        builder.save(output_file)

    return builder.build()


def create_bar_plot(
    x: List[Union[str, float, int]],
    y: List[Union[float, int]],
    title: str = "Bar Plot",
    xaxis_title: str = "Category",
    yaxis_title: str = "Value",
    color: str = "#1976d2",
    output_file: Optional[str] = None
) -> Dict[str, Any]:
    builder = PlotlyPlotBuilder(title=title)
    builder.add_bar(x, y, color=color)
    builder.set_axis_titles(xaxis_title, yaxis_title)

    if output_file:
        builder.save(output_file)

    return builder.build()
