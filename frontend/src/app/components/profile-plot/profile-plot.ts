import {Component, Input} from '@angular/core';
import {DataFrame, IDataFrame} from 'data-forge';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {PlotlyModule} from "angular-plotly.js";
import {MatButtonModule} from "@angular/material/button";

@Component({
  selector: 'app-profile-plot',
  imports: [PlotlyModule, ReactiveFormsModule, MatButtonModule],
  templateUrl: './profile-plot.html',
  styleUrl: './profile-plot.scss',
})
export class ProfilePlot {
  _annotation: IDataFrame<number, {Sample: string, Condition: string}> = new DataFrame();
  conditions: string[] = [];

  @Input() set annotation(value: IDataFrame<number, {Sample: string, Condition: string}>) {
    this._annotation = value;
    this.conditions = this.annotation.getSeries('Condition').distinct().toArray();
    const form: any = {};
    for (const col of this.annotation.getColumnNames()) {
      form[col] = new FormControl(col);
    }
    this.form = this.fb.group(form);
  }

  get annotation(): IDataFrame<number, {Sample: string, Condition: string}> {
    return this._annotation;
  }

  _data: IDataFrame<number, any> = new DataFrame();

  @Input() set data(value: IDataFrame<number, any>) {
    this._data = value;
    this.drawPlot();
  }

  get data(): IDataFrame<number, any> {
    return this._data;
  }

  form!: FormGroup;

  graphData: any[] = [];
  graphLayout: any = {
    title: 'Profile Plot',
    width: 1000,
    height: 500,
    xaxis: {
      title: 'Sample'
    },
    yaxis: {
      title: 'Log2 Intensity'
    }
  };
  revision = 0;

  constructor(private fb: FormBuilder) {
    this.form = this.fb.group({});
  }

  drawPlot() {
    const graphData = [];
    for (const s in this.form.value) {
      graphData.push({
        x: this.data.getSeries(s).toArray(),
        y: s,
        line: {
          color: 'black'
        },
        type: 'box',
        name: this.form.value[s],
        boxpoints: false,
        showlegend: false,
      });
    }
    this.graphData = graphData;
    this.revision++;
  }
}
