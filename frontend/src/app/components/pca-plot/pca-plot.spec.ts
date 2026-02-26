import { ComponentFixture, TestBed } from '@angular/core/testing';
import { PcaPlot } from './pca-plot';
import { ThemeService } from '../../core/services/theme.service';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock, mockMatchMedia } from '../../core/mocks/plotly-mock';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { ReactiveFormsModule } from '@angular/forms';
import { vi, beforeAll } from 'vitest';

describe('PcaPlot (Component)', () => {
  let component: PcaPlot;
  let fixture: ComponentFixture<PcaPlot>;
  let themeServiceMock: any;

  beforeAll(() => {
    mockMatchMedia();
  });

  beforeEach(async () => {
    themeServiceMock = {
      colorPalette: vi.fn().mockReturnValue(['#ff0000', '#00ff00', '#0000ff']),
      colorblindPalette: vi.fn().mockReturnValue('default')
    };

    await TestBed.configureTestingModule({
      imports: [
        PcaPlot,
        PlotlyModule.forRoot(PlotlyMock),
        NoopAnimationsModule,
        ReactiveFormsModule
      ],
      providers: [
        { provide: ThemeService, useValue: themeServiceMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PcaPlot);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
