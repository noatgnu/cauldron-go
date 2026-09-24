import { Component, OnInit, signal, ChangeDetectionStrategy, effect, untracked } from '@angular/core';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Wails, BatchStatus } from '../../core/services/wails';
import { JobBatchService } from '../../core/services/job-batch';

@Component({
  selector: 'app-job-batches',
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './job-batches.html',
  styleUrl: './job-batches.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobBatches implements OnInit {
  protected batches = signal<BatchStatus[]>([]);
  protected loading = signal(false);
  protected displayedColumns: string[] = ['label', 'plugin', 'status', 'createdAt', 'actions'];

  constructor(
    private wails: Wails,
    private jobBatchService: JobBatchService,
    private router: Router
  ) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (!job || !job.batchId) return;

      const batches = untracked(() => this.batches());
      if (!batches.some(b => b.batchId === job.batchId)) return;

      this.refreshBatch(job.batchId);
    });
  }

  async ngOnInit(): Promise<void> {
    await this.loadBatches();
  }

  async loadBatches(): Promise<void> {
    this.loading.set(true);
    try {
      const list = await this.jobBatchService.getAllBatches(100, 0);
      const statuses = await Promise.all(list.map(b => this.jobBatchService.getBatchStatus(b.id)));
      this.batches.set(statuses);
    } catch (error: any) {
      await this.wails.logToFile(`[JobBatches] Failed to load batches: ${error?.message || String(error)}`);
    } finally {
      this.loading.set(false);
    }
  }

  private async refreshBatch(batchId: string): Promise<void> {
    try {
      const status = await this.jobBatchService.getBatchStatus(batchId);
      this.batches.update(batches => batches.map(b => (b.batchId === batchId ? status : b)));
    } catch (error: any) {
      await this.wails.logToFile(`[JobBatches] Failed to refresh batch ${batchId}: ${error?.message || String(error)}`);
    }
  }

  viewBatchDetail(id: string): void {
    this.router.navigate(['/job-batch', id]);
  }

  async deleteBatch(event: Event, id: string): Promise<void> {
    event.stopPropagation();
    try {
      await this.jobBatchService.deleteBatch(id);
      this.batches.update(batches => batches.filter(b => b.batchId !== id));
    } catch (error: any) {
      await this.wails.logToFile(`[JobBatches] Failed to delete batch: ${error?.message || String(error)}`);
    }
  }

  isDone(batch: BatchStatus): boolean {
    return batch.pendingCount === 0 && batch.inProgressCount === 0;
  }
}
