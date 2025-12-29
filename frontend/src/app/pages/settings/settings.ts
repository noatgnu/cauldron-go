import { Component, OnInit, signal } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { SettingsPython } from './settings-python/settings-python';
import { SettingsR } from './settings-r/settings-r';
import { SettingsRegistry } from './settings-registry/settings-registry';
import { SettingsGeneral } from './settings-general/settings-general';
import { SettingsEnv } from './settings-env/settings-env';
import { SettingsGit } from './settings-git/settings-git';
import { SettingsAppearance } from './settings-appearance/settings-appearance';

type SettingsSection = 'general' | 'appearance' | 'python' | 'r' | 'registry' | 'env' | 'git';

@Component({
  selector: 'app-settings',
  imports: [
    MatIconModule,
    SettingsPython,
    SettingsR,
    SettingsRegistry,
    SettingsGeneral,
    SettingsEnv,
    SettingsGit,
    SettingsAppearance
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
      case 'appearance': return 'Appearance';
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
      case 'python': return 'language';
      case 'r': return 'analytics';
      case 'registry': return 'cloud';
      case 'env': return 'tune';
      case 'git': return 'key';
      default: return 'settings';
    }
  }
}
