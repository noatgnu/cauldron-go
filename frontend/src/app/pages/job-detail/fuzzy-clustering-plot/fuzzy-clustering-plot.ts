import { Component, Input, OnInit, signal, NgZone } from '@angular/core';
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
  plotData: any[] = [];
  plotLayout: any = {};
  plotConfig: any = {
    responsive: true,
    displayModeBar: true,
    displaylogo: false,
    modeBarButtonsToRemove: ['toImage']
  };
  revision: number = 0;
  protected availableFiles = signal<string[]>([]);
  protected selectedFile = signal<string>('');
  protected customizationForm: FormGroup;
  protected showCustomization = signal(false);

  private _clusterDataCache = new Map<string, ClusterData>();
  private _explainedVariance: number[] | null = null;

  constructor(
    private wails: Wails,
    private fb: FormBuilder,
    private plotlyExport: PlotlyExport,
    private ngZone: NgZone
  ) {
    this.customizationForm = this.fb.group({
      title: [''],
      showGrid: [true],
      markerSize: [10],
      width: [800],
      height: [600],
      marginTop: [100],
      marginRight: [100],
      marginBottom: [100],
      marginLeft: [100]
    });
  }

  async ngOnInit() {
    try {
      this.loading.set(true);
      const savedSettings = await this.loadSavedSettings();
      if (savedSettings) {
        const completeSettings = { ...this.customizationForm.value, ...savedSettings };
        this.customizationForm.patchValue(completeSettings, { emitEvent: false });
      }
      await this.loadExplainedVariance();
      await this.findAndRenderFirstPlot();
    } catch (err: any) {
      this.error.set('Failed to initialize plot: ' + err.message);
      await this.wails.logToFile('[Fuzzy Clustering Plot] Error on init: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  private async findAndRenderFirstPlot() {
    const clusterFiles: string[] = [];
    for (let i = 2; i <= 10; i++) {
      try {
        const filename = `fcm_${i}_clusters.txt`;
        await this.wails.readJobOutputFile(this.jobId, filename); // Check existence
        clusterFiles.push(filename);
      } catch (e) {
        break; // Stop when a file is not found
      }
    }
    this.availableFiles.set(clusterFiles);

    if (clusterFiles.length > 0) {
      this.selectedFile.set(clusterFiles[0]);
      await this.updatePlot(clusterFiles[0]);
    } else {
      this.error.set('No clustering output files found');
    }
  }

  private async loadExplainedVariance() {
    if (this._explainedVariance) return;
    try {
      const varianceData = await this.wails.readJobOutputFile(this.jobId, 'explained_variance_ratio.json');
      if (!varianceData) throw new Error('explained_variance_ratio.json file is empty or not found');
      this._explainedVariance = JSON.parse(varianceData);
    } catch (err) {
      throw new Error('Failed to load explained variance data');
    }
  }

  async onFileChange(fileName: string) {
    this.selectedFile.set(fileName);
    await this.updatePlot(fileName);
  }

  private async getOrLoadClusterData(fileName: string): Promise<ClusterData> {
    if (this._clusterDataCache.has(fileName)) {
      return this._clusterDataCache.get(fileName)!;
    }
    const data = await this.wails.readJobOutputFile(this.jobId, fileName);
    if (!data) throw new Error(`${fileName} file is empty or not found`);
    const parsedData = this.parseClusterData(data);
    this._clusterDataCache.set(fileName, parsedData);
    return parsedData;
  }

  async updatePlot(fileName: string) {
    if (!fileName) return;
    this.loading.set(true);
    try {
      const clusterData = await this.getOrLoadClusterData(fileName);
      if (!this._explainedVariance) await this.loadExplainedVariance();

      this.ngZone.run(() => {
        const settings = this.customizationForm.value;
        const pc1Variance = (this._explainedVariance![0] * 100).toFixed(1);
        const pc2Variance = (this._explainedVariance![1] * 100).toFixed(1);

        const uniqueClusters = [...new Set(clusterData.cluster)];
        const colors = ['#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2', '#0097a7', '#c2185b', '#5d4037'];
        const markerSize = settings.markerSize || 10;
        const showGrid = settings.showGrid !== false;

        this.plotData = uniqueClusters.map((clusterName, index) => {
          const indices = clusterData.cluster.map((c, i) => c === clusterName ? i : -1).filter(i => i >= 0);
          return {
            x: indices.map(i => clusterData.x[i]),
            y: indices.map(i => clusterData.y[i]),
            mode: 'markers', type: 'scatter', name: `Cluster ${clusterName}`,
            text: indices.map(i => clusterData.text[i]),
            marker: { size: markerSize, color: colors[index % colors.length], line: { color: '#fff', width: 1 } }
          };
        });

        this.plotLayout = {
          title: { text: settings.title !== '' ? settings.title : `Fuzzy Clustering Results (${fileName})` },
          width: settings.width || 800, height: settings.height || 600,
          xaxis: { title: { text: `PC1 (${pc1Variance}%)` }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          yaxis: { title: { text: `PC2 (${pc2Variance}%)` }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          plot_bgcolor: '#fafafa', paper_bgcolor: '#ffffff',
          hovermode: 'closest', showlegend: true, legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop, r: settings.marginRight, b: settings.marginBottom, l: settings.marginLeft }
        };

        this.revision++;
      });
    } catch (err: any) {
      this.error.set(`Failed to render plot for ${fileName}: ${err.message}`);
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Error rendering plot: ${err.message}`);
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
      throw new Error(`Missing required columns in cluster data. Found: ${headers.join(', ')}`);
    }

    const x: number[] = [], y: number[] = [], text: string[] = [], cluster: string[] = [];
    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split('\t');
      if (values.length > Math.max(xIndex, yIndex, sampleIndex, clusterIndex)) {
        x.push(parseFloat(values[xIndex]));
        y.push(parseFloat(values[yIndex]));
        const samplePath = values[sampleIndex];
        text.push(samplePath.split(/[/\\]/).pop() || samplePath);
        cluster.push(values[clusterIndex]);
      }
    }
    return { x, y, text, cluster };
  }

  private async loadSavedSettings(): Promise<any> {
    try {
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, '.plot-settings-fuzzy-clustering.json');
      return savedSettings ? JSON.parse(savedSettings) : null;
    } catch (err) {
      return null;
    }
  }

  private async saveSettings() {
    try {
      const settings = this.customizationForm.value;
      await this.wails.writeJobOutputFile(this.jobId, '.plot-settings-fuzzy-clustering.json', JSON.stringify(settings, null, 2));
    } catch (err: any) {
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Failed to save settings: ${err.message}`);
    }
  }

  async applyCustomization() {
    await this.saveSettings();
    await this.updatePlot(this.selectedFile());
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('fuzzy-clustering-plot-div', {
        filename: `fuzzy_clustering_${this.selectedFile()}`,
        width: this.customizationForm.value.width || 1200,
        height: this.customizationForm.value.height || 800
      });
    } catch (error) {
      await this.wails.logToFile(`[Fuzzy Clustering Plot] Failed to export to SVG: ${error}`);
    }
  }
}
