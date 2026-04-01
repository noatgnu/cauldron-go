import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSelectModule } from '@angular/material/select';
import { FormsModule } from '@angular/forms';
import { Wails, Config } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import * as models from '../../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

@Component({
  selector: 'app-settings-general',
  imports: [
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatSelectModule,
    FormsModule
  ],
  templateUrl: './settings-general.html',
  styleUrl: './settings-general.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SettingsGeneral implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected forceUpdating = signal(false);
  protected forceUpdatingSingle = signal(false);
  protected remotePlugins = signal<models.PluginRegistry[]>([]);
  protected selectedPluginID: string | null = null;

  constructor(
    private wails: Wails,
    private notification: NotificationService
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    await this.loadRemotePlugins();
  }

  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to load settings: ${error}`);
    }
  }

  async loadRemotePlugins(): Promise<void> {
    try {
      const plugins = await this.wails.getRemotePlugins();
      this.remotePlugins.set(plugins);
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to load remote plugins: ${error}`);
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

  async forceUpdateSinglePlugin(): Promise<void> {
    if (!this.selectedPluginID) {
      this.notification.showError('Please select a plugin to force update');
      return;
    }

    this.forceUpdatingSingle.set(true);

    try {
      const pluginName = this.remotePlugins().find(p => p.pluginId === this.selectedPluginID)?.name || this.selectedPluginID;
      this.notification.showInfo(`Force updating ${pluginName}...`, 2000);

      await this.wails.forceUpdateRemotePlugin(this.selectedPluginID);

      this.notification.showSuccess(`${pluginName} force updated successfully!`);
      await this.wails.logToFile(`[SettingsGeneral] Successfully force updated ${pluginName} (${this.selectedPluginID})`);

      this.selectedPluginID = null;
      await this.loadRemotePlugins();

    } catch (err) {
      this.notification.showError(`Force update failed: ${err}`);
      await this.wails.logToFile(`[SettingsGeneral] Force update of ${this.selectedPluginID} failed: ${err}`);
    } finally {
      this.forceUpdatingSingle.set(false);
    }
  }
}
