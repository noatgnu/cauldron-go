import { Component, Inject, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';

export interface PackagesModalData {
  environmentName: string;
  packages: string[];
  loading: boolean;
}

@Component({
  selector: 'app-packages-modal',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatListModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule
  ],
  templateUrl: './packages-modal.html',
  styleUrl: './packages-modal.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PackagesModal {
  packages = signal<string[]>([]);
  loading = signal(true);
  searchQuery = signal('');

  filteredPackages = computed(() => {
    const query = this.searchQuery().toLowerCase().trim();
    const allPackages = this.packages();

    if (!query) {
      return allPackages;
    }

    return allPackages.filter(pkg => pkg.toLowerCase().includes(query));
  });

  constructor(
    public dialogRef: MatDialogRef<PackagesModal>,
    @Inject(MAT_DIALOG_DATA) public data: PackagesModalData
  ) {
    if (data) {
      this.packages.set(data.packages || []);
      this.loading.set(data.loading !== undefined ? data.loading : true);
    }
  }

  setPackages(packages: string[]): void {
    this.packages.set(packages);
  }

  setLoading(loading: boolean): void {
    this.loading.set(loading);
  }

  clearSearch(): void {
    this.searchQuery.set('');
  }

  close(): void {
    this.dialogRef.close();
  }
}
