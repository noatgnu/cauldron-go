import { Component, inject, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatDividerModule } from '@angular/material/divider';
import { MatDialog } from '@angular/material/dialog';
import { Wails, BackupSummary, RestoreResult } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { ConfirmDialogComponent } from '../../../components/confirm-dialog/confirm-dialog';

@Component({
  selector: 'app-settings-backup',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatCheckboxModule,
    MatProgressSpinnerModule,
    MatDividerModule
  ],
  templateUrl: './settings-backup.html',
  styleUrl: './settings-backup.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SettingsBackup {
  private wails = inject(Wails);
  private notification = inject(NotificationService);
  private dialog = inject(MatDialog);

  includeSecrets = signal(false);
  backingUp = signal(false);
  restoring = signal(false);
  lastBackup = signal<BackupSummary | null>(null);
  lastRestore = signal<RestoreResult | null>(null);

  async backupNow() {
    const defaultName = `cauldron-backup-${this.timestampForFilename()}.json`;
    let path: string;
    try {
      path = await this.wails.saveFileDialog('Save Backup As', defaultName);
    } catch (error) {
      this.notification.showError('Failed to open save dialog');
      return;
    }
    if (!path) {
      return;
    }

    this.backingUp.set(true);
    this.lastRestore.set(null);
    try {
      const summary = await this.wails.createSettingsBackup(path, this.includeSecrets());
      this.lastBackup.set(summary);
      this.notification.showSuccess(`Backup saved to ${path}`);
    } catch (error) {
      this.notification.showError(`Backup failed: ${error}`);
    } finally {
      this.backingUp.set(false);
    }
  }

  async restoreFromFile() {
    let path: string;
    try {
      path = await this.wails.openFile('Select Backup File');
    } catch (error) {
      this.notification.showError('Failed to open file dialog');
      return;
    }
    if (!path) {
      return;
    }

    let summary: BackupSummary | null;
    try {
      summary = await this.wails.previewSettingsBackup(path);
    } catch (error) {
      this.notification.showError(`Failed to read backup file: ${error}`);
      return;
    }
    if (!summary) {
      this.notification.showError('Backup file could not be read');
      return;
    }

    const message =
      `This backup was created ${new Date(summary.createdAt).toLocaleString()} and contains ` +
      `${summary.settingsCount} setting(s) and ${summary.pluginsCount} plugin(s)` +
      (summary.includesSecrets ? ` and ${summary.envVarsCount} environment variable(s).` : '.') +
      ' Restoring will overwrite current settings and reinstall any missing plugins. Continue?';

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '500px',
      data: { title: 'Restore Backup', message, confirmText: 'Restore' }
    });

    dialogRef.afterClosed().subscribe(async (confirmed) => {
      if (!confirmed) {
        return;
      }
      await this.doRestore(path);
    });
  }

  private async doRestore(path: string) {
    this.restoring.set(true);
    this.lastBackup.set(null);
    try {
      const result = await this.wails.restoreSettingsBackup(path);
      this.lastRestore.set(result);
      const failedCount = result ? Object.keys(result.pluginsFailed || {}).length : 0;
      if (failedCount > 0) {
        this.notification.showWarning(`Restore completed with ${failedCount} plugin(s) failed`);
      } else {
        this.notification.showSuccess('Backup restored');
      }
    } catch (error) {
      this.notification.showError(`Restore failed: ${error}`);
    } finally {
      this.restoring.set(false);
    }
  }

  failedPluginEntries(result: RestoreResult): { id: string; message: string }[] {
    return Object.entries(result.pluginsFailed || {}).map(([id, message]) => ({ id, message: message ?? '' }));
  }

  private timestampForFilename(): string {
    const now = new Date();
    const pad = (n: number) => n.toString().padStart(2, '0');
    return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
  }
}
