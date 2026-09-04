import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatDialog } from '@angular/material/dialog';
import { FormsModule } from '@angular/forms';
import { Wails, Config } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import * as models from '../../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import { UpdateAvailableDialog } from '../../../components/update-available-dialog/update-available-dialog';

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
    MatSlideToggleModule,
    FormsModule
  ],
  templateUrl: './settings-general.html',
  styleUrl: './settings-general.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SettingsGeneral implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected debugMode = signal(false);
  protected autoCheckForUpdates = signal(true);
  protected checkingForUpdate = signal(false);
  protected forceUpdating = signal(false);
  protected forceUpdatingSingle = signal(false);
  protected remotePlugins = signal<models.PluginRegistry[]>([]);
  protected selectedPluginID: string | null = null;

  constructor(
    private wails: Wails,
    private notification: NotificationService,
    private dialog: MatDialog
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    await this.loadRemotePlugins();
  }

  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
      this.debugMode.set(!!config.debugMode);
      this.autoCheckForUpdates.set(config.autoCheckForUpdates !== false);
    } catch (error) {
      await this.wails.logToFile(`[SettingsGeneral] Failed to load settings: ${error}`);
    }
  }

  async setDebugMode(value: boolean): Promise<void> {
    this.debugMode.set(value);
    await this.saveSetting('debugMode', value);
  }

  async setAutoCheckForUpdates(value: boolean): Promise<void> {
    this.autoCheckForUpdates.set(value);
    await this.saveSetting('autoCheckForUpdates', value);
  }

  async checkForUpdateNow(): Promise<void> {
    this.checkingForUpdate.set(true);
    try {
      const info = await this.wails.checkForUpdate();
      if (info?.available) {
        this.dialog.open(UpdateAvailableDialog, { data: info, width: '480px' });
      } else {
        this.notification.showInfo("You're up to date.");
      }
    } catch (error) {
      this.notification.showError(`Failed to check for updates: ${error}`);
      await this.wails.logToFile(`[SettingsGeneral] Failed to check for updates: ${error}`);
    } finally {
      this.checkingForUpdate.set(false);
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
