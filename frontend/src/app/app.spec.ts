import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
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
    loadingServiceMock = { loading: signal({ isLoading: false, message: '' }) };
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
      bindingsUpdated: signal(0),
      progress: signal(null),
      getSettings: vi.fn().mockResolvedValue({ autoCheckForUpdates: true }),
      checkForUpdate: vi.fn().mockResolvedValue({ available: false }),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    pluginV2ServiceMock = {
      getAllPlugins: vi.fn().mockResolvedValue([]),
      pluginListVersion: signal(0)
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

  it('moves focus to the main content region after navigation', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    const mainContent: HTMLElement = fixture.nativeElement.querySelector('#main-content');
    expect(mainContent).not.toBeNull();
    expect(mainContent.getAttribute('tabindex')).toBe('-1');
    expect(document.activeElement).toBe(mainContent);
  });

  it('opens the update dialog on startup when auto-check is enabled and an update is available', async () => {
    wailsMock.getSettings.mockResolvedValue({ autoCheckForUpdates: true });
    wailsMock.checkForUpdate.mockResolvedValue({ available: true, latestVersion: 'v0.1.0' });

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(dialogMock.open).toHaveBeenCalled();
  });

  it('does not check for updates on startup when auto-check is disabled', async () => {
    wailsMock.getSettings.mockResolvedValue({ autoCheckForUpdates: false });

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(wailsMock.checkForUpdate).not.toHaveBeenCalled();
    expect(dialogMock.open).not.toHaveBeenCalled();
  });

  it('does not open the dialog on startup when already up to date', async () => {
    wailsMock.getSettings.mockResolvedValue({ autoCheckForUpdates: true });
    wailsMock.checkForUpdate.mockResolvedValue({ available: false });

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(dialogMock.open).not.toHaveBeenCalled();
  });

  it('silently swallows startup update-check failures without showing the user an error', async () => {
    wailsMock.getSettings.mockResolvedValue({ autoCheckForUpdates: true });
    wailsMock.checkForUpdate.mockRejectedValue(new Error('network down'));

    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(dialogMock.open).not.toHaveBeenCalled();
    expect(notificationMock.showError).not.toHaveBeenCalled();
    expect(wailsMock.logToFile).toHaveBeenCalled();
  });
});
