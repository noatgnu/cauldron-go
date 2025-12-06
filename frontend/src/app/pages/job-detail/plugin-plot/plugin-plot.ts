import { Component, Input, OnInit, signal } from '@angular/core';
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
})
export class PluginPlot implements OnInit {
  @Input() jobId!: string;
  @Input() plotFileName!: string;
  @Input() plotTitle?: string;

  protected loading = signal(true);
  protected error = signal('');
  protected plotData = signal<any[]>([]);
  protected plotLayout = signal<any>({});
  protected plotConfig = signal<any>({
    responsive: true,
    displayModeBar: true,
    displaylogo: false
  });

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
      await this.wails.logToFile(`[Plugin Plot] Loading plot from: ${this.plotFileName}`);

      const plotDataJson = await this.wails.readJobOutputFile(this.jobId, this.plotFileName);

      if (!plotDataJson) {
        throw new Error(`${this.plotFileName} file is empty or not found`);
      }

      const plotlyData: PlotlyData = JSON.parse(plotDataJson);

      if (!plotlyData.data || !Array.isArray(plotlyData.data)) {
        throw new Error('Invalid plot data format: missing or invalid data array');
      }

      this.plotData.set(plotlyData.data);

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

      this.plotLayout.set(layout);

      if (plotlyData.config) {
        this.plotConfig.update(config => ({ ...config, ...plotlyData.config }));
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
