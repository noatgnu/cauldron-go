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
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';

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
    MatProgressSpinnerModule
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

  categoryIcons: Record<string, string> = {
    'analysis': 'analytics',
    'visualization': 'bar_chart',
    'preprocessing': 'tune',
    'utilities': 'build'
  };

  constructor(
    private pluginService: PluginV2Service,
    private router: Router
  ) {}

  async ngOnInit() {
    await this.loadPlugins();
  }

  async loadPlugins() {
    try {
      this.loading.set(true);
      this.error.set('');
      const plugins = await this.pluginService.getAllPlugins();
      this.plugins.set(plugins);
      this.filteredPlugins.set(plugins);
    } catch (err) {
      this.error.set(`Failed to load plugins: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  onSearch() {
    if (!this.searchQuery.trim()) {
      this.filteredPlugins.set(this.plugins());
      return;
    }

    const results = this.pluginService.searchPlugins(this.plugins(), this.searchQuery);
    this.filteredPlugins.set(results);
  }

  getPluginsByCategory(): Map<string, models.PluginV2[]> {
    return this.pluginService.getPluginsByCategory(this.filteredPlugins());
  }

  getCategoryIcon(category: string): string {
    return this.categoryIcons[category] || 'extension';
  }

  navigateToPlugin(pluginId: string) {
    this.router.navigate(['/plugin', pluginId]);
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
}
