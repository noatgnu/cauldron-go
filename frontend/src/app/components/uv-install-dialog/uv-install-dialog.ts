import { Component, OnInit, ChangeDetectorRef, ChangeDetectionStrategy, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { Wails } from '../../core/services/wails';

@Component({
  selector: 'app-uv-install-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressBarModule
  ],
  templateUrl: './uv-install-dialog.html',
  styleUrl: './uv-install-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class UvInstallDialog implements OnInit {
  form: FormGroup;

  installingUv = false;
  uvInstalled = false;
  uvPath = '';

  installingPython = false;
  pythonInstalled = false;

  progressMessage = '';
  progressPercentage = 0;
  errorMessage = '';

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    public dialogRef: MatDialogRef<UvInstallDialog>,
    private cdr: ChangeDetectorRef
  ) {
    this.form = this.fb.group({
      pythonVersion: ['', Validators.pattern(/^\d+\.\d+(\.\d+)?$/)]
    });

    effect(() => {
      const progress = this.wails.progress();
      if (!progress) return;

      if (progress.id !== 'uv-install' && !progress.id?.startsWith('uv-python-install')) {
        return;
      }

      const { message, percentage, status } = progress;

      if (status === 'completed') {
        this.progressMessage = message;
        this.progressPercentage = 100;
        if (progress.id === 'uv-install') {
          this.installingUv = false;
          this.uvInstalled = true;
          this.refreshUvStatus();
        } else {
          this.installingPython = false;
          this.pythonInstalled = true;
        }
        this.cdr.detectChanges();
      } else if (status === 'error') {
        this.errorMessage = message;
        this.installingUv = false;
        this.installingPython = false;
        this.cdr.detectChanges();
      } else {
        this.progressMessage = message;
        this.progressPercentage = percentage;
        this.cdr.detectChanges();
      }
    });
  }

  async ngOnInit(): Promise<void> {
    await this.refreshUvStatus();
  }

  private async refreshUvStatus(): Promise<void> {
    try {
      this.uvInstalled = await this.wails.isUvAvailable();
      if (this.uvInstalled) {
        this.uvPath = await this.wails.getUvPath();
      }
    } catch (error) {
      this.uvInstalled = false;
      this.uvPath = '';
    }
    this.cdr.detectChanges();
  }

  async installUv(): Promise<void> {
    if (this.installingUv) return;

    this.errorMessage = '';
    this.progressMessage = '';
    this.progressPercentage = 0;
    this.installingUv = true;

    try {
      await this.wails.downloadUv();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.errorMessage = errorMessage;
      this.installingUv = false;
      this.cdr.detectChanges();
      await this.wails.logToFile(`[UvInstallDialog] Install error: ${errorMessage}`);
    }
  }

  async installPythonVersion(): Promise<void> {
    const version = this.form.get('pythonVersion')?.value;
    if (!version || this.installingPython) return;

    this.errorMessage = '';
    this.progressMessage = '';
    this.progressPercentage = 0;
    this.installingPython = true;
    this.pythonInstalled = false;

    try {
      await this.wails.installUvPythonVersion(version);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.errorMessage = errorMessage;
      this.installingPython = false;
      this.cdr.detectChanges();
      await this.wails.logToFile(`[UvInstallDialog] Python install error: ${errorMessage}`);
    }
  }

  close(): void {
    this.dialogRef.close(this.uvInstalled || this.pythonInstalled);
  }
}
