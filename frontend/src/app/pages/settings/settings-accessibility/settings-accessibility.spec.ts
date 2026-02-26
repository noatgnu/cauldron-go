import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { signal, computed } from '@angular/core';
import { SettingsAccessibilityComponent } from './settings-accessibility';
import { ThemeService } from '../../../core/services/theme.service';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('SettingsAccessibilityComponent', () => {
  let component: SettingsAccessibilityComponent;
  let fixture: ComponentFixture<SettingsAccessibilityComponent>;
  let mockThemeService: any;

  beforeEach(async () => {
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

    mockThemeService = {
      fontScale: signal('100'),
      highContrast: signal(false),
      reducedMotion: signal(false),
      colorblindPalette: signal('default'),
      colorPalette: computed(() => [] as string[]),
      availablePalettes: computed(() => [] as { name: string; colors: string[] }[]),
      setFontScale: vi.fn(),
      setHighContrast: vi.fn(),
      setReducedMotion: vi.fn(),
      setColorblindPalette: vi.fn(),
      resetAccessibilitySettings: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsAccessibilityComponent, NoopAnimationsModule],
      providers: [
        { provide: ThemeService, useValue: mockThemeService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(SettingsAccessibilityComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
