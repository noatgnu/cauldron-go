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

interface PHATEData {
  x: number[];
  y: number[];
  z?: number[];
  text: string[];
}

@Component({
  selector: 'app-phate-plot',
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
  templateUrl: './phate-plot.html',
  styleUrl: './phate-plot.scss',
})
export class PhatePlot implements OnInit {
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
  protected customizationForm: FormGroup;
  protected showCustomization = signal(false);

  private _phateData: PHATEData | null = null;
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
      const savedSettings = await this.loadSavedSettings();
      if (savedSettings) {
        const completeSettings = { ...this.customizationForm.value, ...savedSettings };
        this.customizationForm.patchValue(completeSettings, { emitEvent: false });
      }

      const [data, annotations] = await Promise.all([
        this.wails.readJobOutputFile(this.jobId, 'phate_output.txt'),
        this.annotationService.loadAnnotationsForJob(this.jobId)
      ]);

      if (!data) {
        throw new Error('phate_output.txt file is empty or not found');
      }

      this._phateData = this.parsePHATEData(data);
      this._annotations = annotations;

      this.updatePlot();
      await this.wails.logToFile('[PHATE Plot] Plot data loaded and rendered successfully.');
    } catch (err: any) {
      this.error.set('Failed to load PHATE plot: ' + err.message);
      await this.wails.logToFile('[PHATE Plot] Error: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  private updatePlot() {
    if (!this._phateData) {
      return;
    }
    this.ngZone.run(() => {
      const settings = this.customizationForm.value;
      const data: PHATEData = this._phateData!;
      const is3D = data.z && data.z.length > 0;
      const markerSize = settings.markerSize || 10;
      const showGrid = settings.showGrid !== false;
      const useAnnotations = this._annotations.length > 0;

      if (useAnnotations) {
        const dataWithAnnotations = data.text.map((sample, index) => {
          const annotation = this._annotations.find(a => a.sample === sample);
          return {
            x: data.x[index],
            y: data.y[index],
            z: is3D ? data.z![index] : undefined,
            text: sample,
            condition: annotation ? annotation.condition : 'Unknown',
            color: annotation ? annotation.color : '#808080'
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
              marker: { size: is3D ? markerSize / 2 : markerSize, color: '#1976d2', line: { color: '#fff', width: is3D ? 0.5 : 1 } }
          };
          if (is3D && data.z) {
              trace.z = data.z;
          }
          this.plotData = [trace];
      }

      let newLayout: any;
      if (is3D) {
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : 'PHATE Analysis Results (3D)' },
          width: settings.width || 800,
          height: settings.height || 600,
          scene: {
            xaxis: { title: { text: 'PHATE 1' }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            yaxis: { title: { text: 'PHATE 2' }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            zaxis: { title: { text: 'PHATE 3' }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid }
          },
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: useAnnotations,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop, r: settings.marginRight, b: settings.marginBottom, l: settings.marginLeft }
        };
      } else {
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : 'PHATE Analysis Results' },
          width: settings.width || 800,
          height: settings.height || 600,
          xaxis: { title: { text: 'PHATE 1' }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          yaxis: { title: { text: 'PHATE 2' }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          plot_bgcolor: '#fafafa',
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: useAnnotations,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop, r: settings.marginRight, b: settings.marginBottom, l: settings.marginLeft }
        };
      }
      this.plotLayout = newLayout;
      this.revision++;
    });
  }

  private parsePHATEData(tsvData: string): PHATEData {
    const lines = tsvData.trim().split('\n');
    const headers = lines[0].split('\t');

    const xIndex = headers.indexOf('x_phate');
    const yIndex = headers.indexOf('y_phate');
    const zIndex = headers.indexOf('z_phate');
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
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, '.plot-settings-phate.json');
      if (savedSettings) {
        return JSON.parse(savedSettings);
      }
    } catch (err: any) {
      await this.wails.logToFile(`[PHATE Plot] No saved settings found or failed to load: ${err.message}`);
    }
    return null;
  }

  private async saveSettings() {
    try {
      const settings = this.customizationForm.value;
      await this.wails.writeJobOutputFile(this.jobId, '.plot-settings-phate.json', JSON.stringify(settings, null, 2));
      await this.wails.logToFile(`[PHATE Plot] Settings saved for job: ${this.jobId}`);
    } catch (err: any) {
      await this.wails.logToFile(`[PHATE Plot] Failed to save settings: ${err.message}`);
    }
  }

  async applyCustomization() {
    await this.saveSettings();
    this.updatePlot();
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('phate-plot-div', {
        filename: 'phate_plot',
        width: this.customizationForm.value.width || 1200,
        height: this.customizationForm.value.height || 800
      });
    } catch (error) {
      await this.wails.logToFile(`[PHATE Plot] Failed to export to SVG: ${error}`);
    }
  }
}
