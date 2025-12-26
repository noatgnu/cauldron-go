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
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Wails } from '../../core/services/wails';
import { InstallPluginDialog } from '../../components/install-plugin-dialog/install-plugin-dialog';
import { PluginInstallProgress } from '../../components/plugin-install-progress/plugin-install-progress';

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
  created_at: string;
  updated_at: string;
  tags?: Array<{
    id: number;
    name: string;
  }>;
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
    MatPaginatorModule
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

  searchQuery = '';
  selectedCategory = '';

  constructor(
    private wails: Wails,
    private dialog: MatDialog,
    private snackBar: MatSnackBar
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadCategories();
    await this.loadPlugins();
  }

  async loadCategories(): Promise<void> {
    this.loadingCategories.set(true);
    try {
      const response = await this.wails.listRegistryCategories() as CategoryListResponse;
      this.categories.set(response.results || []);
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Failed to load categories: ${error}`);
      this.showError('Failed to load categories from registry');
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
      this.showError('Failed to load plugins from registry. Please check your registry URL in settings.');
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

    dialogRef.afterClosed().subscribe((result: { repoURL: string, commitHash?: string }) => {
      if (result && result.repoURL) {
        this.dialog.open(PluginInstallProgress, {
          data: {
            repoURL: result.repoURL,
            commitHash: result.commitHash
          },
          disableClose: true,
          width: '500px'
        });
      }
    });
  }

  async installPlugin(plugin: RegistryPlugin): Promise<void> {
    if (!plugin.repository) {
      this.showError('This plugin does not have a repository URL');
      return;
    }

    try {
      await this.wails.installPluginFromRegistry(plugin.id);
      this.snackBar.open(`Installing ${plugin.name}...`, 'Close', { duration: 3000 });
    } catch (error) {
      await this.wails.logToFile(`[PluginRegistry] Failed to install plugin ${plugin.id}: ${error}`);
      this.showError(`Failed to install plugin: ${error}`);
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

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }
}
