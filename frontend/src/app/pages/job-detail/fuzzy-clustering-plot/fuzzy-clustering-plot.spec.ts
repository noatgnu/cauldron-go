import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FuzzyClusteringPlot } from './fuzzy-clustering-plot';
import { Wails } from '../../../core/services/wails';
import { AnnotationService } from '../../../core/services/annotation.service';
import { PlotlyExport } from '../../../core/services/plotly-export';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock } from '../../../core/mocks/plotly-mock';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ReactiveFormsModule } from '@angular/forms';

describe('FuzzyClusteringPlot', () => {
  let component: FuzzyClusteringPlot;
  let fixture: ComponentFixture<FuzzyClusteringPlot>;
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
        FuzzyClusteringPlot, 
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

    fixture = TestBed.createComponent(FuzzyClusteringPlot);
    component = fixture.componentInstance;
    component.jobId = 'test-job';
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
