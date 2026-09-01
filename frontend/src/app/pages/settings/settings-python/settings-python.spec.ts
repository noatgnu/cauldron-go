import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { MatDialog } from '@angular/material/dialog';
import { SettingsPython } from './settings-python';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { DownloadPortableEnvDialogComponent } from '../../../components/download-portable-env-dialog/download-portable-env-dialog';

describe('SettingsPython', () => {
  let component: SettingsPython;
  let fixture: ComponentFixture<SettingsPython>;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: { open: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      detectPythonEnvironments: vi.fn().mockResolvedValue([]),
      getActivePythonEnvironment: vi.fn().mockResolvedValue(null),
      getPythonVersion: vi.fn().mockResolvedValue('3.10.0'),
      checkDockerVersion: vi.fn().mockResolvedValue('20.10.0'),
      getVirtualEnvironments: vi.fn().mockResolvedValue([]),
      getAllPluginEnvironmentBindings: vi.fn().mockResolvedValue([]),
      getPluginsV2: vi.fn().mockResolvedValue([]),
      bindingsUpdated: signal(0),
      progress: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({
        componentInstance: {},
        afterClosed: () => of(undefined)
      })
    };

    await TestBed.configureTestingModule({
      imports: [SettingsPython, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsPython);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('opens the portable env dialog set to python and refreshes on close', () => {
    const detectSpy = vi.spyOn(component, 'detectAllPythonEnvironments').mockResolvedValue(undefined);
    const loadVersionSpy = vi.spyOn(component, 'loadVersion').mockResolvedValue(undefined);

    component.downloadPythonEnvironment();

    expect(dialogMock.open).toHaveBeenCalledWith(
      DownloadPortableEnvDialogComponent,
      expect.objectContaining({ width: '600px', disableClose: true })
    );

    const dialogRef = dialogMock.open.mock.results[0].value;
    expect(dialogRef.componentInstance.environment).toBe('python');
    expect(detectSpy).toHaveBeenCalled();
    expect(loadVersionSpy).toHaveBeenCalled();
  });
});
