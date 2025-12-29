import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatMenuModule } from '@angular/material/menu';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatBadgeModule } from '@angular/material/badge';
import { MatToolbarModule } from '@angular/material/toolbar';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { NotificationService } from '../../core/services/notification.service';
import { models } from '../../../wailsjs/go/models';
import { PluginEnvironmentDialog } from '../../components/plugin-environment-dialog/plugin-environment-dialog';
import { UninstallPluginDialog, UninstallPluginResult } from '../../components/uninstall-plugin-dialog/uninstall-plugin-dialog';
import { Wails } from '../../core/services/wails';

interface UpdateInfo {
  hasUpdate: boolean;
  currentCommit: string;
  latestCommit?: string;
  recommendedCommit?: string;
  changelogUrl?: string;
  checking?: boolean;
  error?: string;
}

@Component({
  selector: 'app-plugin-list',
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatDialogModule,
    MatMenuModule,
    MatButtonToggleModule,
    MatTooltipModule,
    MatBadgeModule,
    MatToolbarModule
  ],
  templateUrl: './plugin-list.html',
  styleUrl: './plugin-list.scss',
})

export class PluginList implements OnInit {
  plugins = signal<models.PluginV2[]>([]);
  filteredPlugins = signal<models.PluginV2[]>([]);
  loading = signal(true);
  updatingAll = signal(false);
  error = signal('');
  searchQuery = '';
  sourceFilter: 'all' | 'builtin' | 'remote' = 'all';
  pluginBindings = signal<Map<string, { python: boolean, r: boolean }>>(new Map());
  updateInfo = signal<Map<string, UpdateInfo>>(new Map());

  categoryIcons: Record<string, string> = {
    'analysis': 'analytics',
    'visualization': 'bar_chart',
    'preprocessing': 'tune',
    'utilities': 'build'
  };

  constructor(
    private pluginService: PluginV2Service,
    private router: Router,
    private dialog: MatDialog,
    private wails: Wails,
    private notification: NotificationService
  ) {}

  async ngOnInit() {
    await this.loadPlugins();

    this.wails.bindingsUpdated$.subscribe(() => {
      this.loadPluginBindings(this.plugins());
    });
  }

