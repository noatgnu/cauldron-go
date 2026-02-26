import { Component, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Router } from '@angular/router';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { InstallPluginDialog, InstallPluginResult } from '../../components/install-plugin-dialog/install-plugin-dialog';
import { PluginInstallProgress } from '../../components/plugin-install-progress/plugin-install-progress';
import { ConfirmPluginInstallDialog, PluginInstallConfirmResult } from '../../components/confirm-plugin-install-dialog/confirm-plugin-install-dialog';
import {
  RegistryPlugin,
  RegistryPluginListResponse,
  RegistryCategory,
  RegistryCategoryListResponse
} from '../../core/models/registry';

@Component({
  selector: 'app-plugin-registry',
  imports: [
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatSelectModule,
    MatPaginatorModule,
    MatDialogModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './plugin-registry.html',
  styleUrl: './plugin-registry.scss',
})
export class PluginRegistry implements OnInit {
  protected plugins = signal<RegistryPlugin[]>([]);
  protected categories = signal<RegistryCategory[]>([]);
  protected loading = signal(false);
  protected loadingCategories = signal(false);
  protected totalCount = signal(0);
  protected pageSize = 10;
  protected pageIndex = 0;
  protected installedPluginRepos = signal<Set<string>>(new Set());

  searchQuery = '';
  selectedCategory = '';

  displayedColumns: string[] = ['name', 'version', 'category', 'author', 'updated', 'actions'];

  constructor(
    private wails: Wails,
    private dialog: MatDialog,
    private notification: NotificationService,
    private router: Router,
    private pluginService: PluginV2Service
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadInstalledPlugins();
    await this.loadCategories();
    await this.loadPlugins();
  }

  async loadInstalledPlugins(): Promise<void> {
    try {
      const installedPlugins = await this.pluginService.getAllPlugins();
      const repos = new Set<string>();

      for (const plugin of installedPlugins) {
        if (plugin.repository) {
          repos.add(this.normalizeRepoUrl(plugin.repository));
        }
      }

      this.installedPluginRepos.set(repos);
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Failed to load installed plugins: ${error}`);
    }
  }

  private normalizeRepoUrl(url: string): string {
    return url.toLowerCase().replace(/\.git$/, '').replace(/\/$/, '');
  }

  isPluginInstalled(plugin: RegistryPlugin): boolean {
    if (!plugin.repository) return false;
    return this.installedPluginRepos().has(this.normalizeRepoUrl(plugin.repository));
  }

  async loadCategories(): Promise<void> {
    this.loadingCategories.set(true);
    try {
      const response = await this.wails.listRegistryCategories() as RegistryCategoryListResponse;
      this.categories.set(response.results || []);
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Failed to load categories: ${error}`);
      this.notification.showError('Failed to load categories from registry');
    } finally {
      this.loadingCategories.set(false);
    }
  }

  async loadPlugins(): Promise<void> {
    this.loading.set(true);
    try {
      const offset = this.pageIndex * this.pageSize;
      const response = await this.wails.listRegistryPlugins(
        this.searchQuery,
        this.selectedCategory,
        '',
        this.pageSize,
        offset
      ) as RegistryPluginListResponse;

      this.plugins.set(response.results || []);
      this.totalCount.set(response.count || 0);
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Failed to load plugins: ${error}`);
      this.notification.showError('Failed to load plugins from registry. Please check your registry URL in settings.');
      this.plugins.set([]);
      this.totalCount.set(0);
    } finally {
      this.loading.set(false);
    }
  }

  async onSearch(): Promise<void> {
    this.pageIndex = 0;
    await this.loadPlugins();
  }

  async onCategoryChange(): Promise<void> {
    this.pageIndex = 0;
    await this.loadPlugins();
  }

  async onPageChange(event: PageEvent): Promise<void> {
    this.pageIndex = event.pageIndex;
    this.pageSize = event.pageSize;
    await this.loadPlugins();
  }

  async refreshRegistry(): Promise<void> {
    this.notification.showInfo('Refreshing registry data...');
    await this.loadCategories();
    await this.loadPlugins();
    await this.loadInstalledPlugins();
    this.notification.showSuccess('Registry data refreshed successfully');
  }

  openManualInstallDialog(): void {
    const dialogRef = this.dialog.open(InstallPluginDialog, {
      width: '600px',
      disableClose: true
    });

    dialogRef.afterClosed().subscribe((result: InstallPluginResult) => {
      if (result && result.repoURL) {
        this.dialog.open(PluginInstallProgress, {
          data: {
            repoURL: result.repoURL,
            commitHash: result.commitHash,
            sshKeyPath: result.sshKeyPath,
            passphrase: result.passphrase
          },
          disableClose: true,
          width: '500px'
        });
      }
    });
  }

  async installPlugin(plugin: RegistryPlugin): Promise<void> {
    try {
      await this.wails.logToFile('========================================');
      await this.wails.logToFile(`[PluginRegistry] Install button clicked for: ${plugin.name}`);

      if (!plugin.repository) {
        this.notification.showError('This plugin does not have a repository URL');
        return;
      }

      await this.wails.logToFile(`[PluginRegistry] Plugin name: ${plugin.name}, repository: ${plugin.repository}`);
      await this.wails.logToFile(`[PluginRegistry] Fetching plugin dependencies for: ${plugin.name}`);

      let hasPythonDeps = false;
      let hasRDeps = false;
      let runtimeEnvironments: string[] = [];

      try {
        const deps = await this.wails.fetchPluginDependencies(plugin.repository);
        await this.wails.logToFile(`[PluginRegistry] Raw deps response: ${JSON.stringify(deps)}`);
        hasPythonDeps = deps['hasPythonDeps'] === true;
        hasRDeps = deps['hasRDeps'] === true;
        runtimeEnvironments = deps['runtimeEnvironments'] || [];
        await this.wails.logToFile(`[PluginRegistry] After assignment - Python: ${hasPythonDeps}, R: ${hasRDeps}, envs: ${runtimeEnvironments.join(',')}`);
      } catch (error) {
        await this.wails.logToFile(`[PluginRegistry] Failed to fetch dependencies: ${error}`);
        if (plugin.runtime?.environments) {
          runtimeEnvironments = plugin.runtime.environments;
        }
      }

      await this.wails.logToFile(`[PluginRegistry] Opening install dialog for: ${plugin.name}`);
      await this.wails.logToFile(`[PluginRegistry] Dialog data - hasPythonDeps: ${hasPythonDeps}, hasRDeps: ${hasRDeps}, runtimeEnvs: ${runtimeEnvironments.join(', ')}`);

      const dialogRef = this.dialog.open(ConfirmPluginInstallDialog, {
        width: '600px',
        disableClose: true,
        data: {
          repo: plugin.repository,
          ref: plugin.commit_hash,
          name: plugin.name,
          id: plugin.id,
          version: plugin.version,
          author: plugin.author?.name || 'Unknown',
          description: plugin.description,
          category: plugin.category?.name || 'Uncategorized',
          requiresAuthentication: plugin.requires_authentication,
          runtimeEnvironments,
          hasPythonDeps,
          hasRDeps
        }
      });

      dialogRef.afterClosed().subscribe((result: PluginInstallConfirmResult) => {
        if (result && result.confirmed) {
          const progressDialogRef = this.dialog.open(PluginInstallProgress, {
            data: {
              repoURL: plugin.repository,
              commitHash: plugin.commit_hash,
              sshKeyPath: result.sshKeyPath,
              passphrase: result.passphrase,
              createVenv: result.createVenv,
              basePythonPath: result.basePythonPath,
              createRenv: result.createRenv,
              renvName: result.renvName
            },
            disableClose: true,
            width: '500px'
          });

          progressDialogRef.afterClosed().subscribe(async () => {
            await this.loadInstalledPlugins();
          });
        }
      });
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Error in installPlugin: ${error}`);
      this.notification.showError('Failed to open install dialog');
    }
  }

  viewDetails(plugin: RegistryPlugin): void {
    this.router.navigate(['/plugin-registry', plugin.id]);
  }

  formatDate(dateString: string): string {
    if (!dateString) {
      return 'N/A';
    }
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) {
        return 'N/A';
      }
      return date.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
      });
    } catch {
      return 'N/A';
    }
  }

  getAuthorName(plugin: RegistryPlugin): string {
    return plugin.author?.name || 'Unknown';
  }

  getCategoryName(plugin: RegistryPlugin): string {
    return plugin.category?.name || 'Uncategorized';
  }

  isIconUrl(icon: string): boolean {
    return icon.startsWith('http://') ||
           icon.startsWith('https://') ||
           icon.startsWith('/') ||
           icon.includes('.');
  }

  getRuntimeEnvironments(plugin: RegistryPlugin): string[] {
    if (!plugin.runtime?.environments) {
      return [];
    }

    return plugin.runtime.environments.map(env =>
      env.charAt(0).toUpperCase() + env.slice(1)
    );
  }

  getRuntimeIcon(env: string): string {
    switch (env.toLowerCase()) {
      case 'python':
        return 'adb';
      case 'r':
        return 'functions';
      case 'julia':
        return 'code';
      case 'node':
        return 'javascript';
      case 'docker':
        return 'dock';
      default:
        return 'terminal';
    }
  }
}
