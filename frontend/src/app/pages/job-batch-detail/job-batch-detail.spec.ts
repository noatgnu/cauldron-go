import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { JobBatchDetail } from './job-batch-detail';
import { Wails } from '../../core/services/wails';
import { JobBatchService } from '../../core/services/job-batch';

describe('JobBatchDetail', () => {
  let component: JobBatchDetail;
  let fixture: ComponentFixture<JobBatchDetail>;
  let wailsMock: any;
  let jobBatchServiceMock: any;
  let routerMock: any;
  let activatedRouteMock: any;
  let dialogMock: any;

  const mockBatchStatus = {
    batchId: 'batch-1',
    label: 'My Batch',
    pluginId: 'test-plugin',
    expectedCount: 2,
    totalJobs: 2,
    pendingCount: 1,
    inProgressCount: 1,
    completedCount: 0,
    failedCount: 0,
    createdAt: new Date().toISOString(),
    jobs: [
      { id: 'job-1', name: 'Job 1', status: 'pending', progress: 0, batchId: 'batch-1' },
      { id: 'job-2', name: 'Job 2', status: 'in_progress', progress: 50, batchId: 'batch-1' }
    ]
  };

  beforeEach(async () => {
    wailsMock = {
      jobUpdate: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    jobBatchServiceMock = {
      getBatchStatus: vi.fn().mockResolvedValue(mockBatchStatus),
      deleteBatch: vi.fn().mockResolvedValue(undefined)
    };
    routerMock = {
      navigate: vi.fn()
    };
    activatedRouteMock = {
      paramMap: of({ get: () => 'batch-1' })
    };
    dialogMock = {
      open: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [JobBatchDetail],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: JobBatchService, useValue: jobBatchServiceMock },
        { provide: Router, useValue: routerMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(JobBatchDetail);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create and load the batch from the route param', () => {
    expect(component).toBeTruthy();
    expect(jobBatchServiceMock.getBatchStatus).toHaveBeenCalledWith('batch-1');
    expect(component['batch']()?.label).toBe('My Batch');
  });

  it('navigates to a child job detail page', () => {
    component.viewJob('job-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/jobs', 'job-1']);
  });

  it('deletes the batch and navigates back to the list when confirmed', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(true) });
    await component.deleteBatch();
    expect(jobBatchServiceMock.deleteBatch).toHaveBeenCalledWith('batch-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/job-batches']);
  });

  it('does not delete the batch when the confirmation is cancelled', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(false) });
    await component.deleteBatch();
    expect(jobBatchServiceMock.deleteBatch).not.toHaveBeenCalled();
  });
});
