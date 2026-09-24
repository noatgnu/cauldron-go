import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Router } from '@angular/router';
import { vi } from 'vitest';
import { JobBatches } from './job-batches';
import { Wails } from '../../core/services/wails';
import { JobBatchService } from '../../core/services/job-batch';

describe('JobBatches', () => {
  let component: JobBatches;
  let fixture: ComponentFixture<JobBatches>;
  let wailsMock: any;
  let jobBatchServiceMock: any;
  let routerMock: any;

  const mockBatchStatus = (batchId: string, overrides: Partial<any> = {}) => ({
    batchId,
    label: `Batch ${batchId}`,
    pluginId: 'test-plugin',
    expectedCount: 2,
    totalJobs: 2,
    pendingCount: 2,
    inProgressCount: 0,
    completedCount: 0,
    failedCount: 0,
    createdAt: new Date().toISOString(),
    jobs: [],
    ...overrides
  });

  beforeEach(async () => {
    wailsMock = {
      jobUpdate: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    jobBatchServiceMock = {
      getAllBatches: vi.fn().mockResolvedValue([{ id: 'batch-1' }]),
      getBatchStatus: vi.fn().mockResolvedValue(mockBatchStatus('batch-1')),
      deleteBatch: vi.fn().mockResolvedValue(undefined)
    };
    routerMock = {
      navigate: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [JobBatches],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: JobBatchService, useValue: jobBatchServiceMock },
        { provide: Router, useValue: routerMock }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(JobBatches);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads batches with their aggregate status', async () => {
    await component.loadBatches();
    expect(component['batches']().length).toBe(1);
    expect(component['batches']()[0].batchId).toBe('batch-1');
  });

  it('navigates to a batch detail page', () => {
    component.viewBatchDetail('batch-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/job-batch', 'batch-1']);
  });

  it('deletes a batch and removes it from the list', async () => {
    await component.loadBatches();
    const event = new Event('click');
    await component.deleteBatch(event, 'batch-1');
    expect(jobBatchServiceMock.deleteBatch).toHaveBeenCalledWith('batch-1');
    expect(component['batches']().length).toBe(0);
  });

  it('reports a batch as done only when nothing is pending or in progress', () => {
    expect(component.isDone(mockBatchStatus('b', { pendingCount: 1 }))).toBe(false);
    expect(component.isDone(mockBatchStatus('b', { inProgressCount: 1 }))).toBe(false);
    expect(component.isDone(mockBatchStatus('b', { pendingCount: 0, inProgressCount: 0 }))).toBe(true);
  });
});
