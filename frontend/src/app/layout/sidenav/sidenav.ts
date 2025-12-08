import { Component, output, signal, computed } from '@angular/core';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';

interface NavItem {
  label: string;
  icon: string;
  route?: string;
  children?: NavItem[];
}

@Component({
  selector: 'app-sidenav',
  imports: [MatListModule, MatIconModule, MatExpansionModule, MatFormFieldModule, MatInputModule, MatButtonModule, FormsModule],
  templateUrl: './sidenav.html',
  styleUrl: './sidenav.scss',
})
export class Sidenav {
  navigationClose = output<void>();
  searchQuery = signal<string>('');

  navItems: NavItem[] = [
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

  filteredNavItems = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();

    if (!query) {
      return this.navItems;
    }

    return this.navItems.map(item => {
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

  constructor(private router: Router) {}

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
