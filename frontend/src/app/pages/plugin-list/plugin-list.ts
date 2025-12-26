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
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';
import { PluginEnvironmentDialog } from '../../components/plugin-environment-dialog/plugin-environment-dialog';
import { Wails } from '../../core/services/wails';

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
    MatBadgeModule
  ],
  templateUrl: './plugin-list.html',
  styleUrl: './plugin-list.scss',
})
export class PluginList implements OnInit {
  plugins = signal<models.PluginV2[]>([]);
  filteredPlugins = signal<models.PluginV2[]>([]);
  loading = signal(true);
  error = signal('');
  searchQuery = '';
  sourceFilter: 'all' | 'builtin' | 'remote' = 'all';
  pluginBindings = signal<Map<string, { python: boolean, r: boolean }>>(new Map());

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
    private wails: Wails
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
        runtimeType: plugin.definition.runtime.type,
        plugin: plugin
      }
    });
  }
}
