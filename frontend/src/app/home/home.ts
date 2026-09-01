import { Component, OnInit, signal, computed, ChangeDetectionStrategy, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { Events } from '@wailsio/runtime';
import { Wails, Job, ImportedFile } from '../core/services/wails';
import { ImportDialog } from '../pages/import-dialog/import-dialog';
import { NotificationService } from '../core/services/notification.service';

@Component({
  selector: 'app-home',
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './home.html',
  styleUrl: './home.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Home implements OnInit {
  protected pythonVersion = signal('');
  protected pythonPath = signal('');
  protected rVersion = signal('');
  protected rPath = signal('');
  protected dockerVersion = signal('');
  protected jobs = signal<Job[]>([]);
  protected recentJobs = computed(() => this.jobs().slice(0, 5));
  protected importedFiles = signal<ImportedFile[]>([]);
  protected loading = signal(false);
  protected loadingVersions = signal(false);
  protected loadingFiles = signal(false);

  protected displayedColumns: string[] = ['status', 'name', 'type', 'createdAt', 'actions'];

  protected debugInfo = signal({
    windowGoExists: false,
    windowRuntimeExists: false,
    jobsLoaded: 'Not started',
    filesLoaded: 'Not started',
    versionsLoaded: 'Not started',
    lastError: '',
    logs: [] as string[]
  });
  protected debugModeEnabled = signal(false);

  constructor(
    protected wails: Wails,
    private dialog: MatDialog,
    private router: Router,
    private notificationService: NotificationService
  ) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (job) {
        const currentJobs = this.jobs();
        const index = currentJobs.findIndex(j => j.id === job.id);
        if (index >= 0) {
          const updated = [...currentJobs];
          updated[index] = job;
          this.jobs.set(updated);
        } else {
          this.jobs.set([job, ...currentJobs]);
        }
      }
    });
  }

  async ngOnInit() {
    this.wails.getSettings()
      .then(config => this.debugModeEnabled.set(!!config.debugMode))
      .catch(() => {});

    const addLog = (msg: string) => {
      const logs = this.debugInfo().logs;
      this.debugInfo.set({ ...this.debugInfo(), logs: [...logs, `[${new Date().toLocaleTimeString()}] ${msg}`] });
    };

    setTimeout(() => {
      if (this.loading()) {
        addLog('⚠️ Timeout: Stopping loading spinner after 5 seconds');
        this.loading.set(false);
        this.loadingVersions.set(false);
        this.loadingFiles.set(false);
        if (!this.debugInfo().lastError) {
          this.debugInfo.set({ ...this.debugInfo(), lastError: 'Loading timeout - check if backend is responding' });
        }
      }
    }, 5000);

    addLog('=== Home Component Initialization ===');

    const windowWailsExists = typeof window !== 'undefined' && '_wails' in window;
    const windowGoExists = typeof window !== 'undefined' && !!(window as any).go;
    const windowRuntimeExists = typeof window !== 'undefined' && !!(window as any).runtime;

    this.debugInfo.set({
      ...this.debugInfo(),
      windowGoExists: windowWailsExists || windowGoExists,
      windowRuntimeExists
    });

    addLog(`window._wails exists: ${windowWailsExists}`);
    addLog(`window.go exists: ${windowGoExists}`);
    addLog(`window.runtime exists: ${windowRuntimeExists}`);
    addLog(`wails.isWails: ${this.wails.isWails}`);

    if (!this.wails.isWails) {
      addLog('❌ Wails runtime not available');
      addLog('This means window.go is not defined');
      this.debugInfo.set({ ...this.debugInfo(), lastError: 'Wails runtime not available - window.go is undefined' });
      this.loading.set(false);
      this.loadingVersions.set(false);
      this.loadingFiles.set(false);
      return;
    }

    addLog('✓ Wails runtime detected, starting initialization...');

    this.loading.set(true);
    try {
      addLog('Calling loadJobs()...');
      const jobsPromise = Promise.race([
        this.loadJobs(),
        new Promise((_, reject) => setTimeout(() => reject(new Error('loadJobs timeout after 3s')), 3000))
      ]);
      await jobsPromise;
      addLog('✓ loadJobs() completed');

      addLog('Calling loadVersions()...');
      this.loadVersions();

      addLog('Calling loadImportedFiles()...');
      this.loadImportedFiles();
    } catch (error: any) {
      const errorMsg = error?.message || String(error);
      addLog(`❌ Failed to initialize: ${errorMsg}`);
      this.debugInfo.set({ ...this.debugInfo(), lastError: errorMsg });
    } finally {
      this.loading.set(false);
    }

    this.setupEventListeners();
    addLog('✓ Initialization complete');
  }

  private setupEventListeners(): void {
    Events.On('menu:import-data', () => {
      this.openImportDialog();
    });

    Events.On('file:imported', () => {
      this.loadImportedFiles();
    });
  }

  async loadImportedFiles() {
    this.loadingFiles.set(true);
    try {
      const files = await this.wails.getImportedFiles();
      this.importedFiles.set(files);
      this.debugInfo.set({ ...this.debugInfo(), filesLoaded: `✓ ${files.length} files` });
    } catch (error: any) {
      const errorMsg = error?.message || String(error);
      await this.wails.logToFile(`[Home] Failed to load imported files: ${errorMsg}`);
      this.debugInfo.set({ ...this.debugInfo(), filesLoaded: `✗ Error: ${errorMsg}`, lastError: errorMsg });
    } finally {
      this.loadingFiles.set(false);
    }
  }

  async loadVersions() {
    this.loadingVersions.set(true);
    try {
      const pythonEnv = await this.wails.getActivePythonEnvironment();

      if (pythonEnv && pythonEnv.path) {
        this.pythonPath.set(pythonEnv.path);
        try {
          const pyVersion = await this.wails.getPythonVersion();
          this.pythonVersion.set(pyVersion);
          this.debugInfo.set({ ...this.debugInfo(), versionsLoaded: `✓ Python: ${pyVersion}` });
        } catch (e: any) {
          this.pythonVersion.set('Error');
        }
      } else {
        this.pythonVersion.set('Not selected');
        this.pythonPath.set('');
      }
    } catch (e: any) {
      this.pythonVersion.set('Not selected');
      this.pythonPath.set('');
      const errorMsg = e?.message || String(e);
      this.debugInfo.set({ ...this.debugInfo(), versionsLoaded: `✗ Error: ${errorMsg}`, lastError: errorMsg });
    }

    try {
      const rEnv = await this.wails.getActiveREnvironment();

      if (rEnv && rEnv.path) {
        this.rPath.set(rEnv.path);
        try {
          const rVer = await this.wails.getRVersion();
          this.rVersion.set(rVer);
        } catch (e: any) {
          this.rVersion.set('Error');
        }
      } else {
        this.rVersion.set('Not selected');
        this.rPath.set('');
      }
    } catch (e) {
      this.rVersion.set('Not selected');
      this.rPath.set('');
    }

    try {
      const dockerVer = await this.wails.checkDockerVersion();
      this.dockerVersion.set(dockerVer);
    } catch (e) {
      this.dockerVersion.set('Not installed');
    }

    this.loadingVersions.set(false);
  }

  async loadJobs() {
    this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: 'Calling backend...' });

    try {
      const allJobs = await this.wails.getAllJobs();

      if (!allJobs) {
        throw new Error('getAllJobs() returned null/undefined');
      }

      this.jobs.set(allJobs);
      this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: `✓ ${allJobs.length} jobs` });
    } catch (error: any) {
      const errorMsg = error?.message || String(error);
      await this.wails.logToFile(`[Home] Failed to load jobs: ${errorMsg}`);
      this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: `✗ ${errorMsg}`, lastError: errorMsg });
    }
  }

  getStatusColor(status: string): string {
    switch (status) {
      case 'completed': return 'primary';
      case 'failed': return 'warn';
      case 'in_progress': return 'accent';
      default: return '';
    }
  }

  getStatusIcon(status: string): string {
    switch (status) {
      case 'completed': return 'check_circle';
      case 'in_progress': return 'sync';
      case 'failed': return 'error';
      default: return 'schedule';
    }
  }

  async deleteJob(id: string): Promise<void> {
    try {
      await this.wails.deleteJob(id);
      this.jobs.update(jobs => jobs.filter(j => j.id !== id));
    } catch (error: any) {
      await this.wails.logToFile(`[Home] Failed to delete job: ${error?.message || String(error)}`);
    }
  }

  async rerunJob(id: string): Promise<void> {
    try {
      const job = this.jobs().find(j => j.id === id);
      if (job?.type) {
        await this.ensureJobBinding(job.type);
      }
      const newJobId = await this.wails.rerunJob(id, true, '', '');
      this.notificationService.showSuccess(`Job ${id} was successfully rerun as new job ${newJobId}`);
      await this.loadJobs();
    } catch (error: any) {
      await this.wails.logToFile(`[Home] Failed to rerun job: ${error?.message || String(error)}`);
      this.notificationService.showError('Failed to rerun job.');
    }
  }

  private async ensureJobBinding(pluginStringId: string): Promise<void> {
    if (!pluginStringId) return;
    try {
      const pythonBinding = await this.wails.getPluginEnvironmentBinding(pluginStringId, 'python').catch(() => null);
      if (!pythonBinding) {
        const pythonEnv = await this.wails.getActivePythonEnvironment();
        if (pythonEnv?.path) {
          await this.wails.bindPluginToEnvironment(pluginStringId, 'python', 0, pythonEnv.path).catch(() => {});
        }
      }
      const rBinding = await this.wails.getPluginEnvironmentBinding(pluginStringId, 'r').catch(() => null);
      if (!rBinding) {
        const rEnv = await this.wails.getActiveREnvironment();
        if (rEnv?.path) {
          await this.wails.bindPluginToEnvironment(pluginStringId, 'r', 0, rEnv.path).catch(() => {});
        }
      }
    } catch (error: any) {
      await this.wails.logToFile(`[Home] Failed to ensure binding for ${pluginStringId}: ${error?.message || String(error)}`);
    }
  }

  viewJobDetail(id: string): void {
    this.router.navigate(['/jobs', id]);
  }

  openImportDialog(): void {
    const dialogRef = this.dialog.open(ImportDialog, {
      width: '800px',
      disableClose: true
    });

    dialogRef.afterClosed().subscribe(async result => {
      if (result?.imported) {
        await this.loadImportedFiles();
      }
    });
  }

  async deleteImportedFile(id: number): Promise<void> {
    try {
      await this.wails.deleteImportedFile(id);
      this.importedFiles.update(files => files.filter(f => f.id !== id));
    } catch (error: any) {
      await this.wails.logToFile(`[Home] Failed to delete imported file: ${error?.message || String(error)}`);
    }
  }
}
