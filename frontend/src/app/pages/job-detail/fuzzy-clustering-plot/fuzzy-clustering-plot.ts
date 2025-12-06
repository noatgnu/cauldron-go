import { Component, Input, OnInit, signal } from '@angular/core';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { PlotlyModule } from 'angular-plotly.js';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { Wails } from '../../../core/services/wails';
import { PlotlyExport } from '../../../core/services/plotly-export';

interface ClusterData {
  x: number[];
  y: number[];
  text: string[];
  cluster: string[];
}

@Component({
  selector: 'app-fuzzy-clustering-plot',
  imports: [
    PlotlyModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatSelectModule,
    MatFormFieldModule,
    ReactiveFormsModule,
    MatExpansionModule,
    MatInputModule,
    MatButtonModule,
    MatCheckboxModule
  ],
  templateUrl: './fuzzy-clustering-plot.html',
  styleUrl: './fuzzy-clustering-plot.scss',
})
export class FuzzyClusteringPlot implements OnInit {
  @Input() jobId!: string;

  protected loading = signal(true);
  protected error = signal('');
  protected plotData = signal<any[]>([]);
  protected plotLayout = signal<any>({});
  protected plotConfig = signal<any>({
    responsive: true,
    displayModeBar: true,
    displaylogo: false
  });
  protected availableFiles = signal<string[]>([]);
  protected selectedFile = signal<string>('');
  protected customizationForm: FormGroup;
  protected showCustomization = signal(false);

  constructor(
    private wails: Wails,
    private fb: FormBuilder,
    private plotlyExport: PlotlyExport
  ) {
    this.customizationForm = this.fb.group({
      title: [''],
      showGrid: [true],
      markerSize: [10],
      width: [800],
      height: [600]
    });
  }

  async ngOnInit() {
    await this.findClusterFiles();
  }

  async findClusterFiles() {
    try {
      this.loading.set(true);
      const clusterFiles: string[] = [];

      for (let i = 2; i <= 10; i++) {
        try {
          const filename = `fcm_${i}_clusters.txt`;
          await this.wails.readJobOutputFile(this.jobId, filename);
          clusterFiles.push(filename);
        } catch (e) {
          break;
        }
      }

      this.availableFiles.set(clusterFiles);

      if (clusterFiles.length > 0) {
        this.selectedFile.set(clusterFiles[0]);
        await this.loadAndRenderPlot(clusterFiles[0]);
        this.loadSavedSettings();
      } else {
        this.error.set('No clustering output files found');
        this.loading.set(false);
      }
    } catch (err: any) {
      this.error.set('Failed to find clustering files: ' + err.message);
      await this.wails.logToFile('[Fuzzy Clustering Plot] Error finding files: ' + err.message);
      this.loading.set(false);
    }
  }

  async onFileChange(fileName: string) {
    this.selectedFile.set(fileName);
    await this.loadAndRenderPlot(fileName);
  }

