import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PcaPlot } from './pca-plot';
import { Wails } from '../../../core/services/wails';
import { AnnotationService } from '../../../core/services/annotation.service';
import { PlotlyExport } from '../../../core/services/plotly-export';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock } from '../../../core/mocks/plotly-mock';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';
import { ReactiveFormsModule } from '@angular/forms';

describe('PcaPlot', () => {
  let component: PcaPlot;
  let fixture: ComponentFixture<PcaPlot>;
  let wailsMock: any;
  let annotationServiceMock: any;
  let plotlyExportMock: any;

  beforeEach(async () => {
    wailsMock = {
      logToFile: vi.fn().mockResolvedValue(undefined),
      readJobOutputFile: vi.fn().mockResolvedValue('')
    };
    annotationServiceMock = {
      loadAnnotationsForJob: vi.fn().mockResolvedValue([])
    };
    plotlyExportMock = {
      exportToSVG: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [
        PcaPlot, 
        PlotlyModule.forRoot(PlotlyMock), 
        NoopAnimationsModule,
        ReactiveFormsModule
      ],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: AnnotationService, useValue: annotationServiceMock },
        { provide: PlotlyExport, useValue: plotlyExportMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PcaPlot);
    component = fixture.componentInstance;
    component.jobId = 'test-job';
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
