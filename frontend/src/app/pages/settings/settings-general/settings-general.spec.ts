import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialog } from '@angular/material/dialog';
import { SettingsGeneral } from './settings-general';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('SettingsGeneral', () => {
  let component: SettingsGeneral;
  let fixture: ComponentFixture<SettingsGeneral>;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      getRemotePlugins: vi.fn().mockResolvedValue([]),
      logToFile: vi.fn().mockResolvedValue(undefined),
      setSetting: vi.fn().mockResolvedValue(undefined),
      checkForUpdate: vi.fn().mockResolvedValue({ available: false })
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };
    dialogMock = { open: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [SettingsGeneral, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsGeneral);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('defaults debug mode to off when not set', async () => {
    await component.loadSettings();
    expect(component['debugMode']()).toBe(false);
  });

  it('loads debug mode from settings', async () => {
    wailsMock.getSettings.mockResolvedValue({ debugMode: true });
    await component.loadSettings();
    expect(component['debugMode']()).toBe(true);
  });

  it('persists debug mode changes', async () => {
    await component.setDebugMode(true);
    expect(component['debugMode']()).toBe(true);
    expect(wailsMock.setSetting).toHaveBeenCalledWith('debugMode', true);
  });

  it('defaults auto-check for updates to on when not explicitly false', async () => {
    wailsMock.getSettings.mockResolvedValue({});
    await component.loadSettings();
    expect(component['autoCheckForUpdates']()).toBe(true);
  });

  it('loads an explicitly disabled auto-check setting', async () => {
    wailsMock.getSettings.mockResolvedValue({ autoCheckForUpdates: false });
    await component.loadSettings();
    expect(component['autoCheckForUpdates']()).toBe(false);
  });

  it('persists auto-check for updates changes', async () => {
    await component.setAutoCheckForUpdates(false);
    expect(component['autoCheckForUpdates']()).toBe(false);
    expect(wailsMock.setSetting).toHaveBeenCalledWith('autoCheckForUpdates', false);
  });

  it('opens the update dialog when Check Now finds an available update', async () => {
    wailsMock.checkForUpdate.mockResolvedValue({ available: true, latestVersion: 'v0.1.0' });
    await component.checkForUpdateNow();
    expect(dialogMock.open).toHaveBeenCalled();
    expect(notificationMock.showInfo).not.toHaveBeenCalled();
  });

  it('shows an up-to-date notification when Check Now finds no update', async () => {
    wailsMock.checkForUpdate.mockResolvedValue({ available: false });
    await component.checkForUpdateNow();
    expect(dialogMock.open).not.toHaveBeenCalled();
    expect(notificationMock.showInfo).toHaveBeenCalled();
  });

  it('shows an error notification when Check Now fails', async () => {
    wailsMock.checkForUpdate.mockRejectedValue(new Error('network down'));
    await component.checkForUpdateNow();
    expect(notificationMock.showError).toHaveBeenCalled();
    expect(component['checkingForUpdate']()).toBe(false);
  });

  it('loads allowed repo hosts from settings', async () => {
    wailsMock.getSettings.mockResolvedValue({ allowedRepoHosts: ['git.internal.example'] });
    await component.loadSettings();
    expect(component['allowedRepoHosts']()).toEqual(['git.internal.example']);
  });

  it('adds a new allowed repo host, lowercased and trimmed', async () => {
    component['newRepoHost'].set('  Git.Internal.Example  ');
    await component.addAllowedRepoHost();
    expect(component['allowedRepoHosts']()).toEqual(['git.internal.example']);
    expect(component['newRepoHost']()).toBe('');
    expect(wailsMock.setSetting).toHaveBeenCalledWith('allowedRepoHosts', ['git.internal.example']);
  });

  it('does not add a duplicate host', async () => {
    component['allowedRepoHosts'].set(['git.internal.example']);
    component['newRepoHost'].set('git.internal.example');
    await component.addAllowedRepoHost();
    expect(component['allowedRepoHosts']()).toEqual(['git.internal.example']);
    expect(wailsMock.setSetting).not.toHaveBeenCalled();
  });

  it('ignores an empty host', async () => {
    component['newRepoHost'].set('   ');
    await component.addAllowedRepoHost();
    expect(component['allowedRepoHosts']()).toEqual([]);
    expect(wailsMock.setSetting).not.toHaveBeenCalled();
  });

  it('removes an allowed repo host', async () => {
    component['allowedRepoHosts'].set(['git.internal.example', 'gitea.internal.example']);
    await component.removeAllowedRepoHost('git.internal.example');
    expect(component['allowedRepoHosts']()).toEqual(['gitea.internal.example']);
    expect(wailsMock.setSetting).toHaveBeenCalledWith('allowedRepoHosts', ['gitea.internal.example']);
  });

  it('defaults restrict-to-allowlist to off when not set', async () => {
    await component.loadSettings();
    expect(component['restrictRepoHostsToAllowlist']()).toBe(false);
  });

  it('loads restrict-to-allowlist from settings', async () => {
    wailsMock.getSettings.mockResolvedValue({ restrictRepoHostsToAllowlist: true });
    await component.loadSettings();
    expect(component['restrictRepoHostsToAllowlist']()).toBe(true);
  });

  it('persists restrict-to-allowlist changes', async () => {
    await component.setRestrictRepoHostsToAllowlist(true);
    expect(component['restrictRepoHostsToAllowlist']()).toBe(true);
    expect(wailsMock.setSetting).toHaveBeenCalledWith('restrictRepoHostsToAllowlist', true);
  });
});
