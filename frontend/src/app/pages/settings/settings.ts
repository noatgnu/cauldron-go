import { Component, OnInit, signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Location } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { SettingsPython } from './settings-python/settings-python';
import { SettingsR } from './settings-r/settings-r';
import { SettingsRegistry } from './settings-registry/settings-registry';
import { SettingsGeneral } from './settings-general/settings-general';
import { SettingsEnv } from './settings-env/settings-env';
import { SettingsGit } from './settings-git/settings-git';
import { SettingsAppearance } from './settings-appearance/settings-appearance';
import { SettingsAccessibilityComponent } from './settings-accessibility/settings-accessibility';

type SettingsSection = 'general' | 'appearance' | 'accessibility' | 'python' | 'r' | 'registry' | 'env' | 'git';

@Component({
  selector: 'app-settings',
  imports: [
    MatIconModule,
    MatButtonModule,
    SettingsPython,
    SettingsR,
    SettingsRegistry,
    SettingsGeneral,
    SettingsEnv,
    SettingsGit,
    SettingsAppearance,
    SettingsAccessibilityComponent
  ],
  templateUrl: './settings.html',
  styleUrl: './settings.scss',
})
export class Settings implements OnInit {
  protected currentSection = signal<SettingsSection>('general');

  constructor(
    private route: ActivatedRoute,
    private location: Location
  ) {}

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      const section = params['section'] as SettingsSection;
      if (section) {
        this.currentSection.set(section);
      }
    });
  }

  getSectionTitle(): string {
    switch (this.currentSection()) {
      case 'general': return 'General Settings';
      case 'appearance': return 'Appearance';
      case 'accessibility': return 'Accessibility';
      case 'python': return 'Python Environment';
      case 'r': return 'R Environment';
      case 'registry': return 'Plugin Registry';
      case 'env': return 'Environment Variables';
      case 'git': return 'Git Authentication';
      default: return 'Settings';
    }
  }

  getSectionIcon(): string {
    switch (this.currentSection()) {
      case 'general': return 'settings';
      case 'appearance': return 'palette';
      case 'accessibility': return 'accessibility_new';
      case 'python': return 'language';
      case 'r': return 'analytics';
      case 'registry': return 'cloud';
      case 'env': return 'tune';
      case 'git': return 'key';
      default: return 'settings';
    }
  }

  goBack(): void {
    this.location.back();
  }
}
