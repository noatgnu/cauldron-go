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

interface RegistryPlugin {
  id: string;
  name: string;
  description: string;
  version: string;
  author: {
    id: number;
    name: string;
    email?: string;
  };
  category: {
    id: number;
    name: string;
    description?: string;
  };
  icon?: string;
  repository?: string;
  commit_hash?: string;
  requires_authentication: boolean;
  created_at: string;
  updated_at: string;
  tags?: Array<{
    id: number;
    name: string;
  }>;
  runtime?: {
    type: string;
    script: string;
  };
  inputs?: any[];
  outputs?: any[];
  env_variables?: any[];
}

interface RegistryPluginListResponse {
  count: number;
  next: string | null;
  previous: string | null;
  results: RegistryPlugin[];
}

interface Category {
  id: number;
  name: string;
  description?: string;
}

interface CategoryListResponse {
  count: number;
  next: string | null;
  previous: string | null;
  results: Category[];
}

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
  protected categories = signal<Category[]>([]);
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
      const response = await this.wails.listRegistryCategories() as CategoryListResponse;
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
      await this.wails.logToFile(`[PluginRegistry] Install button clicked for: ${plugin.name}`);

      if (!plugin.repository) {
        this.notification.showError('This plugin does not have a repository URL');
        return;
      }

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
          requiresAuthentication: plugin.requires_authentication
        }
      });

      dialogRef.afterClosed().subscribe((result: PluginInstallConfirmResult) => {
        if (result && result.confirmed) {
          const progressDialogRef = this.dialog.open(PluginInstallProgress, {
            data: {
              repoURL: plugin.repository,
              commitHash: plugin.commit_hash,
              sshKeyPath: result.sshKeyPath,
              passphrase: result.passphrase
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
}
