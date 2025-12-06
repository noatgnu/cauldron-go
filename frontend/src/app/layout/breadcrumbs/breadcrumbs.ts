import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { MatIconModule } from '@angular/material/icon';
import { filter } from 'rxjs/operators';

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
    private activatedRoute: ActivatedRoute
  ) {}

  ngOnInit() {
    this.updateBreadcrumbs();

    this.router.events.pipe(
      filter(event => event instanceof NavigationEnd)
    ).subscribe(() => {
      this.updateBreadcrumbs();
    });
  }

  private updateBreadcrumbs() {
    const url = this.router.url;
    const paths = url.split('/').filter(p => p);

    const crumbs: Breadcrumb[] = [
      { label: 'Home', url: '/' }
    ];

    let currentUrl = '';
    for (const path of paths) {
      currentUrl += `/${path}`;
      const label = this.routeLabels[path] || path;
      crumbs.push({ label, url: currentUrl });
    }

    this.breadcrumbs.set(crumbs);
  }

  navigate(url: string) {
    this.router.navigate([url]);
  }
}
