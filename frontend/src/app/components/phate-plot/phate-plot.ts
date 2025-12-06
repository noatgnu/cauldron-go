import {Component, Input} from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import {DataFrame, IDataFrame} from "data-forge";
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {MatButtonModule} from "@angular/material/button";

@Component({
  selector: 'app-phate-plot',
  imports: [PlotlyModule, ReactiveFormsModule, MatButtonModule],
  templateUrl: './phate-plot.html',
  styleUrl: './phate-plot.scss',
})
export class PhatePlot {
  _data: IDataFrame<number, {x_phate: number, y_phate: number, z_phate?: number, sample: string}> = new DataFrame();
  _annotation: IDataFrame<number, {Sample: string, Condition: string}> = new DataFrame();
  conditions: string[] = [];
  _jobId: string = '';

  @Input() set jobId(value: string) {
    this._jobId = value;
  }

  get jobId(): string {
    return this._jobId;
  }

  @Input() set annotation(value: IDataFrame<number, {Sample: string, Condition: string}>) {
    this._annotation = value;
    this.conditions = this._annotation.getSeries('Condition').distinct().toArray();
    const form: any = {};
    if (this.conditions.length > 0) {
      for (const c of this.conditions) {
        form[c] = new FormControl<string>('');
      }
    }
    this.form = this.fb.group(form);
  }

  get annotation(): IDataFrame<number, {Sample: string, Condition: string}> {
    return this._annotation;
  }

  @Input() set data(value: IDataFrame<number, {x_phate: number, y_phate: number, z_phate?: number, sample: string}>) {
    this._data = value;
    if (this._data.count() > 0) {
      this.drawPlot();
    }
  }

  get data(): IDataFrame<number, {x_phate: number, y_phate: number, z_phate?: number, sample: string}> {
    return this._data;
  }

  graphData: any[] = [];
  graphLayout: any = {
    scene: {
      xaxis: {
        title: 'Phate 1'
      },
      yaxis: {
        title: 'Phate 2'
      },
      zaxis: {
        title: 'Phate 3'
      }
    },
    width: 600,
    height: 600,
    margin: {
      l: 40,
      r: 40,
      b: 40,
      t: 40,
    },
    xaxis: {
      title: 'Phate 1'
    },
    yaxis: {
      title: 'Phate 2'
    },
    legend: {
      orientation: 'h'
    }
  };

  revision: number = 0;

  form!: FormGroup;

  defaultColorList: string[] = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
  ];

  constructor(private fb: FormBuilder) {
    this.form = this.fb.group({});
  }

  drawPlot() {
    const first = this._data.first();
    let graphData: any[] = [];
    if (this.annotation.count() === 0) {
      const x = this._data.getSeries('x_phate').toArray();
      const y = this._data.getSeries('y_phate').toArray();
      if (first.z_phate) {
        const z = this._data.getSeries('z_phate').toArray();
        graphData = [
          {
            x: x,
            y: y,
            z: z,
            mode: 'markers',
            type: 'scatter3d',
            text: this._data.getSeries('sample').toArray(),
            marker: { size: 12 }
          }
        ];
      } else {
        graphData = [
          {
            x: x,
            y: y,
            mode: 'markers',
            type: 'scatter',
            text: this._data.getSeries('sample').toArray(),
            marker: { size: 12 }
          }
        ];
      }
    } else {
      const joined = this._data.join(this._annotation, (row: any) => row.sample, (row: any) => row.Sample, (l: any, r: any) => {
        return {
          x_phate: l.x_phate,
          y_phate: l.y_phate,
          z_phate: l.z_phate,
          sample: l.sample,
          condition: r.Condition
        };
      });

      let position = 0;
      for (const g of joined.groupBy((row: any) => row.condition)) {
        let color = '';
        if (this.form.controls[g.first().condition].value === '') {
          color = this.defaultColorList[position%this.defaultColorList.length];
          this.form.controls[g.first().condition].setValue(color);
        } else {
          color = this.form.controls[g.first().condition].value;
        }
        const x = g.getSeries('x_phate').toArray();
        const y = g.getSeries('y_phate').toArray();
        if (first.z_phate) {
          const z = g.getSeries('z_phate').toArray();
          graphData.push({
            x: x,
            y: y,
            z: z,
            mode: 'markers',
            type: 'scatter3d',
            text: g.getSeries('sample').toArray(),
            name: g.first().condition,
            marker: { size: 12, color: color }
          });
        } else {
          graphData.push({
            x: x,
            y: y,
            mode: 'markers',
            type: 'scatter',
            text: g.getSeries('sample').toArray(),
            name: g.first().condition,
            marker: { size: 12, color: color }
          });
        }
        position ++;
      }
    }
    this.graphData = graphData;
    this.revision += 1;
  }
}
