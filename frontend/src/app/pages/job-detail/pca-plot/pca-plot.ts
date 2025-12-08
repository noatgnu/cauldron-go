import { Component, Input, OnInit, signal, NgZone } from '@angular/core';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { PlotlyModule } from 'angular-plotly.js';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { Wails } from '../../../core/services/wails';
import { Annotation, AnnotationService } from '../../../core/services/annotation.service';
import { PlotlyExport } from '../../../core/services/plotly-export';

interface PCAData {
  x: number[];
  y: number[];
  z?: number[];
  text: string[];
}

@Component({
  selector: 'app-pca-plot',
  imports: [
    PlotlyModule,
    MatProgressSpinnerModule,
    MatIconModule,
    ReactiveFormsModule,
    MatExpansionModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatCheckboxModule
  ],
  templateUrl: './pca-plot.html',
  styleUrl: './pca-plot.scss',
})
export class PcaPlot implements OnInit {
  @Input() jobId!: string;

  loading = signal(true);
  error = signal('');
  plotData: any[] = [];
  plotLayout: any = {};
  plotConfig: any = {
    responsive: true,
    displayModeBar: true,
    displaylogo: false,
    modeBarButtonsToRemove: ['toImage']
  };
  revision: number = 0;
  customizationForm: FormGroup;
  protected showCustomization = signal(false);

  private _pcaData: PCAData | null = null;
  private _explainedVariance: number[] | null = null;
  private _annotations: Annotation[] = [];
  
  constructor(
    private wails: Wails,
    private fb: FormBuilder,
    private annotationService: AnnotationService,
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
    await this.loadData();
  }

  private async loadData() {
    try {
      this.loading.set(true);
      await this.wails.logToFile(`[PCA Plot] Starting to load plot for job: ${this.jobId}`);

      const savedSettings = await this.loadSavedSettings();
      if (savedSettings) {
        const completeSettings = { ...this.customizationForm.value, ...savedSettings };
        this.customizationForm.patchValue(completeSettings, { emitEvent: false });
      }

      const [data, varianceData, annotations] = await Promise.all([
        this.wails.readJobOutputFile(this.jobId, 'pca_output.txt').catch(err => { throw new Error(`Failed to read pca_output.txt: ${err.message || String(err)}`); }),
        this.wails.readJobOutputFile(this.jobId, 'explained_variance_ratio.json').catch(err => { throw new Error(`Failed to read explained_variance_ratio.json: ${err.message || String(err)}`); }),
        this.annotationService.loadAnnotationsForJob(this.jobId)
      ]);

      if (!data) throw new Error('pca_output.txt file is empty');
      if (!varianceData) throw new Error('explained_variance_ratio.json file is empty');

      this._explainedVariance = JSON.parse(varianceData);
      this._pcaData = this.parsePCAData(data);
      this._annotations = annotations;

      this.updatePlot();
      await this.wails.logToFile('[PCA Plot] Plot data created successfully.');

    } catch (err: any) {
      const errorMsg = 'Failed to load PCA plot: ' + (err.message || String(err));
      this.error.set(errorMsg);
      await this.wails.logToFile('[PCA Plot] Error: ' + errorMsg);
    } finally {
      this.loading.set(false);
    }
  }