  async loadPlugins() {
    try {
      this.loading.set(true);
      this.error.set('');
      const plugins = await this.pluginService.getAllPlugins();
      this.plugins.set(plugins);
      this.filteredPlugins.set(plugins);
      await this.loadPluginBindings(plugins);
    } catch (err) {
      this.error.set(`Failed to load plugins: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadPluginBindings(plugins: models.PluginV2[]) {
    const bindingsMap = new Map<string, { python: boolean, r: boolean }>();

    for (const plugin of plugins) {
      const pluginId = plugin.id.toString();
      let hasPython = false;
      let hasR = false;

      try {
        const pythonBinding = await this.wails.getPluginEnvironmentBinding(pluginId, 'python');
        hasPython = !!pythonBinding;
      } catch {}

      try {
        const rBinding = await this.wails.getPluginEnvironmentBinding(pluginId, 'r');
        hasR = !!rBinding;
      } catch {}

      if (hasPython || hasR) {
        bindingsMap.set(pluginId, { python: hasPython, r: hasR });
      }
    }

    this.pluginBindings.set(bindingsMap);
  }

  hasCustomBinding(pluginId: number): boolean {
    return this.pluginBindings().has(pluginId.toString());
  }

  getBindingTooltip(pluginId: number): string {
    const binding = this.pluginBindings().get(pluginId.toString());
    if (!binding) return '';

    const parts: string[] = [];
    if (binding.python) parts.push('Python');
    if (binding.r) parts.push('R');

    return `Custom environment bound: ${parts.join(' & ')}`;
  }

  async reloadPlugins() {
    try {
      this.loading.set(true);
      this.error.set('');
      await this.pluginService.reloadPlugins();
      await this.loadPlugins();
    } catch (err) {
      this.error.set(`Failed to reload plugins: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  onSearch() {
    this.applyFilters();
  }

  onSourceFilterChange() {
    this.applyFilters();
  }

  private applyFilters() {
    let results = this.plugins();

    if (this.sourceFilter !== 'all') {
      results = results.filter(p => p.installSource === this.sourceFilter);
    }

    if (this.searchQuery.trim()) {
      results = this.pluginService.searchPlugins(results, this.searchQuery);
    }

    this.filteredPlugins.set(results);
  }

  getPluginsByCategory(): Map<string, models.PluginV2[]> {
    return this.pluginService.getPluginsByCategory(this.filteredPlugins());
  }

  getCategoryIcon(category: string): string {
    return this.categoryIcons[category] || 'extension';
  }

  navigateToPlugin(pluginId: number) {
    this.router.navigate(['/plugin', pluginId.toString()]);
  }

  getRuntimeIconFromPlugin(plugin: models.PluginV2): string {
    const envs = this.getPluginEnvironments(plugin);
    if (envs.length > 1) {
      return 'hub';
    }
    if (envs.includes('python')) {
      return 'code';
    }
    if (envs.includes('r')) {
      return 'analytics';
    }
    return 'terminal';
  }

  getRuntimeLabelFromPlugin(plugin: models.PluginV2): string {
    const envs = this.getPluginEnvironments(plugin);
    if (envs.length === 0) {
      return 'Direct';
    }
    return envs.map(e => e.charAt(0).toUpperCase() + e.slice(1)).join(' + ');
  }

  private getPluginEnvironments(plugin: models.PluginV2): string[] {
    return plugin.definition.runtime.environments || [];
  }

  getRuntimeIcon(runtime: string): string {
    switch (runtime) {
      case 'python':
        return 'code';
      case 'r':
        return 'analytics';
      case 'pythonWithR':
        return 'hub';
      default:
        return 'terminal';
    }
  }

  getRuntimeLabel(runtime: string): string {
    switch (runtime) {
      case 'python':
        return 'Python';
      case 'r':
        return 'R';
      case 'pythonWithR':
        return 'Python + R';
      default:
        return runtime;
    }
  }

  openEnvironmentDialog(event: Event, plugin: models.PluginV2) {
    event.stopPropagation();

    this.dialog.open(PluginEnvironmentDialog, {
      width: '600px',
      disableClose: true,
      data: {
        pluginId: plugin.id.toString(),
        pluginName: plugin.definition.plugin.name,
        runtimeEnvironments: plugin.definition.runtime.environments,
        plugin: plugin
      }
    });
  }

  async checkForUpdate(event: Event, plugin: models.PluginV2) {
    event.stopPropagation();

    if (plugin.installSource !== 'remote' || !plugin.repository) {
      return;
    }

    const key = plugin.id.toString();
    const currentMap = new Map(this.updateInfo());
    currentMap.set(key, {
      hasUpdate: false,
      currentCommit: plugin.commitHash || '',
      checking: true
    });
    this.updateInfo.set(currentMap);

    try {
      const result = await this.wails.checkPluginUpdate(
        plugin.repository,
        plugin.commitHash || '',
        null
      );

      const newMap = new Map(this.updateInfo());
      newMap.set(key, {
        hasUpdate: result.has_update || false,
        currentCommit: result.current_commit || plugin.commitHash || '',
        latestCommit: result.latest_commit,
        recommendedCommit: result.recommended_commit,
        changelogUrl: result.changelog_url,
        checking: false
      });
      this.updateInfo.set(newMap);
    } catch (err) {
      const errorMap = new Map(this.updateInfo());
      errorMap.set(key, {
        hasUpdate: false,
        currentCommit: plugin.commitHash || '',
        checking: false,
        error: String(err)
      });
      this.updateInfo.set(errorMap);
    }
  }

  hasUpdateInfo(pluginId: number): boolean {
    return this.updateInfo().has(pluginId.toString());
  }

  getUpdateInfo(pluginId: number): UpdateInfo | undefined {
    return this.updateInfo().get(pluginId.toString());
  }

  isCheckingUpdate(pluginId: number): boolean {
    const info = this.getUpdateInfo(pluginId);
    return info?.checking || false;
  }

  hasAvailableUpdate(pluginId: number): boolean {
    const info = this.getUpdateInfo(pluginId);
    return info?.hasUpdate || false;
  }

  async installUpdate(event: Event, plugin: models.PluginV2) {
    event.stopPropagation();

    const updateInfo = this.getUpdateInfo(plugin.id);
    if (!updateInfo || !updateInfo.hasUpdate || !plugin.repository) {
      return;
    }

    const targetCommit = updateInfo.recommendedCommit || updateInfo.latestCommit;
    if (!targetCommit) {
      this.notification.showWarning('No target commit available for update');
      return;
    }

    try {
      const shortCommit = targetCommit.substring(0, 7);
      this.notification.showInfo(`Updating ${plugin.definition.plugin.name} to ${shortCommit}...`, 2000);

      await this.wails.updatePluginToCommit(plugin.repository, targetCommit);

      this.notification.showSuccess('Plugin updated successfully! Reloading...', 2000);

      const updateMap = new Map(this.updateInfo());
      updateMap.delete(plugin.id.toString());
      this.updateInfo.set(updateMap);

      await this.loadPlugins();

    } catch (err) {
      this.notification.showError(`Update failed: ${err}`);
    }
  }

  async updateAllRemotePlugins() {
    this.updatingAll.set(true);

    try {
      this.notification.showInfo('Updating all external plugins...', 2000);

      await this.wails.updateAllRemotePlugins();

      this.notification.showSuccess('All external plugins updated successfully! Reloading...');

      this.updateInfo.set(new Map());

      await this.loadPlugins();

    } catch (err) {
      this.notification.showError(`Update failed: ${err}`);
    } finally {
      this.updatingAll.set(false);
    }
  }

  async uninstallPlugin(event: Event, plugin: models.PluginV2) {
    event.stopPropagation();

    if (plugin.installSource !== 'remote' || !plugin.repository) {
      this.notification.showWarning('Only externally installed plugins can be uninstalled');
      return;
    }

    const dialogRef = this.dialog.open(UninstallPluginDialog, {
      width: '600px',
      disableClose: true,
      data: {
        pluginId: plugin.definition.plugin.id,
        pluginName: plugin.definition.plugin.name,
        repositoryURL: plugin.repository
      }
    });

    dialogRef.afterClosed().subscribe(async (result: UninstallPluginResult) => {
      if (result && result.confirmed) {
        try {
          this.notification.showInfo(`Uninstalling ${plugin.definition.plugin.name}...`, 2000);

          await this.wails.uninstallPluginFromRepo(
            plugin.repository!,
            result.removeGitAuth,
            result.deleteJobHistory,
            result.deleteEnvironments
          );

          this.notification.showSuccess('Plugin uninstalled successfully! Reloading...');

          const updateMap = new Map(this.updateInfo());
          updateMap.delete(plugin.id.toString());
          this.updateInfo.set(updateMap);

          await this.loadPlugins();

        } catch (err) {
          this.notification.showError(`Uninstall failed: ${err}`);
        }
      }
    });
  }

}
