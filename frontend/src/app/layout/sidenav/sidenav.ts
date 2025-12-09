import { Component, output, signal, computed, OnInit } from '@angular/core';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';

interface NavItem {
  label: string;
  icon: string;
  route?: string;
  children?: NavItem[];
}

@Component({
  selector: 'app-sidenav',
  imports: [
    MatListModule,
    MatIconModule,
    MatExpansionModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatCheckboxModule,
    MatDividerModule,
    FormsModule
  ],
  templateUrl: './sidenav.html',
  styleUrl: './sidenav.scss',
})
export class Sidenav implements OnInit {
  navigationClose = output<void>();
  searchQuery = signal<string>('');
  showPlugins = signal<boolean>(false);
  plugins = signal<models.PluginV2[]>([]);

  hardcodedNavItems: NavItem[] = [
    { label: 'Home', icon: 'home', route: '/' },
    {
      label: 'Data Transformation',
      icon: 'transform',
      children: [
        { label: 'Imputation', icon: 'auto_fix_high', route: '/analysis/imputation' },
        { label: 'Normalization', icon: 'tune', route: '/analysis/normalization' },
        { label: 'MaxLFQ Normalization', icon: 'science', route: '/analysis/maxlfq' },
        { label: 'Batch Correction', icon: 'auto_awesome', route: '/analysis/batch-correction' }
      ]
    },
    {
      label: 'Dimensionality Reduction',
      icon: 'compress',
      children: [
        { label: 'PCA', icon: 'scatter_plot', route: '/analysis/pca' },
        { label: 'PHATE', icon: 'bubble_chart', route: '/analysis/phate' },
        { label: 'Fuzzy Clustering', icon: 'grain', route: '/analysis/fuzzy-clustering' }
      ]
    },
    {
      label: 'Differential Analysis',
      icon: 'analytics',
      children: [
        { label: 'Limma', icon: 'show_chart', route: '/analysis/limma' },
        { label: 'QFeatures + Limma', icon: 'multiline_chart', route: '/analysis/qfeatures-limma' },
        { label: 'AlphaStats', icon: 'insert_chart', route: '/analysis/alphastats' }
      ]
    },
    {
      label: 'Visualization',
      icon: 'insert_chart_outlined',
      children: [
        { label: 'Correlation Matrix', icon: 'grid_on', route: '/analysis/correlation-matrix' },
        { label: 'Venn Diagram', icon: 'donut_small', route: '/analysis/venn-diagram' },
        { label: 'Violin Plot', icon: 'insights', route: '/analysis/violin-plot' }
      ]
    },
    {
      label: 'Utilities',
      icon: 'build',
      children: [
        { label: 'UniProt Lookup', icon: 'search', route: '/utilities/uniprot' },
        { label: 'Coverage Map', icon: 'map', route: '/utilities/coverage-map' },
        { label: 'PTM Remapping', icon: 'swap_horiz', route: '/utilities/ptm-remap' },
        { label: 'Peptide Library Check', icon: 'check_circle', route: '/utilities/peptide-check' },
        { label: 'Format Conversion', icon: 'sync_alt', route: '/utilities/format-conversion' }
      ]
    }
  ];

  categoryIcons: Record<string, string> = {
    'analysis': 'analytics',
    'visualization': 'insert_chart_outlined',
    'preprocessing': 'transform',
    'utilities': 'build'
  };

  pluginNavItems = computed(() => {
    const navItems: NavItem[] = [{ label: 'Home', icon: 'home', route: '/' }];
    const categoryMap = new Map<string, models.PluginV2[]>();

    for (const plugin of this.plugins()) {
      const category = plugin.definition.plugin.category || 'uncategorized';
      if (!categoryMap.has(category)) {
        categoryMap.set(category, []);
      }
      categoryMap.get(category)!.push(plugin);
    }

    for (const [category, pluginList] of categoryMap) {
      const children: NavItem[] = pluginList.map(plugin => ({
        label: plugin.definition.plugin.name,
        icon: plugin.definition.plugin.icon || 'extension',
        route: `/plugin/${plugin.definition.plugin.id}`
      }));

      navItems.push({
        label: this.formatCategoryLabel(category),
        icon: this.categoryIcons[category] || 'folder',
        children
      });
    }

    return navItems;
  });

  filteredNavItems = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();
    const sourceItems = this.showPlugins() ? this.pluginNavItems() : this.hardcodedNavItems;

    if (!query) {
      return sourceItems;
    }

    return sourceItems.map(item => {
      if (item.children) {
        const filteredChildren = item.children.filter(child =>
          child.label.toLowerCase().includes(query) ||
          item.label.toLowerCase().includes(query)
        );

        if (filteredChildren.length > 0) {
          return { ...item, children: filteredChildren };
        }

        return null;
      } else {
        if (item.label.toLowerCase().includes(query)) {
          return item;
        }
        return null;
      }
    }).filter(item => item !== null) as NavItem[];
  });

  constructor(
    private router: Router,
    private pluginService: PluginV2Service
  ) {}

  async ngOnInit() {
    await this.loadPlugins();
  }

  async loadPlugins() {
    try {
      const plugins = await this.pluginService.getAllPlugins();
      this.plugins.set(plugins);
    } catch (error) {
      console.error('Failed to load plugins:', error);
    }
  }

  togglePluginView() {
    this.showPlugins.update(v => !v);
    this.clearSearch();
  }

  formatCategoryLabel(category: string): string {
    return category.split('-').map(word =>
      word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
  }

  navigate(route: string): void {
    this.router.navigate([route]);
    this.navigationClose.emit();
  }

  updateSearch(value: string): void {
    this.searchQuery.set(value);
  }

  clearSearch(): void {
    this.searchQuery.set('');
  }
}
