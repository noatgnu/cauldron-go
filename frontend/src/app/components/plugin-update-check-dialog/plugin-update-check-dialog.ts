import { Component, Inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { Wails } from '../../core/services/wails';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import { ConfirmDialogComponent } from '../confirm-dialog/confirm-dialog';
import { MatDialog } from '@angular/material/dialog';

export interface PluginUpdateCheckDialogData {
  plugin: models.PluginV2;
}

export interface PluginUpdateCheckDialogResult {
  updated: boolean;
}

interface UpdateCheckResult {
  hasUpdate: boolean;
  currentCommit: string;
  latestCommit?: string;
  recommendedCommit?: string;
  changelogUrl?: string;
  schemaMigrationAvailable?: boolean;
  error?: string;
}

@Component({
  selector: 'app-plugin-update-check-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatCardModule,
    MatChipsModule
  ],
  templateUrl: './plugin-update-check-dialog.html',
  styleUrl: './plugin-update-check-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PluginUpdateCheckDialog implements OnInit {
  checking = signal(true);
  updating = signal(false);
  updateResult = signal<UpdateCheckResult | null>(null);

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: PluginUpdateCheckDialogData,
    private dialogRef: MatDialogRef<PluginUpdateCheckDialog>,
    private dialog: MatDialog,
    private wails: Wails
  ) {}

  async ngOnInit() {
    await this.checkForUpdate();
  }

  async checkForUpdate() {
    this.checking.set(true);

    if (!this.data.plugin.repository || this.data.plugin.installSource !== 'remote') {
      this.updateResult.set({
        hasUpdate: false,
        currentCommit: this.data.plugin.commitHash || '',
        error: 'Plugin is not from a remote repository'
      });
      this.checking.set(false);
      return;
    }

    try {
      const result = await this.wails.checkPluginUpdate(
        this.data.plugin.repository,
        this.data.plugin.commitHash || '',
        null
      );

      this.updateResult.set({
        hasUpdate: result.has_update || false,
        currentCommit: result.current_commit || this.data.plugin.commitHash || '',
        latestCommit: result.latest_commit,
        recommendedCommit: result.recommended_commit,
        changelogUrl: result.changelog_url,
        schemaMigrationAvailable: result.schema_migration_available || false
      });

      await this.wails.logToFile(`[PluginUpdateCheckDialog] Checked update for ${this.data.plugin.definition.plugin.name}: hasUpdate=${result.has_update}`);

    } catch (err) {
      const errorMsg = String(err);
      this.updateResult.set({
        hasUpdate: false,
        currentCommit: this.data.plugin.commitHash || '',
        error: errorMsg
      });
      await this.wails.logToFile(`[PluginUpdateCheckDialog] Failed to check update: ${errorMsg}`);
    } finally {
      this.checking.set(false);
    }
  }

  getShortCommit(commit: string): string {
    return commit.substring(0, 7);
  }

  openChangelog() {
    const url = this.updateResult()?.changelogUrl;
    if (url) {
      window.open(url, '_blank');
    }
  }

  async proceedWithUpdate() {
    const result = this.updateResult();
    if (!result || !result.hasUpdate || !this.data.plugin.repository) {
      return;
    }

    const targetCommit = result.recommendedCommit || result.latestCommit;
    if (!targetCommit) {
      await this.wails.logToFile('[PluginUpdateCheckDialog] No target commit available');
      return;
    }

    this.updating.set(true);

    try {
      await this.wails.updatePluginToCommit(this.data.plugin.repository, targetCommit);
      await this.wails.logToFile(`[PluginUpdateCheckDialog] Updated ${this.data.plugin.definition.plugin.name} to ${targetCommit}`);

      this.dialogRef.close({ updated: true });

    } catch (err) {
      const errorMessage = String(err);

      if (errorMessage.includes('LOCAL_MODIFICATIONS')) {
        const confirmDialogRef = this.dialog.open(ConfirmDialogComponent, {
          width: '500px',
          data: {
            title: 'Local Modifications Detected',
            message: `Plugin "${this.data.plugin.definition.plugin.name}" has local modifications. Updating will discard all local changes. Do you want to continue?`,
            confirmText: 'Discard & Update',
            cancelText: 'Cancel'
          }
        });

        const confirmed = await confirmDialogRef.afterClosed().toPromise();
        if (confirmed) {
          try {
            await this.wails.updatePluginToCommitForce(this.data.plugin.repository, targetCommit, true);
            await this.wails.logToFile(`[PluginUpdateCheckDialog] Force updated ${this.data.plugin.definition.plugin.name}`);
            this.dialogRef.close({ updated: true });
          } catch (forceErr) {
            this.updateResult.set({
              ...result,
              error: `Failed to force update: ${forceErr}`
            });
            await this.wails.logToFile(`[PluginUpdateCheckDialog] Force update failed: ${forceErr}`);
          }
        }
      } else {
        this.updateResult.set({
          ...result,
          error: `Update failed: ${errorMessage}`
        });
        await this.wails.logToFile(`[PluginUpdateCheckDialog] Update failed: ${errorMessage}`);
      }
    } finally {
      this.updating.set(false);
    }
  }

  close() {
    this.dialogRef.close({ updated: false });
  }
}
