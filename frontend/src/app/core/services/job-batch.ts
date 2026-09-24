import { Injectable, inject } from '@angular/core';
import { Wails, JobBatch, BatchStatus } from './wails';

@Injectable({
  providedIn: 'root'
})
export class JobBatchService {
  private wails = inject(Wails);

  async getAllBatches(limit: number, offset: number): Promise<JobBatch[]> {
    return this.wails.getAllJobBatches(limit, offset);
  }

  async getBatch(id: string): Promise<JobBatch> {
    return this.wails.getJobBatch(id);
  }

  async getBatchStatus(id: string): Promise<BatchStatus> {
    return this.wails.getJobBatchStatus(id);
  }

  async deleteBatch(id: string): Promise<void> {
    return this.wails.deleteJobBatch(id);
  }
}
