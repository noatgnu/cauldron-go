import { Component, OnInit, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { ImportedFileSelection } from '../../../components/imported-file-selection/imported-file-selection';
import { Wails } from '../../../core/services/wails';
import { PlotlyExport } from '../../../core/services/plotly-export';
import * as Plotly from 'plotly.js-dist-min';

interface TraceConfig {
  name: string;
  column: string;
  color: string;
  fillColor: string;
  showBox: boolean;
  showMeanLine: boolean;
  pointsMode: string;
  jitter: number;
  pointSize: number;
  bandwidth: number;
}

@Component({
  selector: 'app-violin-plot',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatButtonModule,
    MatIconModule,
    MatExpansionModule,
    ImportedFileSelection
  ],
  templateUrl: './violin-plot.html',
  styleUrl: './violin-plot.scss'
})
export class ViolinPlot implements OnInit {
  protected form: FormGroup;
  protected columns: string[] = [];
  protected selectedColumns = signal<string[]>([]);
  protected traceConfigs = signal<TraceConfig[]>([]);
  protected plotData: any = null;
  protected parseFloat = parseFloat;
  protected parseInt = parseInt;

  private defaultColors = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
  ];

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private plotlyExport: PlotlyExport
  ) {
    this.form = this.fb.group({
      filePath: ['', Validators.required],
      selectedCols: [[], Validators.required],
      plotTitle: ['Violin Plot'],
      xAxisTitle: [''],
      yAxisTitle: ['Value'],
      orientation: ['v'],
      showLegend: [true],
      violinMode: ['group'],
      bandwidth: [0]
    });

    effect(() => {
      const cols = this.selectedColumns();
      this.updateTraceConfigs(cols);
    });
  }

  ngOnInit() {}

  async openFile() {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls['filePath'].setValue(path);
        const preview = await this.wails.parseDataFile(path, 1);
        if (preview && preview.headers) {
          this.columns = preview.headers;
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(e: string, formControl: string) {
    this.form.controls[formControl].setValue(e);
  }

  updateColumns(cols: any) {
    if (Array.isArray(cols)) {
      this.columns = cols;
    } else if (cols && typeof cols === 'object' && 'target' in cols) {
      return;
    } else {
      this.columns = [];
    }
  }

  onColumnsChange(event: any) {
    const selected = event.value;
    this.selectedColumns.set(selected);
  }

  updateTraceConfigs(cols: string[]) {
    const configs: TraceConfig[] = cols.map((col, idx) => {
      const existing = this.traceConfigs().find(c => c.column === col);
      if (existing) return existing;

      const color = this.defaultColors[idx % this.defaultColors.length];
      return {
        name: col,
        column: col,
        color: color,
        fillColor: color,
        showBox: true,
        showMeanLine: true,
        pointsMode: 'outliers',
        jitter: 0.3,
        pointSize: 4,
        bandwidth: this.form.value.bandwidth || 0
      };
    });
    this.traceConfigs.set(configs);
  }

  updateTraceConfig(index: number, field: string, value: any) {
    const configs = [...this.traceConfigs()];
    configs[index] = { ...configs[index], [field]: value };
    this.traceConfigs.set(configs);
  }

  async generatePlot() {
    if (this.form.invalid || this.selectedColumns().length === 0) return;

    try {
      const filePath = this.form.value.filePath;
      const fullData = await this.wails.parseDataFile(filePath, 0);

      if (!fullData || !fullData.rows) {
        alert('Failed to load data');
        return;
      }

      const traces: any[] = [];
      const configs = this.traceConfigs();
      const headers = fullData.headers;

      configs.forEach((config, idx) => {
        const columnIndex = headers.indexOf(config.column);
        if (columnIndex === -1) {
          return;
        }

        const columnData = fullData.rows.map((row: any) => {
          const val = row[columnIndex];
          return val !== null && val !== undefined && val !== '' ? parseFloat(val) : null;
        }).filter((v: any) => v !== null && !isNaN(v));

        if (columnData.length === 0) {
          console.warn(`No valid data for column: ${config.column}`);
          return;
        }

        const trace: any = {
          type: 'violin',
          name: config.name,
          y: this.form.value.orientation === 'v' ? columnData : undefined,
          x: this.form.value.orientation === 'h' ? columnData : undefined,
          orientation: this.form.value.orientation,
          marker: {
            color: config.color,
            size: config.pointSize,
            line: {
              color: config.color,
              width: 1
            }
          },
          line: {
            color: config.color,
            width: 2
          },
          fillcolor: this.hexToRgba(config.fillColor, 0.6),
          box: {
            visible: config.showBox,
            fillcolor: this.hexToRgba(config.fillColor, 0.3),
            line: {
              color: config.color,
              width: 1.5
            }
          },
          meanline: {
            visible: config.showMeanLine,
            color: config.color,
            width: 2
          },
          points: config.pointsMode,
          jitter: config.jitter,
          pointpos: 0,
          bandwidth: config.bandwidth > 0 ? config.bandwidth : undefined
        };

        traces.push(trace);
      });

      const layout: any = {
        title: this.form.value.plotTitle,
        xaxis: {
          title: this.form.value.orientation === 'v' ? this.form.value.xAxisTitle : this.form.value.yAxisTitle,
          zeroline: false
        },
        yaxis: {
          title: this.form.value.orientation === 'v' ? this.form.value.yAxisTitle : this.form.value.xAxisTitle,
          zeroline: false
        },
        violinmode: this.form.value.violinMode,
        showlegend: this.form.value.showLegend,
        hovermode: 'closest',
        plot_bgcolor: 'white',
        paper_bgcolor: 'white'
      };

      const config: any = {
        responsive: true,
        displayModeBar: true,
        displaylogo: false,
        modeBarButtonsToRemove: ['lasso2d', 'select2d'] as any,
        toImageButtonOptions: {
          format: 'png' as const,
          filename: 'violin_plot',
          height: 800,
          width: 1200,
          scale: 2
        }
      };

      const plotDiv = document.getElementById('violin-plot-div');
      if (plotDiv) {
        await Plotly.newPlot(plotDiv, traces, layout, config);
        this.plotData = { traces, layout, config };
      }
    } catch (error) {
      console.error('Failed to generate plot:', error);
      alert('Failed to generate plot. Please check your data.');
    }
  }

  hexToRgba(hex: string, alpha: number): string {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    if (result) {
      const r = parseInt(result[1], 16);
      const g = parseInt(result[2], 16);
      const b = parseInt(result[3], 16);
      return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    }
    return `rgba(0, 0, 0, ${alpha})`;
  }

  reset() {
    this.form.reset({
      filePath: '',
      selectedCols: [],
      plotTitle: 'Violin Plot',
      xAxisTitle: '',
      yAxisTitle: 'Value',
      orientation: 'v',
      showLegend: true,
      violinMode: 'group',
      bandwidth: 0
    });
    this.columns = [];
    this.selectedColumns.set([]);
    this.traceConfigs.set([]);
    this.plotData = null;

    const plotDiv = document.getElementById('violin-plot-div');
    if (plotDiv) {
      Plotly.purge(plotDiv);
    }
  }

  async loadExample() {
    try {
      const filePath = await this.wails.getExampleFilePath('diann', 'imputed.data.txt');
      this.form.patchValue({ filePath });
      const preview = await this.wails.parseDataFile(filePath, 1);
      if (preview && preview.headers) {
        this.columns = preview.headers;
        const sampleColumns = this.columns.filter(h => !['Protein.Ids', 'Precursor.Id', 'Genes'].includes(h));
        const selected = sampleColumns.slice(0, 4);
        this.form.patchValue({ selectedCols: selected });
        this.selectedColumns.set(selected);
      }
    } catch (error) {
      alert('Failed to load example data. Please ensure example files are available.');
    }
  }

  async exportToSVG() {
    try {
      await this.plotlyExport.exportToSVG('violin-plot-div', {
        filename: 'violin_plot',
        width: 1200,
        height: 800
      });
    } catch (error) {
      alert('Failed to export plot to SVG. Please try again.');
    }
  }
}
