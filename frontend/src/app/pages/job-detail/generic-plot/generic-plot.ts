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

interface PlotData {
  x: number[];
  y: number[];
  z?: number[];
  text: string[];
  colorBy?: string[];
}

@Component({
  selector: 'app-generic-plot',
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
  templateUrl: './generic-plot.html',
  styleUrl: './generic-plot.scss',
})
export class GenericPlot implements OnInit {
  @Input() jobId!: string;
  @Input() plotConfig!: any;
  @Input() pluginOutputs!: any[];

  protected loading = signal(true);
  protected error = signal('');
  plotData: any[] = [];
  plotLayout: any = {};
  plotConfig_: any = {
    responsive: true,
    displayModeBar: true,
    displaylogo: false,
    modeBarButtonsToRemove: ['toImage']
  };
  revision: number = 0;
  customizationForm: FormGroup;
  protected showCustomization = signal(false);

  private _data: PlotData | null = null;
  private _annotations: Annotation[] = [];
  private _dataFile: string = '';

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
      await this.wails.logToFile(`[Generic Plot] Loading plot: ${this.plotConfig.name}`);

      this._dataFile = this.resolveDataFile();
      if (!this._dataFile) {
        throw new Error('Could not resolve data file from dataSource');
      }

      await this.wails.logToFile(`[Generic Plot] Data file resolved to: ${this._dataFile}`);

      this.initializeCustomizationForm();

      const savedSettings = await this.loadSavedSettings();
      if (savedSettings) {
        const completeSettings = { ...this.customizationForm.value, ...savedSettings };
        this.customizationForm.patchValue(completeSettings, { emitEvent: false });
      }

      const [data, annotations] = await Promise.all([
        this.wails.readJobOutputFile(this.jobId, this._dataFile),
        this.annotationService.loadAnnotationsForJob(this.jobId)
      ]);

      if (!data) {
        throw new Error(`${this._dataFile} file is empty or not found`);
      }

      this._data = this.parseData(data);
      this._annotations = annotations;

