import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

import { SettingsBackup } from './settings-backup';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';

describe('SettingsBackup', () => {
  let component: SettingsBackup;
  let fixture: ComponentFixture<SettingsBackup>;
  let wailsMock: any;
  let notificationMock: any;

  beforeEach(async () => {
    wailsMock = {
      saveFileDialog: vi.fn().mockResolvedValue('/tmp/backup.json'),
      openFile: vi.fn().mockResolvedValue('/tmp/backup.json'),
      createSettingsBackup: vi.fn().mockResolvedValue({
        createdAt: new Date().toISOString(),
        settingsCount: 3,
        pluginsCount: 2,
        envVarsCount: 0,
        includesSecrets: false
      }),
      previewSettingsBackup: vi.fn().mockResolvedValue({
        createdAt: new Date().toISOString(),
        settingsCount: 3,
        pluginsCount: 2,
        envVarsCount: 0,
        includesSecrets: false
      }),
      restoreSettingsBackup: vi.fn().mockResolvedValue({
        settingsRestored: 3,
        pluginsInstalled: [],
        pluginsSkipped: ['builtin-plugin'],
        pluginsFailed: {},
        envVarsRestored: 0
      })
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showWarning: vi.fn(),
      showInfo: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [SettingsBackup, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
      .compileComponents();

    fixture = TestBed.createComponent(SettingsBackup);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('backupNow writes a backup and stores the summary', async () => {
    await component.backupNow();

    expect(wailsMock.createSettingsBackup).toHaveBeenCalledWith('/tmp/backup.json', false);
    expect(component.lastBackup()?.settingsCount).toBe(3);
    expect(notificationMock.showSuccess).toHaveBeenCalled();
  });

  it('backupNow does nothing when the save dialog is cancelled', async () => {
    wailsMock.saveFileDialog.mockResolvedValue('');

    await component.backupNow();

    expect(wailsMock.createSettingsBackup).not.toHaveBeenCalled();
  });

  it('failedPluginEntries flattens the pluginsFailed map', () => {
    const entries = component.failedPluginEntries({
      settingsRestored: 0,
      pluginsInstalled: [],
      pluginsSkipped: [],
      pluginsFailed: { 'plugin-a': 'boom' },
      envVarsRestored: 0
    } as any);

    expect(entries).toEqual([{ id: 'plugin-a', message: 'boom' }]);
  });
});
