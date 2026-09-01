import { Component, Input, OnInit, ChangeDetectorRef, inject, ChangeDetectionStrategy, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatDialogModule, MatDialogRef, MatDialog } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { Wails, RVersionRelease, ProgressNotification } from '../../core/services/wails';
import { ConfirmDialogComponent } from '../confirm-dialog/confirm-dialog';

type StepStatus = 'pending' | 'active' | 'done' | 'error';

interface InstallStep {
  key: string;
  label: string;
  status: StepStatus;
  detail?: string;
  percentage?: number;
  downloaded?: number;
  total?: number;
}

const PYTHON_STEPS: Omit<InstallStep, 'status'>[] = [
  { key: 'download', label: 'Download' },
  { key: 'extract', label: 'Extract' },
  { key: 'install', label: 'Install' }
];

const R_STEPS: Omit<InstallStep, 'status'>[] = [
  { key: 'download', label: 'Download' },
  { key: 'verify', label: 'Verify checksum' },
  { key: 'extract', label: 'Extract' },
  { key: 'install', label: 'Install' }
];

@Component({
  selector: 'app-download-portable-env-dialog',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatInputModule,
    MatButtonModule,
    MatProgressBarModule,
    MatIconModule
  ],
  templateUrl: './download-portable-env-dialog.html',
  styleUrl: './download-portable-env-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class DownloadPortableEnvDialogComponent implements OnInit {
  @Input() environment: 'python' | 'r-portable' = 'python';

  form: FormGroup;

  availableRVersions: RVersionRelease[] = [];
  installedRVersions: string[] = [];
  selectedRVersion = '';
  loadingRVersions = false;

  installing = false;
  steps: InstallStep[] = [];
  finished = false;
  errorMessage = '';

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
      if (this.environment === 'python') {
        this.handlePythonProgress(progress);
      } else {
        this.handleRProgress(progress);
      }
    });
  }

  ngOnInit() {
    if (this.environment === 'python') {
      this.detectPlatformAndArch();
      this.setupFormListeners();
      this.updateURL();
    } else {
      this.refreshRVersions();
    }
  }

  isInstalled(version: string): boolean {
    return this.installedRVersions.includes(version);
  }

  private async refreshRVersions() {
    this.loadingRVersions = true;
    this.cdr.detectChanges();
    try {
      const [available, installed] = await Promise.all([
        this.wails.listAvailableRVersions(),
        this.wails.listInstalledRVersions()
      ]);
      this.availableRVersions = available;
      this.installedRVersions = installed;
      if (!this.selectedRVersion && available.length > 0) {
        this.selectedRVersion = available[0].version;
      }
    } catch (error) {
      await this.wails.logToFile(`[DownloadPortableEnv] Failed to list R versions: ${error}`);
      this.errorMessage = error instanceof Error ? error.message : String(error);
    } finally {
      this.loadingRVersions = false;
      this.cdr.detectChanges();
    }
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
    const version = 'latest';

    try {
      const url = await this.wails.getPortableEnvironmentURL(platform, arch, version, this.environment);
      this.form.patchValue({ url });
    } catch (error) {
      await this.wails.logToFile(`[DownloadPortableEnv] Failed to get download URL: ${error}`);
      this.form.patchValue({ url: 'Error: Could not fetch download URL - ' + (error as Error).message });
    }
  }

  private startSteps(template: Omit<InstallStep, 'status'>[]) {
    this.steps = template.map((s, i) => ({ ...s, status: i === 0 ? 'active' : 'pending' }));
    this.installing = true;
    this.finished = false;
    this.errorMessage = '';
    this.cdr.detectChanges();
  }

  private setStep(key: string, patch: Partial<InstallStep>) {
    const step = this.steps.find(s => s.key === key);
    if (step) Object.assign(step, patch);
  }

  private completeUpTo(key: string) {
    const idx = this.steps.findIndex(s => s.key === key);
    if (idx < 0) return;
    for (let i = 0; i < idx; i++) {
      this.steps[i].status = 'done';
    }
    this.steps[idx].status = 'active';
  }

  private failAllFrom(message: string) {
    const active = this.steps.find(s => s.status === 'active');
    if (active) active.status = 'error';
    this.errorMessage = message;
    this.installing = false;
    this.cdr.detectChanges();
  }

  private handlePythonProgress(progress: ProgressNotification) {
    if (progress.type !== 'download' && progress.type !== 'extract' && progress.type !== 'install') return;
    if (!this.installing) return;

    const { message, percentage, status, type, data } = progress;

    if (status === 'error') {
      this.failAllFrom(message);
      return;
    }

    this.completeUpTo(type);

    if (status === 'completed' && type === 'install' && progress.id === 'python.tar.xz') {
      this.steps.forEach(s => (s.status = 'done'));
      this.finished = true;
      this.installing = false;
      this.cdr.detectChanges();
      this.wails.getPortableEnvironmentPath(this.environment).then(path => {
        this.wails.setSetting('pythonPath', path);
      }).catch(err => {
        this.errorMessage = 'Warning: ' + (err instanceof Error ? err.message : String(err));
        this.cdr.detectChanges();
      });
      return;
    }

    this.setStep(type, {
      detail: message,
      percentage,
      downloaded: type === 'download' ? (data?.['downloaded'] as number) : undefined,
      total: type === 'download' ? (data?.['total'] as number) : undefined
    });
    this.cdr.detectChanges();
  }

  private handleRProgress(progress: ProgressNotification) {
    if (progress.type !== 'install' || !this.installing) return;
    if (progress.id !== 'r-portable-' + this.selectedRVersion) return;

    const { message, status } = progress;

    if (status === 'error') {
      this.failAllFrom(message);
      return;
    }

    if (status === 'completed') {
      this.steps.forEach(s => (s.status = 'done'));
      this.finished = true;
      this.installing = false;
      this.cdr.detectChanges();
      this.wails.getRPortablePath(this.selectedRVersion).then(async path => {
        await this.wails.setSetting('rPath', path);
        // detectREnvironments persists this version's DB row -- setActiveREnvironment needs it to already exist.
        await this.wails.detectREnvironments();
        await this.wails.setActiveREnvironment(path);
      }).catch(err => {
        this.errorMessage = 'Warning: ' + (err instanceof Error ? err.message : String(err));
        this.cdr.detectChanges();
      });
      this.refreshRVersions();
      return;
    }

    if (message.startsWith('Downloading')) {
      this.completeUpTo('download');
      this.setStep('download', { detail: message });
    } else if (message.startsWith('Verifying')) {
      this.completeUpTo('verify');
      this.setStep('verify', { detail: message });
    } else if (message.startsWith('Extracting')) {
      this.completeUpTo('extract');
      this.setStep('extract', { detail: message });
    } else {
      this.completeUpTo('install');
      this.setStep('install', { detail: message });
    }
    this.cdr.detectChanges();
  }

  async download() {
    if (this.installing) return;

    const url = this.form.get('url')?.value;
    if (!url || url.startsWith('Error:')) {
      await this.wails.logToFile('[DownloadPortableEnv] Invalid download URL');
      return;
    }

    try {
      const existingPath = await this.wails.getPortableEnvironmentPath(this.environment);
      if (existingPath && existingPath !== '') {
        const confirmed = await this.confirmReplace('Python', existingPath);
        if (!confirmed) return;
      }
    } catch (error) {
      await this.wails.logToFile(`[DownloadPortableEnv] Could not check existing environment: ${error}`);
    }

    this.startSteps(PYTHON_STEPS);
    this.form.disable();

    try {
      await this.wails.downloadPortableEnvironment(url, this.environment);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.failAllFrom(errorMessage);
      this.form.enable();
      await this.wails.logToFile(`[DownloadPortableEnv] Download error: ${errorMessage}`);
    }
  }

  async installR() {
    if (this.installing || !this.selectedRVersion) return;

    if (this.isInstalled(this.selectedRVersion)) {
      const confirmed = await this.confirmReplace('R ' + this.selectedRVersion, this.selectedRVersion);
      if (!confirmed) return;
      try {
        await this.wails.uninstallRVersion(this.selectedRVersion);
      } catch (error) {
        this.errorMessage = error instanceof Error ? error.message : String(error);
        this.cdr.detectChanges();
        return;
      }
    }

    this.startSteps(R_STEPS);

    try {
      await this.wails.installRVersion(this.selectedRVersion);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.failAllFrom(errorMessage);
      await this.wails.logToFile(`[DownloadPortableEnv] R install error: ${errorMessage}`);
    }
  }

  async uninstallR(version: string) {
    const confirmed = await this.confirmReplace('R ' + version, version, true);
    if (!confirmed) return;
    try {
      await this.wails.uninstallRVersion(version);
      await this.refreshRVersions();
    } catch (error) {
      this.errorMessage = error instanceof Error ? error.message : String(error);
      this.cdr.detectChanges();
    }
  }

  private async confirmReplace(envName: string, existingPath: string, uninstallOnly = false): Promise<boolean> {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      disableClose: true,
      data: uninstallOnly
        ? {
            title: `Remove ${envName}?`,
            message: `This will remove the installed environment. Do you want to continue?`,
            confirmText: 'Yes, Remove',
            cancelText: 'Cancel'
          }
        : {
            title: `Replace Existing ${envName} Environment?`,
            message: `A portable ${envName} environment is already installed at:\n\n${existingPath}\n\nContinuing will remove the existing one first. Do you want to continue?`,
            confirmText: 'Yes, Replace',
            cancelText: 'Cancel'
          }
    });
    return !!(await dialogRef.afterClosed().toPromise());
  }

  close() {
    this.dialogRef.close();
  }

  formatBytes(bytes: number): string {
    if (!bytes) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  }
}
