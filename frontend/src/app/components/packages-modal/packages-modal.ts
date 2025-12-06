import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';

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
    MatIconModule
  ],
  templateUrl: './packages-modal.html',
  styleUrl: './packages-modal.scss',
})
export class PackagesModal {
  packages: string[] = [];
  loading = true;

  constructor(
    public dialogRef: MatDialogRef<PackagesModal>,
    @Inject(MAT_DIALOG_DATA) public data: PackagesModalData
  ) {
    if (data) {
      this.packages = data.packages || [];
      this.loading = data.loading !== undefined ? data.loading : true;
    }
  }

  setPackages(packages: string[]): void {
    this.packages = packages;
  }

  setLoading(loading: boolean): void {
    this.loading = loading;
  }

  close(): void {
    this.dialogRef.close();
  }
}
