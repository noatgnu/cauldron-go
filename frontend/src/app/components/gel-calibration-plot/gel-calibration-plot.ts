import { Component, computed, input, ChangeDetectionStrategy } from '@angular/core';
import { PlotlyModule } from 'angular-plotly.js';
import { GelCalibrationCurve } from '../../core/services/wails';

@Component({
  imports: [PlotlyModule],
  selector: 'app-gel-calibration-plot',
  styleUrl: './gel-calibration-plot.scss',
  templateUrl: './gel-calibration-plot.html',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class GelCalibrationPlot {
  curve = input<GelCalibrationCurve | null>(null);

  graphLayout = {
    autosize: true,
    height: 320,
    margin: { l: 50, r: 20, b: 40, t: 30 },
    xaxis: { title: 'Relative migration distance' },
    yaxis: { title: 'log10(MW)' },
    showlegend: false
  };

  graphData = computed(() => {
    const c = this.curve();
    if (!c || c.points.length === 0) return [];

    const points = [...c.points].sort((a, b) => a.position - b.position);
    const minX = points[0].position;
    const maxX = points[points.length - 1].position;
    const linePoints = [minX, maxX];
    const lineValues = linePoints.map(x => c.slope * x + c.intercept);

    return [
      {
        x: points.map(p => p.position),
        y: points.map(p => p.logMw),
        mode: 'markers',
        type: 'scatter',
        name: 'Markers',
        text: points.map(p => `${p.mw} kDa`),
        marker: { size: 10 }
      },
      {
        x: linePoints,
        y: lineValues,
        mode: 'lines',
        type: 'scatter',
        name: 'Fit'
      }
    ];
  });

  rSquaredLabel = computed(() => {
    const c = this.curve();
    return c ? `R² = ${c.rSquared.toFixed(4)}` : '';
  });
}
