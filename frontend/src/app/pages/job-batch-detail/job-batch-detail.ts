import { Component, OnInit, OnDestroy, signal, ChangeDetectionStrategy, effect, untracked } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription, firstValueFrom } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { Wails, BatchStatus } from '../../core/services/wails';
import { JobBatchService } from '../../core/services/job-batch';
import { ConfirmDialogComponent } from '../../components/confirm-dialog/confirm-dialog';

@Component({
  selector: 'app-job-batch-detail',
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatProgressBarModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './job-batch-detail.html',
  styleUrl: './job-batch-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobBatchDetail implements OnInit, OnDestroy {
  protected batch = signal<BatchStatus | null>(null);
  protected loading = signal(true);
  protected displayedColumns: string[] = ['status', 'name', 'progress', 'createdAt', 'actions'];
  private batchId = '';
  private paramMapSubscription?: Subscription;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private jobBatchService: JobBatchService,
    private dialog: MatDialog
  ) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (!job || job.batchId !== untracked(() => this.batchId)) return;
      this.loadBatch();
    });
  }

  ngOnInit(): void {
    this.paramMapSubscription = this.route.paramMap.subscribe(async params => {
      const id = params.get('id');
      if (!id) return;
      this.batchId = id;
      await this.loadBatch();
    });
  }

  ngOnDestroy(): void {
    this.paramMapSubscription?.unsubscribe();
  }

  async loadBatch(): Promise<void> {
    this.loading.set(true);
    try {
      const status = await this.jobBatchService.getBatchStatus(this.batchId);
      this.batch.set(status);
    } catch (error: any) {
      await this.wails.logToFile(`[JobBatchDetail] Failed to load batch: ${error?.message || String(error)}`);
    } finally {
      this.loading.set(false);
    }
  }

  viewJob(id: string): void {
    this.router.navigate(['/jobs', id]);
  }

  async deleteBatch(): Promise<void> {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      disableClose: true,
      data: {
        title: 'Delete this batch?',
        message: 'This deletes the batch and all of its jobs, including any still pending or in progress. This cannot be undone.',
        confirmText: 'Yes, Delete',
        cancelText: 'Cancel'
      }
    });

    const confirmed = await firstValueFrom(dialogRef.afterClosed());
    if (!confirmed) return;

    try {
      await this.jobBatchService.deleteBatch(this.batchId);
      await this.router.navigate(['/job-batches']);
    } catch (error: any) {
      await this.wails.logToFile(`[JobBatchDetail] Failed to delete batch: ${error?.message || String(error)}`);
    }
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
}
