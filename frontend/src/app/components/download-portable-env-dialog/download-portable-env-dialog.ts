import { Component, Input, OnInit, ChangeDetectorRef, inject, ChangeDetectionStrategy, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { MatDialogModule, MatDialogRef, MatDialog } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { Wails } from '../../core/services/wails';
import { ConfirmDialogComponent } from '../confirm-dialog/confirm-dialog';

@Component({
  selector: 'app-download-portable-env-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatInputModule,
    MatButtonModule,
    MatProgressBarModule
  ],
  templateUrl: './download-portable-env-dialog.html',
  styleUrl: './download-portable-env-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DownloadPortableEnvDialogComponent implements OnInit {
  @Input() environment: 'python' | 'r-portable' = 'python';

  form: FormGroup;
  platforms = ['win', 'linux', 'darwin'];
  archs = ['x86_64', 'arm64'];

  downloading = false;
  progressItems: Record<string, {
    percentage: number;
    downloaded?: number;
    total?: number;
    files?: number;
  }> = {};
  currentPhase = '';
  overallProgress = 0;

  private dialog = inject(MatDialog);

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    public dialogRef: MatDialogRef<DownloadPortableEnvDialogComponent>,
    private cdr: ChangeDetectorRef
  ) {
    this.form = this.fb.group({
      platform: ['win'],
      arch: ['x86_64'],
      url: [{value: '', disabled: true}]
    });

    effect(() => {
      const progress = this.wails.progress();
      if (!progress) return;

      if (progress.type !== 'download' && progress.type !== 'extract' && progress.type !== 'install') {
        return;
      }

      const { message, percentage, status, type, data } = progress;

      if (status === 'completed') {
        if (type === 'install') {
          this.currentPhase = 'Installation complete!';
          this.overallProgress = 100;
          this.downloading = false;
          this.form.enable();
          this.cdr.detectChanges();

          this.wails.getPortableEnvironmentPath(this.environment).then(path => {
            if (this.environment === 'python') {
              this.wails.setSetting('pythonPath', path);
            } else {
              this.wails.setSetting('rPath', path);
            }
          }).catch(err => {
            const errorMessage = err instanceof Error ? err.message : String(err);
            this.currentPhase = 'Warning: ' + errorMessage;
            this.cdr.detectChanges();
          });
        }
      } else if (status === 'error') {
        this.currentPhase = 'Error: ' + message;
        this.downloading = false;
        this.form.enable();
        this.cdr.detectChanges();
      } else {
        let phaseWeight = 0;
        let phaseOffset = 0;

        if (type === 'download') {
          this.currentPhase = 'Downloading...';
          phaseWeight = 0.6;
          phaseOffset = 0;
        } else if (type === 'extract') {
          this.currentPhase = 'Extracting files...';
          phaseWeight = 0.2;
          phaseOffset = 60;
        } else if (type === 'install') {
          this.currentPhase = 'Installing...';
          phaseWeight = 0.2;
          phaseOffset = 80;
        }

        this.overallProgress = Math.round(phaseOffset + (percentage * phaseWeight));

        const downloaded = data?.['downloaded'] || 0;
        const total = data?.['total'] || 0;
        const files = data?.['files'] || 0;

        this.progressItems[message] = {
          percentage,
          downloaded: type === 'download' ? downloaded : undefined,
          total: type === 'download' ? total : undefined,
          files: (type === 'extract' || type === 'install') && files > 0 ? files : undefined
        };
        this.cdr.detectChanges();
      }
    });
  }

  ngOnInit() {
    this.detectPlatformAndArch();
    this.setupFormListeners();
    this.updateURL();
  }

  private detectPlatformAndArch() {
    const userAgent = window.navigator.userAgent.toLowerCase();
    let platform = 'win';

    if (userAgent.includes('linux')) {
      platform = 'linux';
    } else if (userAgent.includes('mac')) {
      platform = 'darwin';
    }

    this.form.patchValue({ platform });
  }

  private setupFormListeners() {
    this.form.get('platform')?.valueChanges.subscribe(() => {
      this.updateURL();
    });

    this.form.get('arch')?.valueChanges.subscribe(() => {
      this.updateURL();
    });
  }

  private async updateURL() {
    const platform = this.form.get('platform')?.value;
    const arch = this.form.get('arch')?.value;
    const version = 'latest'; // Use latest release

    try {
      const url = await this.wails.getPortableEnvironmentURL(
        platform,
        arch,
        version,
        this.environment
      );
      this.form.patchValue({ url });
    } catch (error) {
      console.error('Failed to get download URL:', error);
      this.form.patchValue({ url: 'Error: Could not fetch download URL - ' + (error as Error).message });
    }
  }

  async download() {
    if (this.downloading) return;

    const url = this.form.get('url')?.value;
    if (!url || url.startsWith('Error:')) {
      await this.wails.logToFile('[DownloadPortableEnv] Invalid download URL');
      return;
    }

    try {
      const existingPath = await this.wails.getPortableEnvironmentPath(this.environment);

      if (existingPath && existingPath !== '') {
        const envName = this.environment === 'python' ? 'Python' : 'R';
        const dialogRef = this.dialog.open(ConfirmDialogComponent, {
          width: '400px',
          disableClose: true,
          data: {
            title: `Replace Existing ${envName} Environment?`,
            message: `A portable ${envName} environment is already installed at:\n\n${existingPath}\n\nDownloading a new environment will remove the existing one. Do you want to continue?`,
            confirmText: 'Yes, Replace',
            cancelText: 'Cancel'
          }
        });

        const confirmed = await dialogRef.afterClosed().toPromise();
        if (!confirmed) {
          await this.wails.logToFile('[DownloadPortableEnv] User cancelled download - existing environment found');
          return;
        }

        await this.wails.logToFile('[DownloadPortableEnv] User confirmed replacement of existing environment');
      }
    } catch (error) {
      await this.wails.logToFile(`[DownloadPortableEnv] Could not check existing environment: ${error}`);
    }

    this.progressItems = {};
    this.downloading = true;
    this.form.disable();

    try {
      await this.wails.downloadPortableEnvironment(url, this.environment);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.progressItems['Error: ' + errorMessage] = { percentage: 0 };
      this.downloading = false;
      this.form.enable();
      this.cdr.detectChanges();
      await this.wails.logToFile(`[DownloadPortableEnv] Download error: ${errorMessage}`);
    }
  }

  getProgressMessages(): string[] {
    return Object.keys(this.progressItems);
  }

  close() {
    this.dialogRef.close();
  }

  formatBytes(bytes: number): string {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  }
}