      this.updatePlot();
      await this.wails.logToFile('[Generic Plot] Plot data loaded successfully');

    } catch (err: any) {
      const errorMsg = 'Failed to load generic plot: ' + (err.message || String(err));
      this.error.set(errorMsg);
      await this.wails.logToFile('[Generic Plot] Error: ' + errorMsg);
    } finally {
      this.loading.set(false);
    }
  }

  private resolveDataFile(): string {
    const dataSourceName = this.plotConfig.dataSource;
    const output = this.pluginOutputs.find((o: any) => o.name === dataSourceName);
    if (!output) {
      throw new Error(`Output not found: ${dataSourceName}`);
    }
    return output.path;
  }

  private initializeCustomizationForm() {
    if (!this.plotConfig.customization || this.plotConfig.customization.length === 0) {
      return;
    }

    const formControls: any = {};
    this.plotConfig.customization.forEach((custom: any) => {
      formControls[custom.name] = [custom.default !== undefined ? custom.default : ''];
    });

    this.customizationForm = this.fb.group(formControls);
  }

  private updatePlot() {
    if (!this._data) {
      return;
    }

    this.ngZone.run(() => {
      const settings = this.customizationForm.value;
      const data: PlotData = this._data!;
      const is3D = data.z && data.z.length > 0;
      const useAnnotations = this._annotations.length > 0;
      const markerSize = settings.markerSize || 10;
      const showGrid = settings.showGrid !== false;

      if (useAnnotations && this.plotConfig.config?.axes?.colorBy) {
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
              color: points.map(p => p.color),
              line: { color: '#fff', width: is3D ? 0.5 : 1 }
            }
          };
          if (is3D) {
            trace.z = points.map(p => p.z);
          }
          return trace;
        });

      } else if (data.colorBy && data.colorBy.length > 0) {
        const uniqueCategories = [...new Set(data.colorBy)];
        const colors = ['#1976d2', '#388e3c', '#d32f2f', '#f57c00', '#7b1fa2', '#0097a7', '#c2185b', '#5d4037'];

        this.plotData = uniqueCategories.map((category, index) => {
          const indices = data.colorBy!.map((c, i) => c === category ? i : -1).filter(i => i >= 0);
          const trace: any = {
            x: indices.map(i => data.x[i]),
            y: indices.map(i => data.y[i]),
            mode: 'markers',
            type: is3D ? 'scatter3d' : 'scatter',
            name: category,
            text: indices.map(i => data.text[i]),
            hoverinfo: 'text+name',
            marker: {
              size: is3D ? markerSize / 2 : markerSize,
              color: colors[index % colors.length],
              line: { color: '#fff', width: is3D ? 0.5 : 1 }
            }
          };
          if (is3D) {
            trace.z = indices.map(i => data.z![i]);
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
            color: '#1976d2',
            line: { color: '#fff', width: is3D ? 0.5 : 1 }
          }
        };
        if (is3D && data.z) {
          trace.z = data.z;
        }
        this.plotData = [trace];
      }

      const axes = this.plotConfig.config?.axes || {};
      const xLabel = axes.x || 'X';
      const yLabel = axes.y || 'Y';
      const zLabel = axes.z || 'Z';
      const showLegend = useAnnotations || (data.colorBy && data.colorBy.length > 0);

      let newLayout: any;
      if (is3D) {
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : this.plotConfig.name },
          width: settings.width || 800,
          height: settings.height || 600,
          scene: {
            xaxis: { title: { text: xLabel }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            yaxis: { title: { text: yLabel }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid },
            zaxis: { title: { text: zLabel }, gridcolor: '#e0e0e0', backgroundcolor: '#fafafa', showgrid: showGrid }
          },
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: showLegend,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop || 100, r: settings.marginRight || 100, b: settings.marginBottom || 100, l: settings.marginLeft || 100 }
        };
      } else {
        newLayout = {
          title: { text: settings.title !== '' ? settings.title : this.plotConfig.name },
          width: settings.width || 800,
          height: settings.height || 600,
          xaxis: { title: { text: xLabel }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          yaxis: { title: { text: yLabel }, zeroline: true, gridcolor: '#e0e0e0', showgrid: showGrid, automargin: true },
          plot_bgcolor: '#fafafa',
          paper_bgcolor: '#ffffff',
          hovermode: 'closest',
          showlegend: showLegend,
          legend: { x: 1.02, y: 1 },
          margin: { t: settings.marginTop || 100, r: settings.marginRight || 100, b: settings.marginBottom || 100, l: settings.marginLeft || 100 }
        };
      }
      this.plotLayout = newLayout;
      this.revision++;
    });
  }

  private parseData(tsvData: string): PlotData {
    const lines = tsvData.trim().split('\n');
    const headers = lines[0].split('\t');

    const axes = this.plotConfig.config?.axes || {};
    const xCol = axes.x;
    const yCol = axes.y;
    const zCol = axes.z;
    const labelCol = axes.labels;
    const colorByCol = axes.colorBy;

    const xIndex = headers.indexOf(xCol);
    const yIndex = headers.indexOf(yCol);
    const zIndex = zCol ? headers.indexOf(zCol) : -1;
    const labelIndex = labelCol ? headers.indexOf(labelCol) : -1;
    const colorByIndex = colorByCol ? headers.indexOf(colorByCol) : -1;

    if (xIndex === -1 || yIndex === -1) {
      throw new Error(`Missing required columns. Found: ${headers.join(', ')}. Looking for x: ${xCol}, y: ${yCol}`);
    }

    const x: number[] = [];
    const y: number[] = [];
    const z: number[] = [];
    const text: string[] = [];
    const colorBy: string[] = [];
    const has3D = zIndex !== -1;

    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split('\t');
      const maxIndex = Math.max(xIndex, yIndex, zIndex, labelIndex, colorByIndex);
      if (values.length > maxIndex) {
        x.push(parseFloat(values[xIndex]));
        y.push(parseFloat(values[yIndex]));
        if (has3D) {
          z.push(parseFloat(values[zIndex]));
        }
        if (labelIndex !== -1) {
          const samplePath = values[labelIndex];
          const sampleName = samplePath.split(/[/\\]/).pop() || samplePath;
          text.push(sampleName);
        } else {
          text.push(`Point ${i}`);
        }
        if (colorByIndex !== -1) {
          colorBy.push(values[colorByIndex]);
        }
      }
    }

    return has3D ? { x, y, z, text, colorBy: colorBy.length > 0 ? colorBy : undefined } : { x, y, text, colorBy: colorBy.length > 0 ? colorBy : undefined };
  }

  private async loadSavedSettings(): Promise<any> {
    try {
      const settingsFile = `.plot-settings-${this.plotConfig.id}.json`;
      const savedSettings = await this.wails.readJobOutputFile(this.jobId, settingsFile);
      if (savedSettings) {
        return JSON.parse(savedSettings);
      }
    } catch (err: any) {
      await this.wails.logToFile(`[Generic Plot] No saved settings found: ${err.message}`);
    }
    return null;
  }

  private async saveSettings() {
    try {
      const settings = this.customizationForm.value;
      const settingsFile = `.plot-settings-${this.plotConfig.id}.json`;
      await this.wails.writeJobOutputFile(this.jobId, settingsFile, JSON.stringify(settings, null, 2));
      await this.wails.logToFile(`[Generic Plot] Settings saved for plot: ${this.plotConfig.id}`);
    } catch (err: any) {
      await this.wails.logToFile(`[Generic Plot] Failed to save settings: ${err.message}`);
    }
  }

  async applyCustomization() {
    await this.saveSettings();
    this.updatePlot();
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('generic-plot-div', {
        filename: this.plotConfig.id || 'plot',
        width: this.customizationForm.value.width || 1200,
        height: this.customizationForm.value.height || 800
      });
    } catch (error) {
      await this.wails.logToFile(`[Generic Plot] Failed to export to SVG: ${error}`);
    }
  }
}
