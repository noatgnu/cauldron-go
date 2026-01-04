import { Component, inject } from '@angular/core';
import { MatSliderModule } from '@angular/material/slider';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { FormsModule } from '@angular/forms';
import { ThemeService, FontScale, ColorblindPalette } from '../../../core/services/theme.service';

@Component({
  selector: 'app-settings-accessibility',
  templateUrl: './settings-accessibility.html',
  styleUrls: ['./settings-accessibility.scss'],
  standalone: true,
  imports: [
    MatSliderModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatButtonModule,
    MatCardModule,
    MatIconModule,
    FormsModule
  ]
})
export class SettingsAccessibilityComponent {
  themeService = inject(ThemeService);

  fontScaleOptions: { value: FontScale; label: string }[] = [
    { value: '75', label: '75% (Smaller)' },
    { value: '100', label: '100% (Default)' },
    { value: '125', label: '125% (Larger)' },
    { value: '150', label: '150% (Extra Large)' },
    { value: '200', label: '200% (Maximum)' }
  ];

  get fontScaleValue(): number {
    return parseInt(this.themeService.fontScale());
  }

  set fontScaleValue(value: number) {
    this.themeService.setFontScale(String(value) as FontScale);
  }

  get highContrast(): boolean {
    return this.themeService.highContrast();
  }

  set highContrast(value: boolean) {
    this.themeService.setHighContrast(value);
  }

  get reducedMotion(): boolean {
    return this.themeService.reducedMotion();
  }

  set reducedMotion(value: boolean) {
    this.themeService.setReducedMotion(value);
  }

  get colorblindPalette(): ColorblindPalette {
    return this.themeService.colorblindPalette();
  }

  set colorblindPalette(value: ColorblindPalette) {
    this.themeService.setColorblindPalette(value);
  }

  get prefersReducedMotion(): boolean {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }

  formatLabel(value: number): string {
    return `${value}%`;
  }

  resetAll(): void {
    this.themeService.resetAccessibilitySettings();
  }
}
