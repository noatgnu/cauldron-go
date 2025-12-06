import {AfterViewInit, Component, Input} from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import {DataFrame, IDataFrame} from "data-forge";
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {MatButtonModule} from "@angular/material/button";

@Component({
  selector: 'app-pca-plot',
  imports: [PlotlyModule, MatButtonModule, ReactiveFormsModule],
  templateUrl: './pca-plot.html',
  styleUrl: './pca-plot.scss',
})
export class PcaPlot implements AfterViewInit {
  _data: IDataFrame<number, {x_pca: number, y_pca: number, z_pca?: number, sample: string}> = new DataFrame();
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

  _explainedVariance: number[] = [0,0,0];

  @Input() set explainedVariance(value: number[]) {
    for (let i = 0; i < value.length; i++) {
      this._explainedVariance[i] = value[i];
    }
    if (this._explainedVariance[0]>0) {
      this.graphLayout.xaxis.title = `PCA 1 (${Math.round(this._explainedVariance[0]*100)}%)`;
    }
    if (this._explainedVariance[1]>0) {
      this.graphLayout.yaxis.title = `PCA 2 (${Math.round(this._explainedVariance[1]*100)}%)`;
    }
    if (this._explainedVariance[2]>0) {
      this.graphLayout.scene = {
        xaxis: {
          title: `PCA 1 (${Math.round(this._explainedVariance[0]*100)}%)`
        },
        yaxis: {
          title: `PCA 2 (${Math.round(this._explainedVariance[1]*100)}%)`
        },
        zaxis: {
          title: `PCA 3 (${Math.round(this._explainedVariance[2]*100)}%)`
        }
      };
    }
  }

  get explainedVariance(): number[] {
    return this._explainedVariance;
  }

  @Input() set data(value: IDataFrame<number, {x_pca: number, y_pca: number, z_pca?: number, sample: string}>) {
    this._data = value;
    if (this._data.count() > 0) {
      this.drawPlot();
    }
  }

  get data(): IDataFrame<number, {x_pca: number, y_pca: number, z_pca?: number, sample: string}> {
    return this._data;
  }

  graphData: any[] = [];
  graphLayout: any = {
    width: 600,
    height: 600,
    margin: {
      l: 40,
      r: 40,
      b: 40,
      t: 40,
    },
    xaxis: {
      title: 'PCA 1'
    },
    yaxis: {
      title: 'PCA 2'
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

  ngAfterViewInit() {
  }

  drawPlot() {
    const first = this._data.first();
    let graphData: any[] = [];
    if (this.annotation.count() === 0) {
      const x = this._data.getSeries('x_pca').toArray();
      const y = this._data.getSeries('y_pca').toArray();
      if (first.z_pca) {
        const z = this._data.getSeries('z_pca').toArray();
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
          x_pca: l.x_pca,
          y_pca: l.y_pca,
          z_pca: l.z_pca,
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
        const x = g.getSeries('x_pca').toArray();
        const y = g.getSeries('y_pca').toArray();
        if (first.z_pca) {
          const z = g.getSeries('z_pca').toArray();
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
