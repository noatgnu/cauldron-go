import { Injectable } from '@angular/core';
import * as Plotly from 'plotly.js-dist-min';

export interface PlotlyExportOptions {
  width?: number;
  height?: number;
  filename?: string;
  format?: 'png' | 'svg' | 'jpeg' | 'webp';
}

@Injectable({
  providedIn: 'root',
})
export class PlotlyExport {

  async exportToSVG(plotDivId: string, options?: PlotlyExportOptions): Promise<void> {
    const plotDiv = document.getElementById(plotDivId);
    if (!plotDiv) {
      throw new Error(`Plot div with ID '${plotDivId}' not found`);
    }

    const defaultOptions: PlotlyExportOptions = {
      width: 1200,
      height: 800,
      filename: 'plot',
      format: 'svg'
    };

    const exportOptions = { ...defaultOptions, ...options };

    try {
      await Plotly.downloadImage(plotDiv, {
        format: exportOptions.format!,
        width: exportOptions.width!,
        height: exportOptions.height!,
        filename: exportOptions.filename!
      });
    } catch (error) {
      throw new Error(`Failed to export plot: ${error}`);
    }
  }

  async exportToPNG(plotDivId: string, options?: PlotlyExportOptions): Promise<void> {
    return this.exportToSVG(plotDivId, { ...options, format: 'png' });
  }

  async exportToJPEG(plotDivId: string, options?: PlotlyExportOptions): Promise<void> {
    return this.exportToSVG(plotDivId, { ...options, format: 'jpeg' });
  }

  async exportToWebP(plotDivId: string, options?: PlotlyExportOptions): Promise<void> {
    return this.exportToSVG(plotDivId, { ...options, format: 'webp' });
  }

  async exportToImage(plotDivId: string, format: 'png' | 'svg' | 'jpeg' | 'webp', options?: PlotlyExportOptions): Promise<void> {
    return this.exportToSVG(plotDivId, { ...options, format });
  }
}
