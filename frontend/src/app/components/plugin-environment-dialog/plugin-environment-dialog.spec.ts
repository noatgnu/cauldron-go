import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PluginEnvironmentDialog } from './plugin-environment-dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PluginEnvironmentDialog', () => {
  let component: PluginEnvironmentDialog;
  let fixture: ComponentFixture<PluginEnvironmentDialog>;
  let dialogRefSpy: any;
  let wailsMock: any;
  let notificationMock: any;

  function createComponent() {
    fixture = TestBed.createComponent(PluginEnvironmentDialog);
    component = fixture.componentInstance;
  }

  beforeEach(async () => {
    dialogRefSpy = {
      close: vi.fn()
    };
    wailsMock = {
      progress: signal(null),
      getPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      detectPythonEnvironments: vi.fn().mockResolvedValue([]),
      detectREnvironments: vi.fn().mockResolvedValue([]),
      getActivePythonEnvironment: vi.fn().mockResolvedValue(null),
      getActiveREnvironment: vi.fn().mockResolvedValue(null),
      getCustomEnvVars: vi.fn().mockResolvedValue([]),
      isUvAvailable: vi.fn().mockResolvedValue(false),
      listUvManagedPythons: vi.fn().mockResolvedValue([]),
      getDefaultVenvPath: vi.fn().mockResolvedValue('/tmp/venv-test'),
      getVirtualEnvironments: vi.fn().mockResolvedValue([]),
      bindPluginToEnvironment: vi.fn().mockResolvedValue(undefined),
      createPythonVirtualEnv: vi.fn().mockResolvedValue(undefined),
      createUvVirtualEnv: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [PluginEnvironmentDialog, NoopAnimationsModule],
      providers: [
        { provide: MatDialogRef, useValue: dialogRefSpy },
        {
          provide: MAT_DIALOG_DATA,
          useValue: {
            pluginId: 'test',
            pluginName: 'Test',
            runtimeEnvironments: []
          }
        },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
    .compileComponents();
  });

  it('should create', async () => {
    createComponent();
    await fixture.whenStable();
    expect(component).toBeTruthy();
  });

  it('does not offer uv-managed Python when uv is unavailable', async () => {
    createComponent();
    await fixture.whenStable();
    expect(component.uvAvailable()).toBe(false);
    expect(component.uvManagedPythons()).toEqual([]);
  });

  it('loads uv-managed Python versions when uv is available', async () => {
    wailsMock.isUvAvailable.mockResolvedValue(true);
    wailsMock.listUvManagedPythons.mockResolvedValue([
      { version: '3.12.1', path: '/uv/pythons/3.12.1', implementation: 'cpython' }
    ]);
    createComponent();
    await fixture.whenStable();

    expect(component.uvAvailable()).toBe(true);
    expect(component.uvManagedPythons()).toEqual([
      { version: '3.12.1', path: '/uv/pythons/3.12.1', implementation: 'cpython' }
    ]);
  });

  it('creates a uv virtual env when a uv-managed Python is selected', async () => {
    wailsMock.isUvAvailable.mockResolvedValue(true);
    wailsMock.listUvManagedPythons.mockResolvedValue([
      { version: '3.12.1', path: '/uv/pythons/3.12.1', implementation: 'cpython' }
    ]);
    createComponent();
    await fixture.whenStable();

    component.selectedBasePython.set('uv:3.12.1');
    await component.confirmVenvCreation();

    expect(wailsMock.createUvVirtualEnv).toHaveBeenCalledWith('3.12.1', '/tmp/venv-test', 'test');
    expect(wailsMock.createPythonVirtualEnv).not.toHaveBeenCalled();
  });

  it('creates a regular virtual env when a detected Python path is selected', async () => {
    createComponent();
    await fixture.whenStable();

    component.selectedBasePython.set('/usr/bin/python3');
    await component.confirmVenvCreation();

    expect(wailsMock.createPythonVirtualEnv).toHaveBeenCalledWith('/usr/bin/python3', '/tmp/venv-test', 'test');
    expect(wailsMock.createUvVirtualEnv).not.toHaveBeenCalled();
  });
});
