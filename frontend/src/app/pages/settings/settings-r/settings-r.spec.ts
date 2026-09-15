import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { MatDialog } from '@angular/material/dialog';
import { SettingsR } from './settings-r';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { DownloadPortableEnvDialogComponent } from '../../../components/download-portable-env-dialog/download-portable-env-dialog';

describe('SettingsR', () => {
  let component: SettingsR;
  let fixture: ComponentFixture<SettingsR>;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: { open: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    wailsMock = {
      getSettings: vi.fn().mockResolvedValue({}),
      detectREnvironments: vi.fn().mockResolvedValue([]),
      getActiveREnvironment: vi.fn().mockResolvedValue(null),
      getRVersion: vi.fn().mockResolvedValue('4.2.0'),
      getRenvEnvironments: vi.fn().mockResolvedValue([]),
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
      imports: [SettingsR, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(SettingsR);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('runs full initialization without error', async () => {
    fixture.detectChanges();
    await fixture.whenStable();

    expect(wailsMock.getSettings).toHaveBeenCalled();
    expect(wailsMock.getRVersion).toHaveBeenCalled();
    expect(wailsMock.detectREnvironments).toHaveBeenCalled();
    expect(wailsMock.getRenvEnvironments).toHaveBeenCalled();
  });

  it('does not warn "No R environments found" when a manually configured R path is verified', async () => {
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent;
    expect(text).not.toContain('No R environments found in PATH');
  });

  it('warns "No R environments found" when nothing is detected or configured', async () => {
    fixture.detectChanges();
    await fixture.whenStable();

    wailsMock.getRVersion.mockResolvedValue('');
    await component.loadVersion();
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent;
    expect(text).toContain('No R environments found in PATH');
  });

  it('opens the R portable env dialog set to r-portable and refreshes on close', () => {
    const detectSpy = vi.spyOn(component, 'detectAllREnvironments').mockResolvedValue(undefined);
    const loadVersionSpy = vi.spyOn(component, 'loadVersion').mockResolvedValue(undefined);

    component.downloadREnvironment();

    expect(dialogMock.open).toHaveBeenCalledWith(
      DownloadPortableEnvDialogComponent,
      expect.objectContaining({ width: '600px', disableClose: true })
    );

    const dialogRef = dialogMock.open.mock.results[0].value;
    expect(dialogRef.componentInstance.environment).toBe('r-portable');
    expect(detectSpy).toHaveBeenCalled();
    expect(loadVersionSpy).toHaveBeenCalled();
  });

  it('registers a manually browsed R path so it shows up in the detected list', async () => {
    wailsMock.openFile = vi.fn().mockResolvedValue('/opt/custom-r/bin/Rscript');
    wailsMock.setSetting = vi.fn().mockResolvedValue(undefined);
    wailsMock.registerManualREnvironment = vi.fn().mockResolvedValue({
      name: 'Manual R', path: '/opt/custom-r/bin/Rscript', type: 'manual', version: 'R 4.4.0'
    });
    const detectSpy = vi.spyOn(component, 'detectAllREnvironments').mockResolvedValue(undefined);
    const loadVersionSpy = vi.spyOn(component, 'loadVersion').mockResolvedValue(undefined);

    await component.browseR();

    expect(wailsMock.registerManualREnvironment).toHaveBeenCalledWith('/opt/custom-r/bin/Rscript');
    expect(detectSpy).toHaveBeenCalled();
    expect(loadVersionSpy).toHaveBeenCalled();
  });
});
