import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Jobs } from './jobs';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

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
      queueStatus$: of({ status: 'running' }),
      jobUpdate$: of(null),
      progress$: of(null),
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
});
