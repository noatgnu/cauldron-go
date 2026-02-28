import { Component } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';
import { ThemeService, Theme } from '../../../core/services/theme.service';
import { ColorTheme, ThemeDefinition } from '../../../core/constants/color-themes';

@Component({
  selector: 'app-settings-appearance',
  imports: [
    MatCardModule,
    MatButtonToggleModule,
    MatIconModule
  ],
  templateUrl: './settings-appearance.html',
  styleUrl: './settings-appearance.scss',
})
export class SettingsAppearance {
  constructor(protected themeService: ThemeService) {}

  setTheme(theme: Theme): void {
    this.themeService.setTheme(theme);
  }

  setColorTheme(theme: ThemeDefinition): void {
    this.themeService.setColorTheme(theme.id);
  }

  isColorThemeSelected(theme: ThemeDefinition): boolean {
    return this.themeService.colorTheme() === theme.id;
  }
}
