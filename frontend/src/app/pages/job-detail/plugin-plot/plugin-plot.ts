import { Component, Input, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { Wails } from '../../../core/services/wails';
import { PlotlyExport } from '../../../core/services/plotly-export';

interface PlotlyData {
  data: any[];
  layout: any;
  config?: any;
}

@Component({
  selector: 'app-plugin-plot',
  imports: [
    PlotlyModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatCardModule,
    MatButtonModule
  ],
  templateUrl: './plugin-plot.html',
  styleUrl: './plugin-plot.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PluginPlot implements OnInit {
  @Input() jobId!: string;
  @Input() plotFileName!: string;
  @Input() plotTitle?: string;
  @Input() plotType: string = 'json';

  protected loading = signal(true);
  protected error = signal('');
  protected imageData = signal<string>('');
  plotData: any[] = [];
  plotLayout: any = {
    paper_bgcolor: '#ffffff',
    hovermode: 'closest'
  };
  plotConfig: any = {
    responsive: true,
    displayModeBar: true,
    displaylogo: false,
    modeBarButtonsToRemove: ['toImage']
  };
  revision: number = 0;

  constructor(
    private wails: Wails,
    private plotlyExport: PlotlyExport
  ) {}

  async ngOnInit() {
    await this.loadPlot();
  }

  async loadPlot() {
    try {
      this.loading.set(true);
      await this.wails.logToFile(`[Plugin Plot] Loading plot from: ${this.plotFileName} (type: ${this.plotType})`);

      if (this.plotType === 'json') {
        await this.loadPlotlyPlot();
      } else if (this.plotType === 'svg' || this.plotType === 'png') {
        await this.loadImagePlot();
      } else if (this.plotType === 'pdf') {
        await this.wails.logToFile(`[Plugin Plot] PDF plots not yet supported: ${this.plotFileName}`);
        this.error.set('PDF plots are not yet supported. Please open the result folder to view the PDF.');
      } else {
        throw new Error(`Unsupported plot type: ${this.plotType}`);
      }

      await this.wails.logToFile(`[Plugin Plot] Successfully loaded plot: ${this.plotFileName}`);
    } catch (err: any) {
      const errorMsg = 'Failed to load plugin plot: ' + (err.message || String(err));
      this.error.set(errorMsg);
      await this.wails.logToFile(`[Plugin Plot] Error: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadPlotlyPlot() {
    const plotDataJson = await this.wails.readJobOutputFile(this.jobId, this.plotFileName);

    if (!plotDataJson) {
      throw new Error(`${this.plotFileName} file is empty or not found`);
    }

    const plotlyData: PlotlyData = JSON.parse(plotDataJson);

    if (!plotlyData.data || !Array.isArray(plotlyData.data)) {
      throw new Error('Invalid plot data format: missing or invalid data array');
    }

    this.plotData = plotlyData.data;

    const layout = plotlyData.layout || {};
    if (this.plotTitle) {
      layout.title = this.plotTitle;
    }

    if (!layout.paper_bgcolor) {
      layout.paper_bgcolor = '#ffffff';
    }
    if (!layout.hovermode) {
      layout.hovermode = 'closest';
    }

    this.plotLayout = layout;

    if (plotlyData.config) {
      this.plotConfig = { ...this.plotConfig, ...plotlyData.config };
    }

    this.revision++;
  }

  async loadImagePlot() {
    const imageContent = await this.wails.readJobOutputFile(this.jobId, this.plotFileName);

    if (!imageContent) {
      throw new Error(`${this.plotFileName} file is empty or not found`);
    }

    if (this.plotType === 'svg') {
      const blob = new Blob([imageContent], { type: 'image/svg+xml' });
      const dataUrl = URL.createObjectURL(blob);
      this.imageData.set(dataUrl);
    } else if (this.plotType === 'png') {
      this.imageData.set(`data:image/png;base64,${imageContent}`);
    }
  }

  async exportToSVG() {
    try {
      const plotId = `plugin-plot-${this.plotFileName.replace(/[^a-zA-Z0-9]/g, '-')}`;
      await this.plotlyExport.exportToSVG(plotId, {
        filename: this.plotFileName.replace('.json', ''),
        width: 1200,
        height: 800
      });
    } catch (error) {
      await this.wails.logToFile(`[Plugin Plot] Failed to export to SVG: ${error}`);
    }
  }

  getPlotId(): string {
    return `plugin-plot-${this.plotFileName.replace(/[^a-zA-Z0-9]/g, '-')}`;
  }
}
