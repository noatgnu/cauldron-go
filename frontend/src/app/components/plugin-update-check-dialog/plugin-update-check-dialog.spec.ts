import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi, Mock } from 'vitest';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PluginUpdateCheckDialog } from './plugin-update-check-dialog';
import { Wails } from '../../core/services/wails';
import { MatDialog } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { of } from 'rxjs';

describe('PluginUpdateCheckDialog', () => {
  let component: PluginUpdateCheckDialog;
  let fixture: ComponentFixture<PluginUpdateCheckDialog>;
  let mockDialogRef: { close: Mock };
  let mockWails: { checkPluginUpdate: Mock; updatePluginToCommit: Mock; updatePluginToCommitForce: Mock; logToFile: Mock };
  let mockDialog: { open: Mock };

  const mockPlugin = {
    id: 1,
    definition: {
      plugin: {
        id: 'test-plugin',
        name: 'Test Plugin',
        description: 'A test plugin',
        version: '1.0.0'
      },
      runtime: {
        environments: ['python']
      }
    },
    repository: 'https://github.com/test/plugin',
    commitHash: 'abc1234',
    installSource: 'remote' as const,
    enabled: true,
    folderPath: '/test/path',
    scriptPath: '/test/path/script.py',
    installedAt: new Date()
  };

  beforeEach(async () => {
    mockDialogRef = { close: vi.fn() };
    mockWails = {
      checkPluginUpdate: vi.fn(),
      updatePluginToCommit: vi.fn(),
      updatePluginToCommitForce: vi.fn(),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    mockDialog = { open: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [PluginUpdateCheckDialog, NoopAnimationsModule],
      providers: [
        { provide: MAT_DIALOG_DATA, useValue: { plugin: mockPlugin } },
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: Wails, useValue: mockWails },
        { provide: MatDialog, useValue: mockDialog }
      ]
    })
    .overrideComponent(PluginUpdateCheckDialog, {
      add: {
        providers: [
          { provide: MatDialog, useValue: mockDialog }
        ]
      }
    })
    .compileComponents();

    fixture = TestBed.createComponent(PluginUpdateCheckDialog);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should check for updates on init', async () => {
    const mockUpdateResult = {
      has_update: true,
      current_commit: 'abc1234',
      latest_commit: 'def5678',
      recommended_commit: 'def5678',
      changelog_url: 'https://github.com/test/plugin/changelog'
    };

    mockWails.checkPluginUpdate.mockResolvedValue(mockUpdateResult);

    await component.ngOnInit();

    expect(mockWails.checkPluginUpdate).toHaveBeenCalledWith(
      mockPlugin.repository,
      mockPlugin.commitHash,
      null
    );
    expect(component.checking()).toBe(false);
    expect(component.updateResult()?.hasUpdate).toBe(true);
  });

  it('should surface a schema migration notice when the registry reports one', async () => {
    const mockUpdateResult = {
      has_update: true,
      current_commit: 'abc1234',
      latest_commit: 'def5678',
      recommended_commit: 'def5678',
      schema_migration_available: true
    };

    mockWails.checkPluginUpdate.mockResolvedValue(mockUpdateResult);

    await component.ngOnInit();

    expect(component.updateResult()?.schemaMigrationAvailable).toBe(true);
  });

  it('should default schemaMigrationAvailable to false when the registry omits it', async () => {
    const mockUpdateResult = {
      has_update: true,
      current_commit: 'abc1234',
      latest_commit: 'def5678',
      recommended_commit: 'def5678'
    };

    mockWails.checkPluginUpdate.mockResolvedValue(mockUpdateResult);

    await component.ngOnInit();

    expect(component.updateResult()?.schemaMigrationAvailable).toBe(false);
  });

  it('should handle update check error', async () => {
    mockWails.checkPluginUpdate.mockRejectedValue('Network error');

    await component.ngOnInit();

    expect(component.checking()).toBe(false);
    expect(component.updateResult()?.error).toContain('Network error');
  });

  it('should get short commit hash', () => {
    const result = component.getShortCommit('abc1234567890');
    expect(result).toBe('abc1234');
  });

  it('should open changelog in new window', () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    component.updateResult.set({
      hasUpdate: true,
      currentCommit: 'abc1234',
      latestCommit: 'def5678',
      changelogUrl: 'https://github.com/test/plugin/changelog'
    });

    component.openChangelog();

    expect(openSpy).toHaveBeenCalledWith('https://github.com/test/plugin/changelog', '_blank');
    openSpy.mockRestore();
  });

  it('should proceed with update', async () => {
    component.updateResult.set({
      hasUpdate: true,
      currentCommit: 'abc1234',
      latestCommit: 'def5678',
      recommendedCommit: 'def5678'
    });

    mockWails.updatePluginToCommit.mockResolvedValue(undefined);

    await component.proceedWithUpdate();

    expect(mockWails.updatePluginToCommit).toHaveBeenCalledWith(
      mockPlugin.repository,
      'def5678'
    );
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: true });
  });

  it('should handle local modifications during update', async () => {
    component.updateResult.set({
      hasUpdate: true,
      currentCommit: 'abc1234',
      latestCommit: 'def5678',
      recommendedCommit: 'def5678'
    });

    mockWails.updatePluginToCommit.mockRejectedValue('LOCAL_MODIFICATIONS');

    const mockConfirmDialogRef = {
      afterClosed: vi.fn().mockReturnValue(of(true))
    };
    mockDialog.open.mockReturnValue(mockConfirmDialogRef);
    mockWails.updatePluginToCommitForce.mockResolvedValue(undefined);

    await component.proceedWithUpdate();

    expect(mockDialog.open).toHaveBeenCalled();
    expect(mockWails.updatePluginToCommitForce).toHaveBeenCalledWith(
      mockPlugin.repository,
      'def5678',
      true
    );
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: true });
  });

  it('should close dialog without updating', () => {
    component.close();
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: false });
  });
});
