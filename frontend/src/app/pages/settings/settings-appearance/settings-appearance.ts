import { Component } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatIconModule } from '@angular/material/icon';
import { ThemeService, Theme } from '../../../core/services/theme.service';

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
}
