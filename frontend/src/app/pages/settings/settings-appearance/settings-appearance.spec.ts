import { ComponentFixture, TestBed } from '@angular/core/testing';
import { SettingsAppearance } from './settings-appearance';
import { ThemeService } from '../../../core/services/theme.service';
import { Wails } from '../../../core/services/wails';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { COLOR_THEMES } from '../../../core/constants/color-themes';

describe('SettingsAppearance', () => {
  let component: SettingsAppearance;
  let fixture: ComponentFixture<SettingsAppearance>;
  let themeServiceMock: Partial<ThemeService>;
  let wailsMock: Partial<Wails>;

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
    wailsMock = {
      isWails: false,
      getSettings: vi.fn().mockResolvedValue({}),
      setSetting: vi.fn().mockResolvedValue(undefined)
    };
    themeServiceMock = {
      theme: vi.fn().mockReturnValue('system') as any,
      isDark: vi.fn().mockReturnValue(false) as any,
      colorTheme: vi.fn().mockReturnValue('azure') as any,
      availableColorThemes: vi.fn().mockReturnValue(Object.values(COLOR_THEMES)) as any,
      setTheme: vi.fn(),
      setColorTheme: vi.fn()
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

  it('should set theme when setTheme is called', () => {
    component.setTheme('dark');
    expect(themeServiceMock.setTheme).toHaveBeenCalledWith('dark');
  });

  it('should set color theme when setColorTheme is called', () => {
    const tealTheme = COLOR_THEMES.teal;
    component.setColorTheme(tealTheme);
    expect(themeServiceMock.setColorTheme).toHaveBeenCalledWith('teal');
  });

  it('should check if color theme is selected', () => {
    const azureTheme = COLOR_THEMES.azure;
    const result = component.isColorThemeSelected(azureTheme);
    expect(result).toBe(true);
  });

  it('should return false for non-selected theme', () => {
    const purpleTheme = COLOR_THEMES.purple;
    const result = component.isColorThemeSelected(purpleTheme);
    expect(result).toBe(false);
  });
});
