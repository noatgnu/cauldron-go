import { TestBed } from '@angular/core/testing';
import { App } from './app';
import { Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { ProtocolHandlerService } from './core/services/protocol-handler.service';
import { LoadingService } from './services/loading';
import { MatDialog } from '@angular/material/dialog';
import { NotificationService } from './core/services/notification.service';
import { ThemeService } from './core/services/theme.service';
import { Wails } from './core/services/wails';
import { PluginV2Service } from './core/services/plugin-v2';
import { vi, beforeAll } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('App', () => {
  let routerMock: any;
  let protocolHandlerMock: any;
  let loadingServiceMock: any;
  let dialogMock: any;
  let notificationMock: any;
  let themeServiceMock: any;
  let wailsMock: any;
  let pluginV2ServiceMock: any;

  beforeAll(() => {
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });
  });

  beforeEach(async () => {
    routerMock = { 
      navigate: vi.fn(),
      events: of(new NavigationEnd(0, '/', '/')),
      url: '/'
    };
    protocolHandlerMock = {};
    loadingServiceMock = { loading$: of(false) };
    dialogMock = { open: vi.fn() };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };
    themeServiceMock = {
      theme: vi.fn().mockReturnValue('system'),
      isDark: vi.fn().mockReturnValue(false)
    };
    wailsMock = {
      isWails: false,
      bindingsUpdated$: of(undefined),
      progress$: of(null)
    };
    pluginV2ServiceMock = {
      getAllPlugins: vi.fn().mockResolvedValue([])
    };

    await TestBed.configureTestingModule({
      imports: [App, NoopAnimationsModule],
      providers: [
        { provide: Router, useValue: routerMock },
        { provide: ProtocolHandlerService, useValue: protocolHandlerMock },
        { provide: LoadingService, useValue: loadingServiceMock },
        { provide: MatDialog, useValue: dialogMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: ThemeService, useValue: themeServiceMock },
        { provide: Wails, useValue: wailsMock },
        { provide: PluginV2Service, useValue: pluginV2ServiceMock },
        { provide: ActivatedRoute, useValue: { paramMap: of({ get: () => null }) } }
      ]
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });
});
