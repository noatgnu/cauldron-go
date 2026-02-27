import { Injectable, signal, effect } from '@angular/core';

export type Theme = 'light' | 'dark' | 'system';

@Injectable({
  providedIn: 'root'
})
export class ThemeService {
  theme = signal<Theme>('system');
  isDark = signal<boolean>(false);

  constructor() {
    this.loadFromLocalStorage();
    this.setupEffects();
    this.initializeSystemThemeListener();
  }

  private loadFromLocalStorage(): void {
    const savedTheme = localStorage.getItem('spa-theme') as Theme || 'system';
    this.theme.set(savedTheme);
    this.applyTheme(savedTheme);
  }

  private setupEffects(): void {
    effect(() => {
      const theme = this.theme();
      this.applyTheme(theme);
      localStorage.setItem('spa-theme', theme);
    });
  }

  setTheme(theme: Theme): void {
    this.theme.set(theme);
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

  private applyTheme(theme: Theme): void {
    const isDark = this.resolveTheme(theme);
    this.isDark.set(isDark);

    document.body.classList.remove('light-theme', 'dark-theme');
    document.body.classList.add(isDark ? 'dark-theme' : 'light-theme');
    document.body.setAttribute('data-theme', isDark ? 'dark' : 'light');
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

  getThemeIcon(): string {
    const theme = this.theme();
    if (theme === 'light') return 'light_mode';
    if (theme === 'dark') return 'dark_mode';
    return 'brightness_auto';
  }

  getThemeLabel(): string {
    const theme = this.theme();
    if (theme === 'light') return 'Light mode';
    if (theme === 'dark') return 'Dark mode';
    return 'System theme';
  }
}