  private updatePlot() {
    if (!this._pcaData || !this._explainedVariance) {
      return;
    }

    this.ngZone.run(() => {
      const settings = this.customizationForm.value;
      const data: PCAData = this._pcaData!;
      const explainedVariance: number[] = this._explainedVariance!;

      const pc1Variance = (explainedVariance[0] * 100).toFixed(1);
      const pc2Variance = (explainedVariance[1] * 100).toFixed(1);
      const is3D = data.z && data.z.length > 0;
      const useConditions = this._annotations.length > 0;
      const markerSize = settings.markerSize || 10;
      const showGrid = settings.showGrid !== false;

      if (useConditions) {
        const dataWithAnnotations = data.text.map((sample, index) => {
          const annotation = this._annotations.find(a => a.sample === sample);
          return {
            x: data.x[index],
            y: data.y[index],
            z: is3D ? data.z![index] : undefined,
            text: sample,
            condition: annotation ? annotation.condition : 'Unknown',
            color: annotation ? annotation.color : '#808080' // Default to grey if no color
          };
        });

        const conditionMap = new Map<string, any[]>();
        dataWithAnnotations.forEach(annotatedPoint => {
          if (!conditionMap.has(annotatedPoint.condition)) {
            conditionMap.set(annotatedPoint.condition, []);
          }
          conditionMap.get(annotatedPoint.condition)!.push(annotatedPoint);
        });

        this.plotData = Array.from(conditionMap.entries()).map(([condition, points]) => {
          const trace: any = {
            x: points.map(p => p.x),
            y: points.map(p => p.y),
            mode: 'markers',
            type: is3D ? 'scatter3d' : 'scatter',
            name: condition,
            text: points.map(p => p.text),
            hoverinfo: 'text+name',
            marker: {
              size: is3D ? markerSize / 2 : markerSize,
              color: points.map(p => p.color), // Array of colors
              line: { color: '#fff', width: is3D ? 0.5 : 1 }
            }
          };
          if (is3D) {
            trace.z = points.map(p => p.z);
          }
          return trace;
        });

      } else {
          const trace: any = {
              x: data.x,
              y: data.y,
              mode: 'markers',
              type: is3D ? 'scatter3d' : 'scatter',
              text: data.text,
              hoverinfo: 'text',
              marker: {
                  size: is3D ? markerSize / 2 : markerSize,
                  color: '#1976d2', // Default single color
                  line: { color: '#fff', width: is3D ? 0.5 : 1 }
              }
          };
          if (is3D && data.z) {
              trace.z = data.z;
          }
          this.plotData = [trace];
      }

      let newLayout: any;
      if (is3D) {
        const pc3Variance = (explainedVariance[2] * 100).toFixed(1);
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : 'PCA Analysis Results (3D)' },
          width: settings.width || 800,
          height: settings.height || 600,
          scene: {
            xaxis: { title: { text: `PC1 (${pc1Variance}%)` }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            yaxis: { title: { text: `PC2 (${pc2Variance}%)` }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            zaxis: { title: { text: `PC3 (${pc3Variance}%)` }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid }
          },
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: useConditions,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop, r: settings.marginRight, b: settings.marginBottom, l: settings.marginLeft }
        };
      } else {
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : 'PCA Analysis Results' },
          width: settings.width || 800,
          height: settings.height || 600,
          xaxis: { title: { text: `PC1 (${pc1Variance}%)` }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          yaxis: { title: { text: `PC2 (${pc2Variance}%)` }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          plot_bgcolor: '#fafafa',
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: useConditions,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop, r: settings.marginRight, b: settings.marginBottom, l: settings.marginLeft }
        };
      }
      this.plotLayout = newLayout;
      this.revision++;
    });
  }

  private parsePCAData(tsvData: string): PCAData {
    const lines = tsvData.trim().split('\n');
    const headers = lines[0].split('\t');

    const xIndex = headers.indexOf('x_pca');
    const yIndex = headers.indexOf('y_pca');
    const zIndex = headers.indexOf('z_pca');
    const sampleIndex = headers.indexOf('sample');

    if (xIndex === -1 || yIndex === -1 || sampleIndex === -1) {
      throw new Error(`Missing required columns. Found: ${headers.join(', ')}`);
    }

    const x: number[] = [];
    const y: number[] = [];
    const z: number[] = [];
    const text: string[] = [];
    const has3D = zIndex !== -1;

    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split('\t');
      const maxIndex = has3D ? Math.max(xIndex, yIndex, zIndex, sampleIndex) : Math.max(xIndex, yIndex, sampleIndex);
      if (values.length > maxIndex) {
        x.push(parseFloat(values[xIndex]));
        y.push(parseFloat(values[yIndex]));
        if (has3D) {
          z.push(parseFloat(values[zIndex]));
        }
        const samplePath = values[sampleIndex];
        const sampleName = samplePath.split(/[/\\]/).pop() || samplePath;
        text.push(sampleName);
      }
    }

    return has3D ? { x, y, z, text } : { x, y, text };
  }

  private async loadSavedSettings(): Promise<any> {
    try {
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, '.plot-settings-pca.json');
      if (savedSettings) {
        return JSON.parse(savedSettings);
      }
    } catch (err: any) {
      await this.wails.logToFile(`[PCA Plot] No saved settings found or failed to load: ${err.message}`);
    }
    return null;
  }

  private async saveSettings() {
    try {
      const settings = this.customizationForm.value;
      await this.wails.writeJobOutputFile(this.jobId, '.plot-settings-pca.json', JSON.stringify(settings, null, 2));
      await this.wails.logToFile(`[PCA Plot] Settings saved for job: ${this.jobId}`);
    } catch (err: any) {
      await this.wails.logToFile(`[PCA Plot] Failed to save settings: ${err.message}`);
    }
  }

  async applyCustomization() {
    await this.saveSettings();
    this.updatePlot();
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('pca-plot-div', {
        filename: 'pca_plot',
        width: this.customizationForm.value.width || 1200,
        height: this.customizationForm.value.height || 800
      });
    } catch (error) {
      await this.wails.logToFile(`[PCA Plot] Failed to export to SVG: ${error}`);
    }
  }
}