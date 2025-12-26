import { Component, OnInit, signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { SettingsPython } from './settings-python/settings-python';
import { SettingsR } from './settings-r/settings-r';
import { SettingsRegistry } from './settings-registry/settings-registry';
import { SettingsGeneral } from './settings-general/settings-general';
import { SettingsEnv } from './settings-env/settings-env';

type SettingsSection = 'general' | 'python' | 'r' | 'registry' | 'env';

@Component({
  selector: 'app-settings',
  imports: [
    MatIconModule,
    SettingsPython,
    SettingsR,
    SettingsRegistry,
    SettingsGeneral,
    SettingsEnv
  ],
  templateUrl: './settings.html',
  styleUrl: './settings.scss',
})
export class Settings implements OnInit {
  protected currentSection = signal<SettingsSection>('general');

  constructor(private route: ActivatedRoute) {}

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
      case 'python': return 'Python Environment';
      case 'r': return 'R Environment';
      case 'registry': return 'Plugin Registry';
      case 'env': return 'Environment Variables';
      default: return 'Settings';
    }
  }

  getSectionIcon(): string {
    switch (this.currentSection()) {
      case 'general': return 'settings';
      case 'python': return 'language';
      case 'r': return 'analytics';
      case 'registry': return 'cloud';
      case 'env': return 'tune';
      default: return 'settings';
    }
  }
}
