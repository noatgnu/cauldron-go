import { Component, output, signal, computed, OnInit, effect } from '@angular/core';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { Router, NavigationEnd } from '@angular/router';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { models } from '../../../wailsjs/go/models';
import { filter } from 'rxjs';

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
    MatDividerModule,
    FormsModule
  ],
  templateUrl: './sidenav.html',
  styleUrl: './sidenav.scss',
})
export class Sidenav implements OnInit {
  navigationClose = output<void>();
  searchQuery = signal<string>('');
  plugins = signal<models.PluginV2[]>([]);
  isSettingsRoute = signal<boolean>(false);

  categoryIcons: Record<string, string> = {
    'analysis': 'analytics',
    'visualization': 'insert_chart_outlined',
    'preprocessing': 'transform',
    'utilities': 'build',
    'statistics': 'functions',
    'data-transformation': 'transform',
    'dimensionality-reduction': 'compress',
    'differential-analysis': 'analytics'
  };

  pluginNavItems = computed(() => {
    const navItems: NavItem[] = [];
    const categoryMap = new Map<string, Map<string | null, models.PluginV2[]>>();

    for (const plugin of this.plugins()) {
      const category = plugin.definition.plugin.category || 'uncategorized';
      const subcategory = plugin.definition.plugin.subcategory || null;

      if (!categoryMap.has(category)) {
        categoryMap.set(category, new Map());
      }
      const subcategoryMap = categoryMap.get(category)!;

      if (!subcategoryMap.has(subcategory)) {
        subcategoryMap.set(subcategory, []);
      }
      subcategoryMap.get(subcategory)!.push(plugin);
    }

    const sortedCategories = Array.from(categoryMap.keys()).sort((a, b) => a.localeCompare(b));

    for (const category of sortedCategories) {
      const subcategoryMap = categoryMap.get(category)!;
      const sortedSubcategories = Array.from(subcategoryMap.keys()).sort((a, b) => {
        if (a === null) return 1;
        if (b === null) return -1;
        return a.localeCompare(b);
      });

      if (sortedSubcategories.length === 1 && sortedSubcategories[0] === null) {
        const pluginList = subcategoryMap.get(null)!;
        const sortedPlugins = [...pluginList].sort((a, b) =>
          a.definition.plugin.name.localeCompare(b.definition.plugin.name)
        );

        const children: NavItem[] = sortedPlugins.map(plugin => ({
          label: plugin.definition.plugin.name,
          icon: plugin.definition.plugin.icon || 'extension',
          route: `/plugin/${plugin.id}`
        }));

        navItems.push({
          label: this.formatCategoryLabel(category),
          icon: this.categoryIcons[category] || 'folder',
          children
        });
      } else {
        const subcategoryChildren: NavItem[] = [];

        for (const subcategory of sortedSubcategories) {
          const pluginList = subcategoryMap.get(subcategory)!;
          const sortedPlugins = [...pluginList].sort((a, b) =>
            a.definition.plugin.name.localeCompare(b.definition.plugin.name)
          );

          const pluginItems: NavItem[] = sortedPlugins.map(plugin => ({
            label: plugin.definition.plugin.name,
            icon: plugin.definition.plugin.icon || 'extension',
            route: `/plugin/${plugin.id}`
          }));

          if (subcategory === null) {
            subcategoryChildren.push(...pluginItems);
          } else {
            subcategoryChildren.push({
              label: this.formatCategoryLabel(subcategory),
              icon: 'folder_open',
              children: pluginItems
            });
          }
        }

        navItems.push({
          label: this.formatCategoryLabel(category),
          icon: this.categoryIcons[category] || 'folder',
          children: subcategoryChildren
        });
      }
    }

    return navItems;
  });

  settingsNavItems: NavItem[] = [
    { label: 'Back to Plugins', icon: 'arrow_back', route: '/home' },
    { label: 'General', icon: 'settings', route: '/settings/general' },
    { label: 'Appearance', icon: 'palette', route: '/settings/appearance' },
    { label: 'Python', icon: 'language', route: '/settings/python' },
    { label: 'R', icon: 'analytics', route: '/settings/r' },
    { label: 'Environment Variables', icon: 'tune', route: '/settings/env' },
    { label: 'Git Authentication', icon: 'key', route: '/settings/git' },
    { label: 'Plugin Registry', icon: 'cloud', route: '/settings/registry' }
  ];

  filteredNavItems = computed(() => {
    if (this.isSettingsRoute()) {
      return this.settingsNavItems;
    }

    const query = this.searchQuery().toLowerCase().trim();
    const staticItems: NavItem[] = [
      { label: 'Plugin Registry', icon: 'cloud', route: '/plugin-registry' }
    ];
    const sourceItems = this.pluginNavItems();
    const allItems = [...staticItems, ...sourceItems];

    if (!query) {
      return allItems;
    }

    const filteredStatic = staticItems.filter(item =>
      item.label.toLowerCase().includes(query)
    );

    const filteredPlugins = sourceItems.map(item => {
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

    return [...filteredStatic, ...filteredPlugins];
  });

  constructor(
    private router: Router,
    private pluginService: PluginV2Service
  ) {
    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe(() => {
      this.isSettingsRoute.set(this.router.url.startsWith('/settings'));
    });
    this.isSettingsRoute.set(this.router.url.startsWith('/settings'));
  }

  async ngOnInit() {
    await this.loadPlugins();

    if (window.runtime) {
      window.runtime.EventsOn('plugin:install:success', () => {
        this.loadPlugins();
      });

      window.runtime.EventsOn('plugin:uninstall:success', () => {
        this.loadPlugins();
      });
    }
  }

  async loadPlugins() {
    try {
      const plugins = await this.pluginService.getAllPlugins();
      this.plugins.set(plugins);
    } catch (error) {
      console.error('Failed to load plugins:', error);
    }
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
