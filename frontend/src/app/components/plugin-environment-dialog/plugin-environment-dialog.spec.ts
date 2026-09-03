import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialog, MatDialogRef } from '@angular/material/dialog';
import { of } from 'rxjs';
import { PluginEnvironmentDialog } from './plugin-environment-dialog';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

function pendingMigration(changes: any[], opts: { totalOperations?: number; large?: boolean } = {}) {
  return {
    changes,
    totalOperations: opts.totalOperations ?? changes.length,
    large: opts.large ?? false
  };
}

describe('PluginEnvironmentDialog', () => {
  let component: PluginEnvironmentDialog;
  let fixture: ComponentFixture<PluginEnvironmentDialog>;
  let dialogRefSpy: any;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: any;

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
      createUvVirtualEnv: vi.fn().mockResolvedValue(undefined),
      getPendingEnvVarMigration: vi.fn().mockResolvedValue(null),
      applyPendingEnvVarMigration: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({
        componentInstance: {},
        afterClosed: () => of(true)
      })
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

    // PluginEnvironmentDialog's own MatDialogModule import would otherwise shadow a plain providers: override.
    TestBed.overrideProvider(MatDialog, { useValue: dialogMock });
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

  const fakePlugin = {
    id: 42,
    definition: { execution: { envVariables: [{ name: 'API_KEY', type: 'text' }] } }
  };

  function useDialogDataWithPlugin() {
    TestBed.overrideProvider(MAT_DIALOG_DATA, {
      useValue: { pluginId: 'test', pluginName: 'Test', runtimeEnvironments: [], plugin: fakePlugin }
    });
  }

  it('does not show a migration banner when nothing is pending', async () => {
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();

    expect(component.pendingEnvVarChanges()).toEqual([]);
  });

  it('loads pending env var changes for the bound plugin on init', async () => {
    wailsMock.getPendingEnvVarMigration.mockResolvedValue(
      pendingMigration([{ from: 'API_KEY', to: 'UNIPROT_API_KEY', removed: false }])
    );
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(wailsMock.getPendingEnvVarMigration).toHaveBeenCalledWith(42);
    expect(component.pendingEnvVarChanges()).toEqual([
      { from: 'API_KEY', to: 'UNIPROT_API_KEY', removed: false }
    ]);
    expect(component.pendingEnvVarMigration()?.large).toBe(false);
  });

  it('dismissing the banner hides it without calling apply', async () => {
    wailsMock.getPendingEnvVarMigration.mockResolvedValue(pendingMigration([{ from: 'A', to: 'B', removed: false }]));
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();

    component.dismissEnvVarMigration();

    expect(component.envVarMigrationDismissed()).toBe(true);
    expect(wailsMock.applyPendingEnvVarMigration).not.toHaveBeenCalled();
  });

  it('applying a normal-sized migration calls the backend with confirmedLarge=false and skips the confirm dialog', async () => {
    wailsMock.getPendingEnvVarMigration
      .mockResolvedValueOnce(pendingMigration([{ from: 'A', to: 'B', removed: false }]))
      .mockResolvedValueOnce(pendingMigration([]));
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();

    await component.applyEnvVarMigration();

    expect(dialogMock.open).not.toHaveBeenCalled();
    expect(wailsMock.applyPendingEnvVarMigration).toHaveBeenCalledWith(42, false);
    expect(component.pendingEnvVarChanges()).toEqual([]);
  });

  it('applying a large migration shows a confirm dialog and proceeds with confirmedLarge=true when confirmed', async () => {
    wailsMock.getPendingEnvVarMigration
      .mockResolvedValueOnce(pendingMigration([{ from: 'A', to: 'B', removed: false }], { totalOperations: 80, large: true }))
      .mockResolvedValueOnce(pendingMigration([]));
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    await component.applyEnvVarMigration();

    expect(dialogMock.open).toHaveBeenCalled();
    expect(wailsMock.applyPendingEnvVarMigration).toHaveBeenCalledWith(42, true);
  });

  it('applying a large migration does nothing if the confirm dialog is cancelled', async () => {
    dialogMock.open.mockReturnValue({ componentInstance: {}, afterClosed: () => of(false) });
    wailsMock.getPendingEnvVarMigration.mockResolvedValue(
      pendingMigration([{ from: 'A', to: 'B', removed: false }], { totalOperations: 80, large: true })
    );
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve, 0));

    await component.applyEnvVarMigration();

    expect(dialogMock.open).toHaveBeenCalled();
    expect(wailsMock.applyPendingEnvVarMigration).not.toHaveBeenCalled();
  });

  it('surfaces an error notification when applying the migration fails', async () => {
    wailsMock.getPendingEnvVarMigration.mockResolvedValue(pendingMigration([{ from: 'A', to: 'B', removed: false }]));
    wailsMock.applyPendingEnvVarMigration.mockRejectedValue(new Error('boom'));
    useDialogDataWithPlugin();
    createComponent();
    await fixture.whenStable();

    await component.applyEnvVarMigration();

    expect(notificationMock.showError).toHaveBeenCalled();
    expect(component.applyingEnvVarMigration()).toBe(false);
  });
});
