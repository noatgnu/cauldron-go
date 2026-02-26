import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SettingsAppearance } from './settings-appearance';
import { ThemeService } from '../../../core/services/theme.service';
import { Wails } from '../../../core/services/wails';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { signal } from '@angular/core';

describe('SettingsAppearance', () => {
  let component: SettingsAppearance;
  let fixture: ComponentFixture<SettingsAppearance>;
  let themeServiceMock: any;
  let wailsMock: any;

  beforeAll(() => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(), // deprecated
        removeListener: vi.fn(), // deprecated
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  });

  beforeEach(async () => {
    wailsMock = {
      isWails: false,
      getSettings: vi.fn().mockResolvedValue({}),
      setSetting: vi.fn().mockResolvedValue(undefined)
    };
    themeServiceMock = {
      theme: vi.fn().mockReturnValue('system'),
      isDark: vi.fn().mockReturnValue(false),
      setTheme: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsAppearance, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: ThemeService, useValue: themeServiceMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsAppearance);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
