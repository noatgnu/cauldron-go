import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PcaPlotModal } from './pca-plot-modal';
import { Wails } from '../../core/services/wails';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock, mockMatchMedia } from '../../core/mocks/plotly-mock';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PcaPlotModal', () => {
  let component: PcaPlotModal;
  let fixture: ComponentFixture<PcaPlotModal>;
  let dialogRefSpy: any;
  let wailsMock: any;

  beforeAll(() => {
    mockMatchMedia();
  });

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      readFile: vi.fn().mockResolvedValue('')
    };

    await TestBed.configureTestingModule({
      imports: [
        PcaPlotModal, 
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

    fixture = TestBed.createComponent(PcaPlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
