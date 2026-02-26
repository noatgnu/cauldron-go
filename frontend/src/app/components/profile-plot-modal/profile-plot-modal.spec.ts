import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { ProfilePlotModal } from './profile-plot-modal';
import { Wails } from '../../core/services/wails';
import { ThemeService } from '../../core/services/theme.service';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock, mockMatchMedia } from '../../core/mocks/plotly-mock';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('ProfilePlotModal', () => {
  let component: ProfilePlotModal;
  let fixture: ComponentFixture<ProfilePlotModal>;
  let dialogRefSpy: any;
  let wailsMock: any;
  let themeServiceMock: any;

  beforeAll(() => {
    mockMatchMedia();
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
        ProfilePlotModal, 
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

    fixture = TestBed.createComponent(ProfilePlotModal);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
