import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { SampleAnnotation } from './sample-annotation';
import { Wails } from '../../core/services/wails';
import { ThemeService } from '../../core/services/theme.service';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('SampleAnnotation', () => {
  let component: SampleAnnotation;
  let fixture: ComponentFixture<SampleAnnotation>;
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
      openDataFileDialog: vi.fn(),
      parseDataFile: vi.fn(),
      isWails: false,
      getSettings: vi.fn().mockResolvedValue({})
    };
    themeServiceMock = {
      colorPalette: vi.fn().mockReturnValue([]),
      colorblindPalette: vi.fn().mockReturnValue('default')
    };

    await TestBed.configureTestingModule({
      imports: [SampleAnnotation, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        { provide: MAT_DIALOG_DATA, useValue: { columns: [] } },
        { provide: Wails, useValue: wailsMock },
        { provide: ThemeService, useValue: themeServiceMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SampleAnnotation);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
