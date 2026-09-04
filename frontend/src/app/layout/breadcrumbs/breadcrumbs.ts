import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { filter } from 'rxjs/operators';
import { PluginV2Service } from '../../core/services/plugin-v2';

interface Breadcrumb {
  label: string;
  url: string;
}

interface RouteConfig {
  label: string;
  listRoute?: string;
}

@Component({
  selector: 'app-breadcrumbs',
  imports: [CommonModule, MatIconModule],
  templateUrl: './breadcrumbs.html',
  styleUrl: './breadcrumbs.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Breadcrumbs implements OnInit {
  protected breadcrumbs = signal<Breadcrumb[]>([]);

  private routeConfig: { [key: string]: RouteConfig } = {
    '': { label: 'Home' },
    'settings': { label: 'Settings' },
    'jobs': { label: 'Jobs' },
    'job': { label: 'Jobs', listRoute: '/jobs' },
    'plugin': { label: 'Plugins', listRoute: '/plugin-list' },
    'plugin-list': { label: 'Plugin List' },
    'plugins': { label: 'Plugin Management' },
    'table-browser': { label: 'Table Browser' }
  };

  constructor(
    private router: Router,
    private activatedRoute: ActivatedRoute,
    private pluginService: PluginV2Service
  ) {}

  ngOnInit() {
    this.updateBreadcrumbs();

    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe(() => {
      this.updateBreadcrumbs();
    });
  }

  private async updateBreadcrumbs() {
    const url = this.router.url;
    const paths = url.split('/').filter(p => p);

    const crumbs: Breadcrumb[] = [
      { label: 'Home', url: '/' }
    ];

    let currentUrl = '';
    for (let i = 0; i < paths.length; i++) {
      const path = paths[i];
      const nextPath = i + 1 < paths.length ? paths[i + 1] : null;
      const prevPath = i > 0 ? paths[i - 1] : null;
      currentUrl += `/${path}`;

      const config = this.routeConfig[path];

      if (!config) {
        if (this.isUUID(path) || this.isNumeric(path)) {
          const label = await this.getDetailLabel(prevPath, path);
          crumbs.push({ label, url: currentUrl });
        } else {
          crumbs.push({ label: path, url: currentUrl });
        }
        continue;
      }

      if (config.listRoute && nextPath && (this.isUUID(nextPath) || this.isNumeric(nextPath))) {
        crumbs.push({ label: config.label, url: config.listRoute });
      } else {
        crumbs.push({ label: config.label, url: currentUrl });
      }
    }

    this.breadcrumbs.set(crumbs);
  }

  private isUUID(str: string): boolean {
    const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    return uuidPattern.test(str);
  }

  private isNumeric(str: string): boolean {
    return !isNaN(parseInt(str, 10)) && isFinite(Number(str));
  }

  private async getDetailLabel(parentRoute: string | null, id: string): Promise<string> {
    if (parentRoute === 'plugin' && this.isNumeric(id)) {
      try {
        const pluginId = parseInt(id, 10);
        const plugin = await this.pluginService.getPlugin(pluginId);
        return plugin.definition.plugin.name;
      } catch (err) {
        return `Plugin ${id}`;
      }
    }

    if (parentRoute === 'job' && this.isUUID(id)) {
      return `Job ${id.substring(0, 8)}...`;
    }

    return id;
  }

  navigate(url: string) {
    this.router.navigate([url]);
  }
}
