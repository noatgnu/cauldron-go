import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Plugins } from './plugins';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('Plugins', () => {
  let component: Plugins;
  let fixture: ComponentFixture<Plugins>;
  let wailsMock: any;
  let notificationMock: any;
  let routerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      getPlugins: vi.fn().mockResolvedValue([]),
      getAllPluginEnvironmentBindings: vi.fn().mockResolvedValue([]),
      getPluginsDirectory: vi.fn().mockResolvedValue(''),
      getPluginsV2: vi.fn().mockResolvedValue([]),
      bindingsUpdated$: of(undefined),
      progress$: of(null),
      jobUpdate$: of(null),
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
      imports: [Plugins, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: Router, useValue: routerMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Plugins);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
