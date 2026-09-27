import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { JobChainDetail } from './job-chain-detail';
import { Wails } from '../../core/services/wails';
import { JobChainService } from '../../core/services/job-chain';

describe('JobChainDetail', () => {
  let component: JobChainDetail;
  let fixture: ComponentFixture<JobChainDetail>;
  let wailsMock: any;
  let jobChainServiceMock: any;
  let routerMock: any;
  let activatedRouteMock: any;
  let dialogMock: any;

  const mockChainStatus = {
    chainId: 'chain-1',
    label: 'My Chain',
    createdAt: new Date().toISOString(),
    status: 'running',
    stages: [
      { stageIndex: 0, pluginId: 1, status: 'completed', job: { id: 'job-1', name: 'Plugin A', createdAt: new Date().toISOString() } },
      { stageIndex: 1, pluginId: 2, status: 'running', job: { id: 'job-2', name: 'Plugin B', createdAt: new Date().toISOString() } }
    ]
  };

  beforeEach(async () => {
    wailsMock = {
      jobUpdate: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    jobChainServiceMock = {
      getChainStatus: vi.fn().mockResolvedValue(mockChainStatus),
      deleteChain: vi.fn().mockResolvedValue(undefined)
    };
    routerMock = {
      navigate: vi.fn()
    };
    activatedRouteMock = {
      paramMap: of({ get: () => 'chain-1' })
    };
    dialogMock = {
      open: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [JobChainDetail],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: JobChainService, useValue: jobChainServiceMock },
        { provide: Router, useValue: routerMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(JobChainDetail);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create and load the chain from the route param', () => {
    expect(component).toBeTruthy();
    expect(jobChainServiceMock.getChainStatus).toHaveBeenCalledWith('chain-1');
    expect(component['chain']()?.label).toBe('My Chain');
  });

  it('labels a stage with its job name when it has run', () => {
    const label = component.stageLabel(mockChainStatus.stages[0]);
    expect(label).toBe('Stage 1: Plugin A');
  });

  it('labels a stage without a job generically', () => {
    const label = component.stageLabel({ stageIndex: 2, status: 'pending' });
    expect(label).toBe('Stage 3');
  });

  it('navigates to a stage job detail page', () => {
    component.viewJob('job-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/jobs', 'job-1']);
  });

  it('deletes the chain and navigates back to the list when confirmed', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(true) });
    await component.deleteChain();
    expect(jobChainServiceMock.deleteChain).toHaveBeenCalledWith('chain-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/job-chains']);
  });

  it('does not delete the chain when the confirmation is cancelled', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(false) });
    await component.deleteChain();
    expect(jobChainServiceMock.deleteChain).not.toHaveBeenCalled();
  });
});
