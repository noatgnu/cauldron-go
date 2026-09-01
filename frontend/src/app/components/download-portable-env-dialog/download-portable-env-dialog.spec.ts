import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef } from '@angular/material/dialog';
import { signal } from '@angular/core';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { DownloadPortableEnvDialogComponent } from './download-portable-env-dialog';
import { Wails } from '../../core/services/wails';

describe('DownloadPortableEnvDialogComponent', () => {
  let component: DownloadPortableEnvDialogComponent;
  let fixture: ComponentFixture<DownloadPortableEnvDialogComponent>;
  let wailsMock: {
    progress: ReturnType<typeof signal>;
    getPortableEnvironmentURL: ReturnType<typeof vi.fn>;
    getPortableEnvironmentPath: ReturnType<typeof vi.fn>;
    downloadPortableEnvironment: ReturnType<typeof vi.fn>;
    listAvailableRVersions: ReturnType<typeof vi.fn>;
    listInstalledRVersions: ReturnType<typeof vi.fn>;
    installRVersion: ReturnType<typeof vi.fn>;
    uninstallRVersion: ReturnType<typeof vi.fn>;
    getRPortablePath: ReturnType<typeof vi.fn>;
    setSetting: ReturnType<typeof vi.fn>;
    detectREnvironments: ReturnType<typeof vi.fn>;
    setActiveREnvironment: ReturnType<typeof vi.fn>;
    logToFile: ReturnType<typeof vi.fn>;
  };

  beforeEach(async () => {
    wailsMock = {
      progress: signal(null),
      getPortableEnvironmentURL: vi.fn().mockResolvedValue('http://example.com/env.tar.xz'),
      getPortableEnvironmentPath: vi.fn().mockResolvedValue(''),
      downloadPortableEnvironment: vi.fn().mockResolvedValue(undefined),
      listAvailableRVersions: vi.fn().mockResolvedValue([
        { version: '4.5.2', assetUrl: '', checksumUrl: '', assetFileName: '' }
      ]),
      listInstalledRVersions: vi.fn().mockResolvedValue([]),
      installRVersion: vi.fn().mockResolvedValue(undefined),
      uninstallRVersion: vi.fn().mockResolvedValue(undefined),
      getRPortablePath: vi.fn().mockResolvedValue('/fake/path/Rscript'),
      setSetting: vi.fn().mockResolvedValue(undefined),
      detectREnvironments: vi.fn().mockResolvedValue([]),
      setActiveREnvironment: vi.fn().mockResolvedValue(undefined),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };

    await TestBed.configureTestingModule({
      imports: [DownloadPortableEnvDialogComponent, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: MatDialogRef, useValue: { close: () => {} } }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(DownloadPortableEnvDialogComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    fixture.detectChanges();
    expect(component).toBeTruthy();
  });

  describe('python environment', () => {
    beforeEach(() => {
      component.environment = 'python';
      fixture.detectChanges();
    });

    it('fetches a download URL on init', () => {
      expect(wailsMock.getPortableEnvironmentURL).toHaveBeenCalled();
    });
  });

  describe('r-portable environment', () => {
    beforeEach(async () => {
      component.environment = 'r-portable';
      fixture.detectChanges();
      await fixture.whenStable();
    });

    it('loads available and installed R versions on init', () => {
      expect(wailsMock.listAvailableRVersions).toHaveBeenCalled();
      expect(wailsMock.listInstalledRVersions).toHaveBeenCalled();
      expect(component.availableRVersions.length).toBe(1);
      expect(component.selectedRVersion).toBe('4.5.2');
    });

    it('reports isInstalled correctly', () => {
      component.installedRVersions = ['4.5.2'];
      expect(component.isInstalled('4.5.2')).toBe(true);
      expect(component.isInstalled('4.4.2')).toBe(false);
    });

    it('installs the selected version and tracks step progress to completion', async () => {
      component.selectedRVersion = '4.5.2';

      const installPromise = component.installR();

      expect(wailsMock.installRVersion).toHaveBeenCalledWith('4.5.2');
      expect(component.installing).toBe(true);
      expect(component.steps.map(s => s.key)).toEqual(['download', 'verify', 'extract', 'install']);
      expect(component.steps[0].status).toBe('active');

      wailsMock.progress.set({
        type: 'install',
        id: 'r-portable-4.5.2',
        message: 'Downloading r-portable-4.5.2-ubuntu-22.04-x86_64.tar.xz...',
        percentage: 0,
        status: 'in_progress'
      });
      TestBed.flushEffects();
      expect(component.steps[0].status).toBe('active');

      wailsMock.progress.set({
        type: 'install',
        id: 'r-portable-4.5.2',
        message: 'Verifying checksum...',
        percentage: 0,
        status: 'in_progress'
      });
      TestBed.flushEffects();
      expect(component.steps[0].status).toBe('done');
      expect(component.steps[1].status).toBe('active');

      wailsMock.progress.set({
        type: 'install',
        id: 'r-portable-4.5.2',
        message: 'R 4.5.2 installed to /fake/path',
        percentage: 100,
        status: 'completed'
      });
      TestBed.flushEffects();

      await installPromise;
      await fixture.whenStable();

      expect(component.finished).toBe(true);
      expect(component.installing).toBe(false);
      expect(component.steps.every(s => s.status === 'done')).toBe(true);
      expect(wailsMock.getRPortablePath).toHaveBeenCalledWith('4.5.2');
      expect(wailsMock.setSetting).toHaveBeenCalledWith('rPath', '/fake/path/Rscript');
      expect(wailsMock.detectREnvironments).toHaveBeenCalled();
      expect(wailsMock.setActiveREnvironment).toHaveBeenCalledWith('/fake/path/Rscript');
    });

    it('records an error and stops installing when the install call rejects', async () => {
      wailsMock.installRVersion.mockRejectedValueOnce(new Error('boom'));
      component.selectedRVersion = '4.5.2';

      await component.installR();

      expect(component.installing).toBe(false);
      expect(component.errorMessage).toBe('boom');
      expect(component.steps.some(s => s.status === 'error')).toBe(true);
    });
  });
});
