import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { PluginExecute } from './plugin-execute';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { NotificationService } from '../../core/services/notification.service';
import { Wails } from '../../core/services/wails';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('PluginExecute', () => {
  let component: PluginExecute;
  let fixture: ComponentFixture<PluginExecute>;
  let activatedRouteMock: any;
  let routerMock: any;
  let pluginV2ServiceMock: any;
  let notificationMock: any;
  let wailsMock: any;

  beforeEach(async () => {
    activatedRouteMock = {
      paramMap: of({ get: () => '1' }),
      snapshot: {
        queryParamMap: { get: () => null }
      }
    };
    routerMock = {
      navigate: vi.fn()
    };
    pluginV2ServiceMock = {
      getPlugin: vi.fn().mockResolvedValue({ id: 1, definition: { plugin: { id: 'test' }, runtime: { environments: [] } } }),
      executePlugin: vi.fn().mockResolvedValue('job-1')
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    wailsMock = {
      bindingsUpdated: signal(0),
      getPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [PluginExecute, NoopAnimationsModule],
      providers: [
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: Wails, useValue: wailsMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginExecute);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
