import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

export interface PluginRequirementsDialogData {
  pluginId: string;
  pluginName: string;
  runtimeEnvironments: string[];
  pythonPackages?: string[];
  rPackages?: string[];
}

@Component({
  selector: 'app-plugin-requirements-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatProgressSpinnerModule
  ],
  template: `
    <h2 mat-dialog-title>
      <mat-icon>extension</mat-icon>
      Install Plugin Requirements
    </h2>

    <mat-dialog-content>
      <p>The plugin <strong>{{ data.pluginName }}</strong> has the following requirements:</p>

      @if (data.pythonPackages && data.pythonPackages.length > 0) {
        <h3>Python Packages ({{ data.pythonPackages.length }})</h3>
        <mat-list dense>
          @for (pkg of data.pythonPackages; track pkg) {
            <mat-list-item>
              <mat-icon matListItemIcon>check_circle</mat-icon>
              <span matListItemTitle>{{ pkg }}</span>
            </mat-list-item>
          }
        </mat-list>
      }

      @if (data.rPackages && data.rPackages.length > 0) {
        <h3>R Packages ({{ data.rPackages.length }})</h3>
        <mat-list dense>
          @for (pkg of data.rPackages; track pkg) {
            <mat-list-item>
              <mat-icon matListItemIcon>check_circle</mat-icon>
              <span matListItemTitle>{{ pkg }}</span>
            </mat-list-item>
          }
        </mat-list>
      }

      <p class="warning-text">
        <mat-icon>warning</mat-icon>
        Installation may take several minutes depending on package sizes.
      </p>
    </mat-dialog-content>

    <mat-dialog-actions align="end">
      <button mat-button (click)="skip()">
        <mat-icon>skip_next</mat-icon>
        Skip for Now
      </button>
      <button mat-raised-button color="primary" (click)="install()">
        <mat-icon>download</mat-icon>
        Install Requirements
      </button>
    </mat-dialog-actions>
  `,
  styles: [`
    mat-dialog-content {
      min-width: 400px;
      max-height: 500px;
    }

    h3 {
      margin-top: 16px;
      margin-bottom: 8px;
      font-size: 14px;
      font-weight: 500;
      color: rgba(0, 0, 0, 0.87);
    }

    .warning-text {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 12px;
      margin-top: 16px;
      background-color: #fff3cd;
      border-left: 4px solid #ffc107;
      border-radius: 4px;
      font-size: 13px;
      color: #856404;
    }

    .warning-text mat-icon {
      font-size: 20px;
      width: 20px;
      height: 20px;
      color: #ffc107;
    }

    mat-list {
      max-height: 200px;
      overflow-y: auto;
    }

    mat-list-item {
      font-size: 13px;
    }
  `]
})
export class PluginRequirementsDialog {
  constructor(
    public dialogRef: MatDialogRef<PluginRequirementsDialog>,
    @Inject(MAT_DIALOG_DATA) public data: PluginRequirementsDialogData
  ) {}

  install() {
    this.dialogRef.close(true);
  }

  skip() {
    this.dialogRef.close(false);
  }
}