  async loadAndRenderPlot(fileName: string) {
    try {
      this.loading.set(true);

      const [data, varianceData] = await Promise.all([
        this.wails.readJobOutputFile(this.jobId, fileName),
        this.wails.readJobOutputFile(this.jobId, 'explained_variance_ratio.json')
      ]);

      if (!data) {
        throw new Error(`${fileName} file is empty or not found`);
      }

      if (!varianceData) {
        throw new Error('explained_variance_ratio.json file is empty or not found');
      }

      const explainedVariance = JSON.parse(varianceData);
      const clusterData = this.parseClusterData(data);
      this.createPlotData(clusterData, explainedVariance);
    } catch (err: any) {
      this.error.set('Failed to load clustering plot: ' + err.message);
      await this.wails.logToFile('[Fuzzy Clustering Plot] Error: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  private parseClusterData(tsvData: string): ClusterData {
    const lines = tsvData.trim().split('\n');
    const headers = lines[0].split('\t');

    const xIndex = headers.indexOf('x');
    const yIndex = headers.indexOf('y');
    const sampleIndex = headers.indexOf('Sample');
    const clusterIndex = headers.indexOf('cluster');

    if (xIndex === -1 || yIndex === -1 || sampleIndex === -1 || clusterIndex === -1) {
      throw new Error(`Missing required columns. Found: ${headers.join(', ')}`);
    }

    const x: number[] = [];
    const y: number[] = [];
    const text: string[] = [];
    const cluster: string[] = [];

    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split('\t');
      if (values.length > Math.max(xIndex, yIndex, sampleIndex, clusterIndex)) {
        x.push(parseFloat(values[xIndex]));
        y.push(parseFloat(values[yIndex]));
        const samplePath = values[sampleIndex];
        const sampleName = samplePath.split(/[/\\]/).pop() || samplePath;
        text.push(sampleName);
        cluster.push(values[clusterIndex]);
      }
    }

    return { x, y, text, cluster };
  }

  private createPlotData(data: ClusterData, explainedVariance: number[]) {
    const pc1Variance = (explainedVariance[0] * 100).toFixed(1);
    const pc2Variance = (explainedVariance[1] * 100).toFixed(1);

    const uniqueClusters = [...new Set(data.cluster)];
    const colors = ['#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2', '#0097a7', '#c2185b', '#5d4037'];

    const traces = uniqueClusters.map((clusterName, index) => {
      const indices = data.cluster.map((c, i) => c === clusterName ? i : -1).filter(i => i >= 0);
      return {
        x: indices.map(i => data.x[i]),
        y: indices.map(i => data.y[i]),
        mode: 'markers',
        type: 'scatter',
        name: `Cluster ${clusterName}`,
        text: indices.map(i => data.text[i]),
        marker: {
          size: 10,
          color: colors[index % colors.length],
          line: {
            color: '#fff',
            width: 1
          }
        }
      };
    });

    this.plotData.set(traces);

    this.plotLayout.set({
      title: 'Fuzzy Clustering Results',
      xaxis: {
        title: `PC1 (${pc1Variance}%)`,
        zeroline: true,
        gridcolor: '#e0e0e0'
      },
      yaxis: {
        title: `PC2 (${pc2Variance}%)`,
        zeroline: true,
        gridcolor: '#e0e0e0'
      },
      plot_bgcolor: '#fafafa',
      paper_bgcolor: '#ffffff',
      hovermode: 'closest',
      showlegend: true,
      legend: {
        x: 1.02,
        y: 1
      },
      margin: { t: 50, r: 150, b: 50, l: 50 }
    });
  }

  private async loadSavedSettings() {
    try {
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, '.plot-settings-fuzzy-clustering.json');
      if (savedSettings) {
        const settings = JSON.parse(savedSettings);
        this.customizationForm.patchValue(settings);
        this.applyCustomization();
      }
    } catch (err: any) {
      await this.wails.logToFile(`[Fuzzy Clustering Plot] No saved settings found or failed to load: ${err.message}`);
    }
  }

  private async saveSettings() {
    try {
      const settings = this.customizationForm.value;
      await this.wails.writeJobOutputFile(this.jobId, '.plot-settings-fuzzy-clustering.json', JSON.stringify(settings, null, 2));
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Settings saved for job: ${this.jobId}`);
    } catch (err: any) {
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Failed to save settings: ${err.message}`);
    }
  }

  async applyCustomization() {
    const values = this.customizationForm.value;

    this.plotLayout.update(layout => ({
      ...layout,
      title: values.title !== undefined && values.title !== null ? values.title : layout.title,
      width: values.width,
      height: values.height,
      xaxis: {
        ...layout.xaxis,
        showgrid: values.showGrid
      },
      yaxis: {
        ...layout.yaxis,
        showgrid: values.showGrid
      }
    }));

    this.plotData.update(data =>
      data.map(trace => ({
        ...trace,
        marker: {
          ...trace.marker,
          size: values.markerSize
        }
      }))
    );

    await this.saveSettings();
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('fuzzy-clustering-plot-div', {
        filename: 'fuzzy_clustering_plot',
        width: this.customizationForm.value.width || 1200,
        height: this.customizationForm.value.height || 800
      });
    } catch (error) {
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Failed to export to SVG: ${error}`);
    }
  }
}
