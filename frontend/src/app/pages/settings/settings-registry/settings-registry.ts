import { Component, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Wails, Config } from '../../../core/services/wails';

@Component({
  selector: 'app-settings-registry',
  imports: [
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './settings-registry.html',
  styleUrl: './settings-registry.scss',
})
export class SettingsRegistry implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected saving = signal(false);
  pluginRegistryURL = '';

  constructor(
    private wails: Wails,
    private snackBar: MatSnackBar
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    this.pluginRegistryURL = this.config().pluginRegistryUrl || '';
  }

  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      await this.wails.logToFile(`[SettingsRegistry] Failed to load settings: ${error}`);
    }
  }

  async savePluginRegistryURL(): Promise<void> {
    try {
      this.saving.set(true);
      await this.saveSetting('pluginRegistryUrl', this.pluginRegistryURL);
      this.config.update(c => ({ ...c, pluginRegistryUrl: this.pluginRegistryURL }));
      this.snackBar.open('Registry URL saved successfully', 'Close', { duration: 3000 });
    } catch (error) {
      this.showError(`Failed to save registry URL: ${error}`);
    } finally {
      this.saving.set(false);
    }
  }

  private async saveSetting(key: string, value: any): Promise<void> {
    try {
      await this.wails.setSetting(key, value);
    } catch (error) {
      await this.wails.logToFile(`[SettingsRegistry] Failed to save setting ${key}: ${error}`);
      throw error;
    }
  }

  private showError(message: string): void {
    this.snackBar.open(message, 'Close', {
      duration: 5000,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['error-snackbar']
    });
  }
}
