import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { Jobs } from './jobs';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('Jobs', () => {
  let component: Jobs;
  let fixture: ComponentFixture<Jobs>;
  let wailsMock: any;
  let notificationMock: any;
  let routerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      getAllJobs: vi.fn().mockResolvedValue([]),
      getJobQueueStatus: vi.fn().mockResolvedValue({ status: 'running' }),
      queueStatus: signal({ status: 'running' }),
      jobUpdate: signal(null),
      progress: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    routerMock = {
      navigate: vi.fn()
    };
    dialogMock = {
      open: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [Jobs, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: Router, useValue: routerMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Jobs);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('keyboard access to job rows', () => {
    const job = { id: 'job-1', name: 'Test Job', type: 'pca-analysis', status: 'completed', createdAt: Date.now() };

    beforeEach(async () => {
      wailsMock.getAllJobs.mockResolvedValue([job]);
      await component.loadJobs();
      fixture.detectChanges();
    });

    function nameCell(): HTMLElement {
      return fixture.nativeElement.querySelector('td.clickable');
    }

    it('marks the job name cell as keyboard-focusable with a descriptive label', () => {
      const cell = nameCell();
      expect(cell.getAttribute('tabindex')).toBe('0');
      expect(cell.getAttribute('aria-label')).toBe('Open job Test Job');
    });

    it('opens the job on Enter', () => {
      nameCell().dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
      expect(routerMock.navigate).toHaveBeenCalledWith(['/jobs', job.id]);
    });

    it('opens the job on Space', () => {
      nameCell().dispatchEvent(new KeyboardEvent('keydown', { key: ' ', bubbles: true }));
      expect(routerMock.navigate).toHaveBeenCalledWith(['/jobs', job.id]);
    });
  });
});
