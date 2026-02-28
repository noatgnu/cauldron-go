import { Injectable, signal, computed, effect, inject } from '@angular/core';
import { Wails } from './wails';
import { getColorPalette, COLOR_PALETTES } from '../constants/color-palettes';
import { ColorTheme, ThemeDefinition, COLOR_THEMES, getColorTheme } from '../constants/color-themes';

export type Theme = 'light' | 'dark' | 'system';
export type FontScale = '75' | '100' | '125' | '150' | '200';
export type ColorblindPalette = 'default' | 'deuteranopia' | 'protanopia' | 'tritanopia' | 'monochrome' | 'highContrast';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  private readonly wails = inject(Wails);

  theme = signal<Theme>('system');
  isDark = signal<boolean>(false);
  colorTheme = signal<ColorTheme>('azure');

  fontScale = signal<FontScale>('100');
  highContrast = signal<boolean>(false);
  reducedMotion = signal<boolean>(false);
  colorblindPalette = signal<ColorblindPalette>('default');

  colorPalette = computed(() => {
    return getColorPalette(this.colorblindPalette());
  });

  availablePalettes = computed(() => {
    return Object.values(COLOR_PALETTES);
  });

  availableColorThemes = computed(() => {
    return Object.values(COLOR_THEMES);
  });

  currentColorTheme = computed((): ThemeDefinition => {
    return getColorTheme(this.colorTheme());
  });

  constructor() {
    this.initializeSettings();
    this.setupEffects();
    this.initializeSystemThemeListener();
    this.initializeReducedMotionListener();
  }

  private async initializeSettings(): Promise<void> {
    if (this.wails.isWails) {
      try {
        const config = await this.wails.getSettings();

        const savedTheme = localStorage.getItem('app-theme') as Theme;
        if (savedTheme) {
          this.theme.set(savedTheme);
        }

        const savedColorTheme = localStorage.getItem('app-color-theme') as ColorTheme;
        if (savedColorTheme && COLOR_THEMES[savedColorTheme]) {
          this.colorTheme.set(savedColorTheme);
        }

        if (config.accessibilityFontScale) {
          this.fontScale.set(config.accessibilityFontScale as FontScale);
        }
        if (config.accessibilityColorblindPalette) {
          this.colorblindPalette.set(config.accessibilityColorblindPalette as ColorblindPalette);
        }
        this.highContrast.set(config.accessibilityHighContrast || false);
        this.reducedMotion.set(config.accessibilityReducedMotion || false);
      } catch (error) {
        this.loadFromLocalStorage();
      }
    } else {
      this.loadFromLocalStorage();
    }
  }

  private loadFromLocalStorage(): void {
    const savedTheme = localStorage.getItem('app-theme') as Theme || 'system';
    this.theme.set(savedTheme);

    const savedColorTheme = localStorage.getItem('app-color-theme') as ColorTheme;
    if (savedColorTheme && COLOR_THEMES[savedColorTheme]) {
      this.colorTheme.set(savedColorTheme);
    }

    const savedFontScale = localStorage.getItem('accessibility.fontScale') as FontScale;
    if (savedFontScale) {
      this.fontScale.set(savedFontScale);
    }

    const savedPalette = localStorage.getItem('accessibility.colorblindPalette') as ColorblindPalette;
    if (savedPalette) {
      this.colorblindPalette.set(savedPalette);
    }

    const savedHighContrast = localStorage.getItem('accessibility.highContrast');
    if (savedHighContrast) {
      this.highContrast.set(savedHighContrast === 'true');
    }

    const savedReducedMotion = localStorage.getItem('accessibility.reducedMotion');
    if (savedReducedMotion) {
      this.reducedMotion.set(savedReducedMotion === 'true');
    }
  }

  private setupEffects(): void {
    effect(() => {
      const theme = this.theme();
      this.applyTheme(theme);
      localStorage.setItem('app-theme', theme);
    });

    effect(() => {
      const colorTheme = this.colorTheme();
      this.applyColorTheme(colorTheme);
      localStorage.setItem('app-color-theme', colorTheme);
      this.saveSetting('appearance.colorTheme', colorTheme);
    });

    effect(() => {
      const scale = this.fontScale();
      this.applyFontScale(scale);
      this.saveSetting('accessibility.fontScale', scale);
      localStorage.setItem('accessibility.fontScale', scale);
    });

    effect(() => {
      const contrast = this.highContrast();
      this.applyHighContrast(contrast);
      this.saveSetting('accessibility.highContrast', contrast);
      localStorage.setItem('accessibility.highContrast', String(contrast));
    });

    effect(() => {
      const motion = this.reducedMotion();
      this.applyReducedMotion(motion);
      this.saveSetting('accessibility.reducedMotion', motion);
      localStorage.setItem('accessibility.reducedMotion', String(motion));
    });

    effect(() => {
      const palette = this.colorblindPalette();
      this.saveSetting('accessibility.colorblindPalette', palette);
      localStorage.setItem('accessibility.colorblindPalette', palette);
    });
  }

  setTheme(theme: Theme): void {
    this.theme.set(theme);
  }

  setColorTheme(colorTheme: ColorTheme): void {
    this.colorTheme.set(colorTheme);
  }

  toggleTheme(): void {
    const current = this.theme();
    if (current === 'light') {
      this.setTheme('dark');
    } else if (current === 'dark') {
      this.setTheme('system');
    } else {
      this.setTheme('light');
    }
  }

  setFontScale(scale: FontScale): void {
    this.fontScale.set(scale);
  }

  setHighContrast(enabled: boolean): void {
    this.highContrast.set(enabled);
  }

  setReducedMotion(enabled: boolean): void {
    this.reducedMotion.set(enabled);
  }

  setColorblindPalette(palette: ColorblindPalette): void {
    this.colorblindPalette.set(palette);
  }

  resetAccessibilitySettings(): void {
    this.setFontScale('100');
    this.setHighContrast(false);
    this.setReducedMotion(false);
    this.setColorblindPalette('default');
  }

  private applyTheme(theme: Theme): void {
    const isDark = this.resolveTheme(theme);
    this.isDark.set(isDark);

    document.body.classList.remove('light-theme', 'dark-theme');
    document.body.classList.add(isDark ? 'dark-theme' : 'light-theme');
    document.body.setAttribute('data-theme', isDark ? 'dark' : 'light');
  }

  private applyColorTheme(colorTheme: ColorTheme): void {
    document.body.setAttribute('data-color-theme', colorTheme);
  }

  private applyFontScale(scale: FontScale): void {
    const rootElement = document.documentElement;
    rootElement.style.setProperty('--base-font-size', `${scale}%`);
  }

  private applyHighContrast(enabled: boolean): void {
    document.body.classList.toggle('high-contrast', enabled);
  }

  private applyReducedMotion(enabled: boolean): void {
    document.body.classList.toggle('reduced-motion', enabled);
  }

  private resolveTheme(theme: Theme): boolean {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches;
    }
    return theme === 'dark';
  }

  private initializeSystemThemeListener(): void {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

    mediaQuery.addEventListener('change', (e) => {
      if (this.theme() === 'system') {
        this.isDark.set(e.matches);
        document.body.classList.remove('light-theme', 'dark-theme');
        document.body.classList.add(e.matches ? 'dark-theme' : 'light-theme');
        document.body.setAttribute('data-theme', e.matches ? 'dark' : 'light');
      }
    });
  }

  private initializeReducedMotionListener(): void {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');

    if (mediaQuery.matches && !this.reducedMotion()) {
      this.applyReducedMotion(true);
    }

    mediaQuery.addEventListener('change', (e) => {
      if (e.matches && !this.reducedMotion()) {
        document.body.classList.add('reduced-motion');
      }
    });
  }

  private async saveSetting(key: string, value: unknown): Promise<void> {
    if (this.wails.isWails) {
      try {
        await this.wails.setSetting(key, value);
      } catch {
        // Silent fail for settings persistence
      }
    }
  }
}
