import { Component, OnInit, signal } from '@angular/core';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatMenuModule } from '@angular/material/menu';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatTableModule } from '@angular/material/table';
import { MatDialog } from '@angular/material/dialog';
import { CommonModule } from '@angular/common';
import { Wails, Job, PythonEnvironment, REnvironment } from '../../core/services/wails';

@Component({
  selector: 'app-jobs',
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatProgressBarModule,
    MatMenuModule,
    MatTooltipModule,
    MatTableModule
  ],
  templateUrl: './jobs.html',
  styleUrl: './jobs.scss',
})
export class Jobs implements OnInit {
  protected jobs = signal<Job[]>([]);
  protected loading = signal(false);
  protected pythonEnvironments = signal<PythonEnvironment[]>([]);
  protected rEnvironments = signal<REnvironment[]>([]);
  protected jobProgress = signal<Record<string, {message: string, percentage: number}>>({});
  protected displayedColumns: string[] = ['status', 'name', 'type', 'environment', 'createdAt', 'actions'];

  constructor(
    private wails: Wails,
    private router: Router,
    private dialog: MatDialog
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadJobs();
    await this.loadEnvironments();
    this.setupJobUpdates();
    this.setupProgressUpdates();
  }

  async loadEnvironments(): Promise<void> {
    try {
      const [pythonEnvs, rEnvs] = await Promise.all([
        this.wails.detectPythonEnvironments(),
        this.wails.detectREnvironments()
      ]);
      this.pythonEnvironments.set(pythonEnvs);
      this.rEnvironments.set(rEnvs);
    } catch (error) {
      console.error('Failed to load environments:', error);
    }
  }

  async loadJobs(): Promise<void> {
    this.loading.set(true);
    try {
      const allJobs = await this.wails.getAllJobs();
      this.jobs.set(allJobs);
    } catch (error) {
      console.error('Failed to load jobs:', error);
    } finally {
      this.loading.set(false);
    }
  }

  setupJobUpdates(): void {
    this.wails.jobUpdate$.subscribe(job => {
      if (!job) return;

      const currentJobs = this.jobs();
      const index = currentJobs.findIndex(j => j.id === job.id);

      if (index >= 0) {
        const updated = [...currentJobs];
        updated[index] = job;
        this.jobs.set(updated);
      } else {
        this.jobs.set([job, ...currentJobs]);
      }
    });
  }

  setupProgressUpdates(): void {
    this.wails.progress$.subscribe(progress => {
      if (!progress) return;

      if (progress.type === 'script' || progress.type === 'analysis') {
        const jobId = progress.id;
        const currentProgress = this.jobProgress();

        if (progress.status === 'completed' || progress.status === 'error') {
          const { [jobId]: _, ...rest } = currentProgress;
          this.jobProgress.set(rest);
        } else {
          this.jobProgress.set({
            ...currentProgress,
            [jobId]: {
              message: progress.message,
              percentage: progress.percentage
            }
          });
        }
      }
    });
  }

  async deleteJob(event: Event, id: string): Promise<void> {
    event.stopPropagation();
    try {
      await this.wails.deleteJob(id);
      this.jobs.update(jobs => jobs.filter(j => j.id !== id));
    } catch (error) {
      console.error('Failed to delete job:', error);
    }
  }

  viewJobDetail(id: string): void {
    this.router.navigate(['/jobs', id]);
  }

  getStatusColor(status: string): string {
    switch (status) {
      case 'completed': return 'primary';
      case 'in_progress': return 'accent';
      case 'failed': return 'warn';
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

  getEnvironmentLabel(job: Job): string {
    const parts: string[] = [];

    if (job.pythonEnvPath) {
      const env = this.pythonEnvironments().find(e => e.path === job.pythonEnvPath);
      if (env) {
        parts.push(`Python: ${env.name} (${this.getEnvironmentTypeLabel(job.pythonEnvType || '')})`);
      } else {
        parts.push(`Python: ${job.pythonEnvType || 'Unknown'}`);
      }
    }

    if (job.rEnvPath) {
      const env = this.rEnvironments().find(e => e.path === job.rEnvPath);
      if (env) {
        parts.push(`R: ${env.name}`);
      } else {
        parts.push(`R: ${job.rEnvType || 'Unknown'}`);
      }
    }

    return parts.length > 0 ? parts.join(', ') : 'No environment recorded';
  }

  getEnvironmentTypeLabel(type: string): string {
    switch (type) {
      case 'system': return 'System';
      case 'conda': return 'Conda';
      case 'venv': return 'Virtual';
      case 'poetry': return 'Poetry';
      case 'portable': return 'Portable';
      default: return type || 'Unknown';
    }
  }

  canRerun(job: Job): boolean {
    return job.status === 'completed' || job.status === 'failed';
  }

  async rerunWithSameEnvironment(event: Event, job: Job): Promise<void> {
    event.stopPropagation();
    try {
      const newJobId = await this.wails.rerunJob(job.id, true, '', '');
      await this.loadJobs();
      this.router.navigate(['/jobs', newJobId]);
    } catch (error) {
      console.error('Failed to rerun job:', error);
    }
  }

  async rerunWithDifferentEnvironment(event: Event, job: Job): Promise<void> {
    event.stopPropagation();

    try {
      const settings = await this.wails.getSettings();

      const newJobId = await this.wails.rerunJob(
        job.id,
        false,
        settings.pythonPath || '',
        settings.rPath || ''
      );
      await this.loadJobs();
      this.router.navigate(['/jobs', newJobId]);
    } catch (error) {
      console.error('Failed to rerun job:', error);
    }
  }

  isEnvironmentMissing(job: Job): boolean {
    if (job.pythonEnvPath) {
      const exists = this.pythonEnvironments().some(e => e.path === job.pythonEnvPath);
      if (!exists) return true;
    }

    if (job.rEnvPath) {
      const exists = this.rEnvironments().some(e => e.path === job.rEnvPath);
      if (!exists) return true;
    }

    return false;
  }

  getJobProgress(jobId: string): {message: string, percentage: number} | null {
    return this.jobProgress()[jobId] || null;
  }

  hasProgress(jobId: string): boolean {
    return !!this.jobProgress()[jobId];
  }

  async openOutputDirectory(event: Event, job: Job): Promise<void> {
    event.stopPropagation();
    if (!job.outputPath) {
      return;
    }
    try {
      await this.wails.openDirectoryInExplorer(job.outputPath);
    } catch (error) {
      console.error('Failed to open output directory:', error);
    }
  }

  hasOutputDirectory(job: Job): boolean {
    return !!job.outputPath && (job.status === 'completed' || job.status === 'failed');
  }
}
