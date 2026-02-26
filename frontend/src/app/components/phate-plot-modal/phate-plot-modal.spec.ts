import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PhatePlotModal } from './phate-plot-modal';
import { Wails } from '../../core/services/wails';
import { ThemeService } from '../../core/services/theme.service';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock } from '../../core/mocks/plotly-mock';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PhatePlotModal', () => {
  let component: PhatePlotModal;
  let fixture: ComponentFixture<PhatePlotModal>;
  let dialogRefSpy: any;
  let wailsMock: any;
  let themeServiceMock: any;

  beforeAll(() => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  });

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      readFile: vi.fn().mockResolvedValue(''),
      isWails: false,
      getSettings: vi.fn().mockResolvedValue({})
    };
    themeServiceMock = {
      colorPalette: vi.fn().mockReturnValue([]),
      colorblindPalette: vi.fn().mockReturnValue('default')
    };

    await TestBed.configureTestingModule({
      imports: [
        PhatePlotModal, 
        NoopAnimationsModule,
        PlotlyModule.forRoot(PlotlyMock)
      ],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { jobId: 'test' } },
        { provide: Wails, useValue: wailsMock },
        { provide: ThemeService, useValue: themeServiceMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PhatePlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
