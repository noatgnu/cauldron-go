import { ComponentFixture, TestBed } from '@angular/core/testing';
import { GelCalibrationPlot } from './gel-calibration-plot';
import { GelCalibrationCurve } from '../../core/services/wails';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock, mockMatchMedia } from '../../core/mocks/plotly-mock';
import { beforeAll } from 'vitest';

const sampleCurve: GelCalibrationCurve = {
  slope: -2,
  intercept: 3,
  rSquared: 0.98,
  points: [
    { position: 0, logMw: 3, mw: 1000 },
    { position: 0.5, logMw: 2, mw: 100 },
    { position: 1, logMw: 1, mw: 10 }
  ]
} as GelCalibrationCurve;

describe('GelCalibrationPlot', () => {
  let component: GelCalibrationPlot;
  let fixture: ComponentFixture<GelCalibrationPlot>;

  beforeAll(() => {
    mockMatchMedia();
  });

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [GelCalibrationPlot, PlotlyModule.forRoot(PlotlyMock)]
    })
      .compileComponents();

    fixture = TestBed.createComponent(GelCalibrationPlot);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('produces no traces when no curve is set', () => {
    expect(component.graphData()).toEqual([]);
    expect(component.rSquaredLabel()).toBe('');
  });

  it('produces a markers trace and a fit-line trace for a sample curve', () => {
    fixture.componentRef.setInput('curve', sampleCurve);
    fixture.detectChanges();

    const traces = component.graphData();
    expect(traces.length).toBe(2);
    expect(traces[0]['type']).toBe('scatter');
    expect(traces[0]['mode']).toBe('markers');
    expect(traces[0]['x']).toEqual([0, 0.5, 1]);
    expect(traces[0]['y']).toEqual([3, 2, 1]);

    expect(traces[1]['mode']).toBe('lines');
    expect(traces[1]['x']).toEqual([0, 1]);
    expect(traces[1]['y']).toEqual([3, 1]);
  });

  it('formats the R-squared label', () => {
    fixture.componentRef.setInput('curve', sampleCurve);
    fixture.detectChanges();
    expect(component.rSquaredLabel()).toBe('R² = 0.9800');
  });
});
