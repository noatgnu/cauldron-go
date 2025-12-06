import { Component, Input, OnInit, signal } from '@angular/core';
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
import { AnnotationService, Annotation, AnnotationColors } from '../../../core/services/annotation.service';
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
  protected plotData = signal<any[]>([]);
  protected plotLayout = signal<any>({});
  protected plotConfig = signal<any>({
    responsive: true,
    displayModeBar: true,
    displaylogo: false
  });
  protected customizationForm: FormGroup;
  protected showCustomization = signal(false);
  protected annotations: Annotation[] = [];
  protected conditionColors: AnnotationColors = {};

  constructor(
    private wails: Wails,
    private fb: FormBuilder,
    private annotationService: AnnotationService,
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
    await this.loadAndRenderPlot();
    this.loadSavedSettings();
  }

  async loadAndRenderPlot() {
    try {
      this.loading.set(true);

      const data = await this.wails.readJobOutputFile(this.jobId, 'phate_output.txt');

      if (!data) {
        throw new Error('phate_output.txt file is empty or not found');
      }

      await this.tryLoadAnnotations();

      const phateData = this.parsePHATEData(data);
      this.createPlotData(phateData);
    } catch (err: any) {
      this.error.set('Failed to load PHATE plot: ' + err.message);
      await this.wails.logToFile('[PHATE Plot] Error: ' + err.message);
    } finally {
      this.loading.set(false);
    }
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

  private async tryLoadAnnotations() {
    this.annotations = await this.annotationService.loadAnnotationsForJob(this.jobId);
    if (this.annotations.length > 0) {
      this.conditionColors = await this.annotationService.loadConditionColorsForJob(this.jobId);
    }
  }

  private createPlotData(data: PHATEData) {
    const is3D = data.z && data.z.length > 0;

    const useConditions = this.annotations.length > 0;
    const colors = ['#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2', '#0097a7', '#c2185b', '#5d4037', '#fbc02d', '#00796b'];

    if (useConditions) {
      const conditionMap = new Map<string, number[]>();
      data.text.forEach((sample, index) => {
        const condition = this.annotationService.getCondition(sample, this.annotations);
        if (!conditionMap.has(condition)) {
          conditionMap.set(condition, []);
        }
        conditionMap.get(condition)!.push(index);
      });

      const traces = Array.from(conditionMap.entries()).map(([condition, indices], colorIndex) => {
        const color = colors[colorIndex % colors.length];
        const trace: any = {
          x: indices.map(i => data.x[i]),
          y: indices.map(i => data.y[i]),
          mode: 'markers',
          type: is3D ? 'scatter3d' : 'scatter',
          name: condition,
          text: indices.map(i => data.text[i]),
          hoverinfo: 'text+name',
          marker: {
            size: is3D ? 6 : 10,
            color: color,
            line: {
              color: '#fff',
              width: is3D ? 0.5 : 1
            }
          }
        };

        if (is3D && data.z) {
          trace.z = indices.map(i => data.z![i]);
        }

        return trace;
      });

      this.plotData.set(traces);

      if (is3D) {
        this.plotLayout.set({
          title: 'PHATE Analysis Results (3D)',
          scene: {
            xaxis: {
              title: 'PHATE 1',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            },
            yaxis: {
              title: 'PHATE 2',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            },
            zaxis: {
              title: 'PHATE 3',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            }
          },
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: true,
          legend: { x: 1.02, y: 1 },
          margin: { t: 50, r: 150, b: 50, l: 50 }
        });
      } else {
        this.plotLayout.set({
          title: 'PHATE Analysis Results',
          xaxis: {
            title: 'PHATE 1',
            zeroline: true,
            gridcolor: '#e0e0e0'
          },
          yaxis: {
            title: 'PHATE 2',
            zeroline: true,
            gridcolor: '#e0e0e0'
          },
          plot_bgcolor: '#fafafa',
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: true,
          legend: { x: 1.02, y: 1 },
          margin: { t: 50, r: 150, b: 50, l: 50 }
        });
      }
    } else {
      if (is3D) {
        this.plotData.set([{
          x: data.x,
          y: data.y,
          z: data.z,
          mode: 'markers',
          type: 'scatter3d',
          text: data.text,
          hoverinfo: 'text',
          marker: {
            size: 6,
            color: '#1976d2',
            line: {
              color: '#fff',
              width: 0.5
            }
          }
        }]);

        this.plotLayout.set({
          title: 'PHATE Analysis Results (3D)',
          scene: {
            xaxis: {
              title: 'PHATE 1',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            },
            yaxis: {
              title: 'PHATE 2',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            },
            zaxis: {
              title: 'PHATE 3',
              gridcolor: '#e0e0e0',
              backgroundcolor: '#fafafa'
            }
          },
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: false,
          margin: { t: 50, r: 50, b: 50, l: 50 }
        });
      } else {
        this.plotData.set([{
          x: data.x,
          y: data.y,
          mode: 'markers',
          type: 'scatter',
          text: data.text,
          hoverinfo: 'text',
          marker: {
            size: 10,
            color: '#1976d2',
            line: {
              color: '#fff',
              width: 1
            }
          }
        }]);

        this.plotLayout.set({
          title: 'PHATE Analysis Results',
          xaxis: {
            title: 'PHATE 1',
            zeroline: true,
            gridcolor: '#e0e0e0'
          },
          yaxis: {
            title: 'PHATE 2',
            zeroline: true,
            gridcolor: '#e0e0e0'
          },
          plot_bgcolor: '#fafafa',
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: false,
          margin: { t: 50, r: 50, b: 50, l: 50 }
        });
      }
    }
  }

  private async loadSavedSettings() {
    try {
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, '.plot-settings-phate.json');
      if (savedSettings) {
        const settings = JSON.parse(savedSettings);
        this.customizationForm.patchValue(settings);
        this.applyCustomization();
      }
    } catch (err: any) {
      await this.wails.logToFile(`[PHATE Plot] No saved settings found or failed to load: ${err.message}`);
    }
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
    const values = this.customizationForm.value;
    const is3D = this.plotData()[0]?.type === 'scatter3d';

    if (is3D) {
      this.plotLayout.update(layout => ({
        ...layout,
        title: values.title !== undefined && values.title !== null ? values.title : layout.title,
        width: values.width,
        height: values.height,
        scene: {
          ...layout.scene,
          xaxis: {
            ...layout.scene?.xaxis,
            title: layout.scene?.xaxis?.title,
            showgrid: values.showGrid
          },
          yaxis: {
            ...layout.scene?.yaxis,
            title: layout.scene?.yaxis?.title,
            showgrid: values.showGrid
          },
          zaxis: {
            ...layout.scene?.zaxis,
            title: layout.scene?.zaxis?.title,
            showgrid: values.showGrid
          }
        }
      }));
    } else {
      this.plotLayout.update(layout => ({
        ...layout,
        title: values.title !== undefined && values.title !== null ? values.title : layout.title,
        width: values.width,
        height: values.height,
        xaxis: {
          ...layout.xaxis,
          title: layout.xaxis?.title,
          showgrid: values.showGrid
        },
        yaxis: {
          ...layout.yaxis,
          title: layout.yaxis?.title,
          showgrid: values.showGrid
        }
      }));
    }

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
