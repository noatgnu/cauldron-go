import {Component, Input} from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import {DataFrame, IDataFrame} from "data-forge";
import {VolcanoDataRow} from "./volcano-data-row";

@Component({
  selector: 'app-volcano-plot',
  imports: [PlotlyModule],
  templateUrl: './volcano-plot.html',
  styleUrl: './volcano-plot.scss',
})
export class VolcanoPlot {
  _data: IDataFrame<number, VolcanoDataRow> = new DataFrame();

  @Input() selection: {[key: string]: {selectionLabels: string[], color: string}} = {};

  cutoffGroupMap: {[key: string]: string} = {};
  _title: string = "Volcano Plot";

  @Input() set title(value: string) {
    if (value) {
      this._title = value;
    }
  }

  get title(): string {
    return this._title;
  }

  @Input() set data(value: IDataFrame<number, VolcanoDataRow>) {
    this._data = value;
    if (this._data.count() > 0) {
      this.maxLog2FC = this._data.getSeries("x").max();
      this.minLog2FC = this._data.getSeries("x").min();
      this.maxPValue = this._data.getSeries("y").max();
      this.minPValue = this._data.getSeries("y").min();
      this._data.forEach((row: any) => {
        let group = "";
        if (Math.abs(row.x) >= this.log2FoldChangeCutoff) {
          group += "Log2FC >= " + this.log2FoldChangeCutoff;
        } else {
          group += "Log2FC < " + this.log2FoldChangeCutoff;
        }

        if (row.y >= this.pValueCutoff) {
          group += "; -Log10(p-value) < " + this.pValueCutoff.toFixed(2);
        } else {
          group += "; -Log10(p-value) >= " + this.pValueCutoff.toFixed(2);
        }
        this.cutoffGroupMap[row.index] = group;
      });
    }
  }

  get data(): IDataFrame<number, VolcanoDataRow> {
    return this._data;
  }

  _revision: number = 0;

  maxLog2FC: number = 0;
  minLog2FC: number = 0;
  maxPValue: number = 0;
  minPValue: number = 0;

  @Input() set revision(value: number) {
    this.draw();
  }

  get revision(): number {
    return this._revision;
  }

  @Input() pValueCutoff: number = -Math.log10(0.05);
  @Input() log2FoldChangeCutoff: number = 0.6;

  @Input() backend: "scatter" | "scattergl" = "scatter";

  graphData: any = [];
  graphLayout: any = {
    height: 700,
    width: 700,
    xaxis: {
      title: "<b>Log2FC</b>",
      tickmode: "linear",
      ticklen: 5,
      showgrid: false,
      visible: true,
    },
    yaxis: {
      title: "<b>-log10(p-value)</b>",
      tickmode: "linear",
      ticklen: 5,
      showgrid: false,
      visible: true,
      showticklabels: true,
      zeroline: true,
    },
    annotations: [],
    showlegend: true,
    legend: {
      orientation: 'h'
    },
    title: {
      text: this.title,
      font: {
        size: 24,
        family: "Arial, sans-serif"
      },
    },
    shapes: [],
  };

  draw() {
    this.graphLayout.title = this.title;
    let shapes: any[] = [];
    const graphData: any = {};

    this.data.forEach((row: any) => {
      if (row.index in this.cutoffGroupMap) {
        if (!graphData[this.cutoffGroupMap[row.index]]) {
          graphData[this.cutoffGroupMap[row.index]] = {
            x: [],
            y: [],
            text: [],
            mode: 'markers',
            type: this.backend,
            name: this.cutoffGroupMap[row.index],
          };
        }
        graphData[this.cutoffGroupMap[row.index]].x.push(row.x);
        graphData[this.cutoffGroupMap[row.index]].y.push(row.y);
        graphData[this.cutoffGroupMap[row.index]].text.push(row.label);
      }
      if (row.index in this.selection) {
        this.selection[row.index].selectionLabels.forEach((label) => {
          if (!graphData[label]) {
            graphData[label] = {
              x: [],
              y: [],
              text: [],
              mode: 'markers',
              type: this.backend,
              name: label,
              marker: {
                color: this.selection[row.index].color
              }
            };
          }
          graphData[label].x.push(row.x);
          graphData[label].y.push(row.y);
          graphData[label].text.push(row.label);
        });
      }
    });

    shapes.push({
      type: 'line',
      x0: this.log2FoldChangeCutoff,
      y0: 0,
      x1: this.log2FoldChangeCutoff,
      y1: this.maxPValue + 1,
      line: {
        width: 1,
        dash: 'dot'
      }
    });
    shapes.push({
      type: 'line',
      x0: -this.log2FoldChangeCutoff,
      y0: 0,
      x1: -this.log2FoldChangeCutoff,
      y1: this.maxPValue + 1,
      line: {
        width: 1,
        dash: 'dot'
      }
    });
    shapes.push({
      type: 'line',
      x0: this.minLog2FC-1,
      y0: this.pValueCutoff,
      x1: this.maxLog2FC+1,
      y1: this.pValueCutoff,
      line: {
        width: 1,
        dash: 'dot'
      }
    });

    this.graphLayout.shapes = shapes;
    this.graphLayout.xaxis.range = [this.minLog2FC - 1, this.maxLog2FC + 1];
    this.graphLayout.yaxis.range = [0, this.maxPValue + 1];

    this.graphData = Object.values(graphData);
    this._revision++;
  }
}
