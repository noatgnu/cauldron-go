import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { VolcanoPlotModal } from './volcano-plot-modal';
import { Wails } from '../../core/services/wails';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock } from '../../core/mocks/plotly-mock';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('VolcanoPlotModal', () => {
  let component: VolcanoPlotModal;
  let fixture: ComponentFixture<VolcanoPlotModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      readFile: vi.fn().mockResolvedValue('')
    };

    await TestBed.configureTestingModule({
      imports: [
        VolcanoPlotModal, 
        NoopAnimationsModule,
        PlotlyModule.forRoot(PlotlyMock)
      ],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { jobId: 'test' } },
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(VolcanoPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
