import {Component, Input} from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import {DataFrame, IDataFrame} from "data-forge";

@Component({
  selector: 'app-fuzzy-clustering-pca-plot',
  imports: [PlotlyModule],
  templateUrl: './fuzzy-clustering-pca-plot.html',
  styleUrl: './fuzzy-clustering-pca-plot.scss',
})
export class FuzzyClusteringPcaPlot {
  @Input() data: IDataFrame<number, {x: number, y: number, Sample: string, Condition: string, cluster: number}> = new DataFrame();

  _revision: number = 0;

  @Input() set revision(value: number) {
    this.updatePlot();
    this._revision = value;
  }

  get revision(): number {
    return this._revision;
  }

  graphData: any[] = [];
  graphLayout: any = {
    title: 'Fuzzy Clustering PCA',
    xaxis: {
      title: 'PC1',
      type: 'linear',
    },
    yaxis: {
      title: 'PC2',
      type: 'linear',
    }
  };

  updatePlot() {
    if (this.data.count() > 0) {
      const graphData: any[] = [];
      this.data.groupBy((row: any) => row.cluster).forEach((group: any) => {
        const data: any = {
          x: [],
          y: [],
          text: [],
          mode: 'markers',
          type: 'scatter',
          name: `Cluster ${group.first().cluster}`
        };
        group.forEach((row: any) => {
          data.x.push(row.x);
          data.y.push(row.y);
          data.text.push(`Sample:${row.Sample}<br>Condition:(${row.Condition})<br>Cluster:${row.cluster}`);
        });
        graphData.push(data);
      });
      this.graphData = graphData;
    }
  }
}
