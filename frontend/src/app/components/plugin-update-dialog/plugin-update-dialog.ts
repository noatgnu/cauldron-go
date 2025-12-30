import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { SelectionModel } from '@angular/cdk/collections';
import { Wails } from '../../core/services/wails';
import { models } from '../../../wailsjs/go/models';

interface PluginUpdateInfo {
  plugin: models.PluginV2;
  currentCommit: string;
  recommendedCommit: string;
  latestCommit: string;
  changelogUrl?: string;
}

@Component({
  selector: 'app-plugin-update-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTableModule,
    MatCheckboxModule
  ],
  templateUrl: './plugin-update-dialog.html',
  styleUrl: './plugin-update-dialog.scss',
})
export class PluginUpdateDialog implements OnInit {
  checking = signal(true);
  updating = signal(false);
  availableUpdates = signal<PluginUpdateInfo[]>([]);
  selection = new SelectionModel<PluginUpdateInfo>(true, []);

  displayedColumns: string[] = ['select', 'name', 'current', 'recommended', 'latest', 'changelog'];

  constructor(
    private dialogRef: MatDialogRef<PluginUpdateDialog>,
    private wails: Wails
  ) {}

  async ngOnInit() {
    await this.checkForUpdates();
  }

  async checkForUpdates() {
    this.checking.set(true);

    try {
      const plugins = await this.wails.getPluginsV2();
      const remotePlugins = plugins.filter(p => p.installSource === 'remote' && p.repository);

      const updates: PluginUpdateInfo[] = [];

      for (const plugin of remotePlugins) {
        try {
          const updateInfo = await this.wails.checkPluginUpdate(
            plugin.repository!,
            plugin.commitHash || '',
            null
          );

          if (updateInfo.has_update) {
            updates.push({
              plugin,
              currentCommit: updateInfo.current_commit || plugin.commitHash || '',
              recommendedCommit: updateInfo.recommended_commit || updateInfo.latest_commit || '',
              latestCommit: updateInfo.latest_commit || '',
              changelogUrl: updateInfo.changelog_url
            });
          }
        } catch (err) {
          await this.wails.logToFile(`[PluginUpdateDialog] Failed to check update for ${plugin.definition.plugin.name}: ${err}`);
        }
      }

      this.availableUpdates.set(updates);

      if (updates.length > 0) {
        updates.forEach(update => this.selection.select(update));
      }

    } catch (err) {
      await this.wails.logToFile(`[PluginUpdateDialog] Failed to check for updates: ${err}`);
    } finally {
      this.checking.set(false);
    }
  }

  isAllSelected(): boolean {
    const numSelected = this.selection.selected.length;
    const numRows = this.availableUpdates().length;
    return numSelected === numRows && numRows > 0;
  }

  masterToggle() {
    if (this.isAllSelected()) {
      this.selection.clear();
    } else {
      this.availableUpdates().forEach(row => this.selection.select(row));
    }
  }

  getShortCommit(commit: string): string {
    return commit.substring(0, 7);
  }

  openChangelog(url: string) {
    if (url) {
      window.open(url, '_blank');
    }
  }

  async updateSelected() {
    const selected = this.selection.selected;
    if (selected.length === 0) return;

    this.updating.set(true);

    try {
      for (const update of selected) {
        try {
          await this.wails.updatePluginToCommit(
            update.plugin.repository!,
            update.recommendedCommit
          );
          await this.wails.logToFile(`[PluginUpdateDialog] Updated ${update.plugin.definition.plugin.name} to ${update.recommendedCommit}`);
        } catch (err) {
          await this.wails.logToFile(`[PluginUpdateDialog] Failed to update ${update.plugin.definition.plugin.name}: ${err}`);
        }
      }

      this.dialogRef.close({ updated: true, count: selected.length });

    } catch (err) {
      await this.wails.logToFile(`[PluginUpdateDialog] Update process failed: ${err}`);
    } finally {
      this.updating.set(false);
    }
  }

  close() {
    this.dialogRef.close({ updated: false });
  }
}
