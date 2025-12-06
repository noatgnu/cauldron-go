import { Component, output } from '@angular/core';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { Router } from '@angular/router';

interface NavItem {
  label: string;
  icon: string;
  route?: string;
  children?: NavItem[];
}

@Component({
  selector: 'app-sidenav',
  imports: [MatListModule, MatIconModule, MatExpansionModule],
  templateUrl: './sidenav.html',
  styleUrl: './sidenav.scss',
})
export class Sidenav {
  navigationClose = output<void>();

  navItems: NavItem[] = [
    { label: 'Home', icon: 'home', route: '/' },
    {
      label: 'Data Transformation',
      icon: 'transform',
      children: [
        { label: 'Imputation', icon: 'auto_fix_high', route: '/analysis/imputation' },
        { label: 'Normalization', icon: 'tune', route: '/analysis/normalization' }
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

  constructor(private router: Router) {}

  navigate(route: string): void {
    this.router.navigate([route]);
    this.navigationClose.emit();
  }
}
