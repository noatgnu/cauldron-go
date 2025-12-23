import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { filter } from 'rxjs/operators';
import { PluginV2Service } from '../../core/services/plugin-v2';

interface Breadcrumb {
  label: string;
  url: string;
}

@Component({
  selector: 'app-breadcrumbs',
  imports: [CommonModule, MatIconModule],
  templateUrl: './breadcrumbs.html',
  styleUrl: './breadcrumbs.scss'
})
export class Breadcrumbs implements OnInit {
  protected breadcrumbs = signal<Breadcrumb[]>([]);

  private routeLabels: { [key: string]: string } = {
    '': 'Home',
    'settings': 'Settings',
    'jobs': 'Jobs',
    'plugin': 'Plugins',
    'plugin-list': 'Plugin List',
    'plugins': 'Plugin Management',
    'analysis': 'Analysis',
    'pca': 'PCA',
    'imputation': 'Imputation',
    'normalization': 'Normalization',
    'limma': 'Limma',
    'phate': 'PHATE',
    'fuzzy-clustering': 'Fuzzy Clustering',
    'alphastats': 'AlphaStats',
    'qfeatures-limma': 'QFeatures + Limma',
    'utilities': 'Utilities',
    'uniprot': 'UniProt Lookup',
    'coverage-map': 'Coverage Map',
    'ptm-remap': 'PTM Remapping',
    'peptide-check': 'Peptide Library Check',
    'format-conversion': 'Format Conversion'
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
      currentUrl += `/${path}`;

      let label = this.routeLabels[path] || path;
      let breadcrumbUrl = currentUrl;

      if (path === 'plugin' && i + 1 < paths.length && !isNaN(parseInt(paths[i + 1], 10))) {
        breadcrumbUrl = '/plugin-list';
      }

      if (paths[i - 1] === 'plugin' && !isNaN(parseInt(path, 10))) {
        try {
          const pluginId = parseInt(path, 10);
          const plugin = await this.pluginService.getPlugin(pluginId);
          label = plugin.definition.plugin.name;
        } catch (err) {
          label = `Plugin ${path}`;
        }
      }

      crumbs.push({ label, url: breadcrumbUrl });
    }

    this.breadcrumbs.set(crumbs);
  }

  navigate(url: string) {
    this.router.navigate([url]);
  }
}
