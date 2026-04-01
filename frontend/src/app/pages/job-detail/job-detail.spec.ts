import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { JobDetail } from './job-detail';
import { Wails } from '../../core/services/wails';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { NotificationService } from '../../core/services/notification.service';
import { PlotlyModule } from 'angular-plotly.js';
import { PlotlyMock } from '../../core/mocks/plotly-mock';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('JobDetail', () => {
  let component: JobDetail;
  let fixture: ComponentFixture<JobDetail>;
  let wailsMock: any;
  let activatedRouteMock: any;
  let routerMock: any;
  let dialogMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    wailsMock = {
      getJob: vi.fn().mockResolvedValue({ id: '1', status: 'completed' }),
      getJobExecutionLog: vi.fn().mockResolvedValue(''),
      listJobOutputFiles: vi.fn().mockResolvedValue([]),
      jobUpdate: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    activatedRouteMock = {
      paramMap: of({ get: () => '1' }),
      params: of({ id: '1' })
    };
    routerMock = {
      navigate: vi.fn()
    };
    dialogMock = {
      open: vi.fn()
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [
        JobDetail, 
        PlotlyModule.forRoot(PlotlyMock), 
        NoopAnimationsModule
      ],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: MatDialog, useValue: dialogMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(JobDetail);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
