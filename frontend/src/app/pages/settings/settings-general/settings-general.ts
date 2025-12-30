import { Component, OnInit, signal } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Wails, Config } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';

@Component({
  selector: 'app-settings-general',
  imports: [
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule
  ],
  templateUrl: './settings-general.html',
  styleUrl: './settings-general.scss',
})
export class SettingsGeneral implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected forceUpdating = signal(false);

  constructor(
    private wails: Wails,
    private notification: NotificationService
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
  }

  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to load settings: ${error}`);
    }
  }

  async browseOutputDirectory(): Promise<void> {
    try {
      const path = await this.wails.openDirectoryDialog('Select Output Directory');
      if (path) {
        this.config.update(c => ({ ...c, outputDirectory: path }));
        await this.saveSetting('outputDirectory', path);
      }
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to browse for output directory: ${error}`);
    }
  }

  private async saveSetting(key: string, value: any): Promise<void> {
    try {
      await this.wails.setSetting(key, value);
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to save setting ${key}: ${error}`);
    }
  }

  async forceUpdateAllPlugins(): Promise<void> {
    this.forceUpdating.set(true);

    try {
      this.notification.showInfo('Force updating all external plugins to latest...', 2000);

      await this.wails.forceUpdateAllRemotePlugins();

      this.notification.showSuccess('All external plugins force updated successfully!');
      await this.wails.logToFile('[SettingsGeneral] Successfully force updated all external plugins');

    } catch (err) {
      this.notification.showError(`Force update failed: ${err}`);
      await this.wails.logToFile(`[SettingsGeneral] Force update failed: ${err}`);
    } finally {
      this.forceUpdating.set(false);
    }
  }
}
