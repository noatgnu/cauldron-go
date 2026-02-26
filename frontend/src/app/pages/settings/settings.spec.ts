import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router, NavigationEnd } from '@angular/router';
import { Settings } from './settings';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('Settings', () => {
  let component: Settings;
  let fixture: ComponentFixture<Settings>;
  let activatedRouteMock: any;
  let routerMock: any;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    activatedRouteMock = {
      paramMap: of({ get: () => 'general' }),
      params: of({ section: 'general' }),
      firstChild: null
    };
    routerMock = {
      navigate: vi.fn(),
      events: of(new NavigationEnd(0, '/', '/'))
    };
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [Settings, NoopAnimationsModule],
      providers: [
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(Settings);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
