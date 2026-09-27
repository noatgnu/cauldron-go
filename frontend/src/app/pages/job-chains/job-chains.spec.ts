import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Router } from '@angular/router';
import { vi } from 'vitest';
import { JobChains } from './job-chains';
import { Wails } from '../../core/services/wails';
import { JobChainService } from '../../core/services/job-chain';

describe('JobChains', () => {
  let component: JobChains;
  let fixture: ComponentFixture<JobChains>;
  let wailsMock: any;
  let jobChainServiceMock: any;
  let routerMock: any;

  const mockChainStatus = (chainId: string, overrides: Partial<any> = {}) => ({
    chainId,
    label: `Chain ${chainId}`,
    createdAt: new Date().toISOString(),
    status: 'pending',
    stages: [],
    ...overrides
  });

  beforeEach(async () => {
    wailsMock = {
      jobUpdate: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    jobChainServiceMock = {
      getAllChains: vi.fn().mockResolvedValue([{ id: 'chain-1' }]),
      getChainStatus: vi.fn().mockResolvedValue(mockChainStatus('chain-1')),
      deleteChain: vi.fn().mockResolvedValue(undefined)
    };
    routerMock = {
      navigate: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [JobChains],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: JobChainService, useValue: jobChainServiceMock },
        { provide: Router, useValue: routerMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(JobChains);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads chains with their aggregate status', async () => {
    await component.loadChains();
    expect(component['chains']().length).toBe(1);
    expect(component['chains']()[0].chainId).toBe('chain-1');
  });

  it('navigates to a chain detail page', () => {
    component.viewChainDetail('chain-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/job-chain', 'chain-1']);
  });

  it('deletes a chain and removes it from the list', async () => {
    await component.loadChains();
    await component.deleteChain(new Event('click'), 'chain-1');
    expect(jobChainServiceMock.deleteChain).toHaveBeenCalledWith('chain-1');
    expect(component['chains']().length).toBe(0);
  });

  it('maps chain status to a chip color', () => {
    expect(component.statusColor('completed')).toBe('primary');
    expect(component.statusColor('running')).toBe('accent');
    expect(component.statusColor('failed')).toBe('warn');
    expect(component.statusColor('pending')).toBe('');
  });
});
