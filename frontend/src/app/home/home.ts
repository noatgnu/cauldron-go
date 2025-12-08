import { Component, OnInit, signal, computed } from '@angular/core';
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
  styleUrl: './home.scss'
})
export class Home implements OnInit {
  protected pythonVersion = signal('');
  protected pythonPath = signal('');
  protected rVersion = signal('');
  protected rPath = signal('');
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

  constructor(
    protected wails: Wails,
    private dialog: MatDialog,
    private router: Router,
    private notificationService: NotificationService
  ) {}

  async ngOnInit() {
    const addLog = (msg: string) => {
      const logs = this.debugInfo().logs;
      this.debugInfo.set({ ...this.debugInfo(), logs: [...logs, `[${new Date().toLocaleTimeString()}] ${msg}`] });
    };

    // Force stop loading after 5 seconds
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

    const windowGoExists = typeof window !== 'undefined' && !!(window as any).go;
    const windowRuntimeExists = typeof window !== 'undefined' && !!(window as any).runtime;

    this.debugInfo.set({
      ...this.debugInfo(),
      windowGoExists,
      windowRuntimeExists
    });

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
    this.wails.jobUpdate$.subscribe(job => {
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

    if (window.runtime) {
      window.runtime.EventsOn('menu:import-data', () => {
        this.openImportDialog();
      });

      window.runtime.EventsOn('file:imported', () => {
        this.loadImportedFiles();
      });
    }
  }

  async loadImportedFiles() {
    console.log('Starting to load imported files...');
    this.loadingFiles.set(true);
    try {
      const files = await this.wails.getImportedFiles();
      console.log('Loaded imported files:', files);
      this.importedFiles.set(files);
      this.debugInfo.set({ ...this.debugInfo(), filesLoaded: `✓ ${files.length} files` });
    } catch (error: any) {
      console.error('Failed to load imported files:', error);
      const errorMsg = error?.message || String(error);
      this.debugInfo.set({ ...this.debugInfo(), filesLoaded: `✗ Error: ${errorMsg}`, lastError: errorMsg });
    } finally {
      this.loadingFiles.set(false);
      console.log('Finished loading imported files');
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

    this.loadingVersions.set(false);
  }

  async loadJobs() {
    console.log('Starting to load jobs...');
    this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: 'Calling backend...' });

    try {
      console.log('About to call wails.getAllJobs()');
      const allJobs = await this.wails.getAllJobs();
      console.log('Received response from getAllJobs():', allJobs);

      if (!allJobs) {
        throw new Error('getAllJobs() returned null/undefined');
      }

      this.jobs.set(allJobs);
      this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: `✓ ${allJobs.length} jobs` });
      console.log('Successfully loaded jobs');
    } catch (error: any) {
      console.error('Failed to load jobs:', error);
      const errorMsg = error?.message || String(error);
      this.debugInfo.set({ ...this.debugInfo(), jobsLoaded: `✗ ${errorMsg}`, lastError: errorMsg });
    }
    console.log('Finished loading jobs');
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
    } catch (error) {
      console.error('Failed to delete job:', error);
    }
  }

  async rerunJob(id: string): Promise<void> {
    try {
      const newJobId = await this.wails.rerunJob(id, true, '', '');
      this.notificationService.showSuccess(`Job ${id} was successfully rerun as new job ${newJobId}`);
      await this.loadJobs(); // Refresh the job list
    } catch (error) {
      console.error('Failed to rerun job:', error);
      this.notificationService.showError('Failed to rerun job.');
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
    } catch (error) {
      console.error('Failed to delete imported file:', error);
    }
  }
}
