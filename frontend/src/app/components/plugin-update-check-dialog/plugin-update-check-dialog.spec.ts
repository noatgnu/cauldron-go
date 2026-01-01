import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { PluginUpdateCheckDialog } from './plugin-update-check-dialog';
import { Wails } from '../../core/services/wails';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';

describe('PluginUpdateCheckDialog', () => {
  let component: PluginUpdateCheckDialog;
  let fixture: ComponentFixture<PluginUpdateCheckDialog>;
  let mockDialogRef: jasmine.SpyObj<MatDialogRef<PluginUpdateCheckDialog>>;
  let mockWails: jasmine.SpyObj<Wails>;
  let mockDialog: jasmine.SpyObj<MatDialog>;

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
    mockDialogRef = jasmine.createSpyObj('MatDialogRef', ['close']);
    mockWails = jasmine.createSpyObj('Wails', ['checkPluginUpdate', 'updatePluginToCommit', 'updatePluginWithForce', 'logToFile']);
    mockDialog = jasmine.createSpyObj('MatDialog', ['open']);

    mockWails.logToFile.and.returnValue(Promise.resolve());

    await TestBed.configureTestingModule({
      imports: [PluginUpdateCheckDialog],
      providers: [
        { provide: MAT_DIALOG_DATA, useValue: { plugin: mockPlugin } },
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: Wails, useValue: mockWails },
        { provide: MatDialog, useValue: mockDialog }
      ]
    }).compileComponents();

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

    mockWails.checkPluginUpdate.and.returnValue(Promise.resolve(mockUpdateResult));

    await component.ngOnInit();

    expect(mockWails.checkPluginUpdate).toHaveBeenCalledWith(
      mockPlugin.repository,
      mockPlugin.commitHash,
      null
    );
    expect(component.checking()).toBe(false);
    expect(component.updateResult()?.hasUpdate).toBe(true);
  });

  it('should handle update check error', async () => {
    mockWails.checkPluginUpdate.and.returnValue(Promise.reject('Network error'));

    await component.ngOnInit();

    expect(component.checking()).toBe(false);
    expect(component.updateResult()?.error).toContain('Network error');
  });

  it('should get short commit hash', () => {
    const result = component.getShortCommit('abc1234567890');
    expect(result).toBe('abc1234');
  });

  it('should open changelog in new window', () => {
    const mockWindow = jasmine.createSpyObj('window', ['open']);
    spyOn(window, 'open');

    component.updateResult.set({
      hasUpdate: true,
      currentCommit: 'abc1234',
      latestCommit: 'def5678',
      changelogUrl: 'https://github.com/test/plugin/changelog'
    });

    component.openChangelog();

    expect(window.open).toHaveBeenCalledWith('https://github.com/test/plugin/changelog', '_blank');
  });

  it('should proceed with update', async () => {
    component.updateResult.set({
      hasUpdate: true,
      currentCommit: 'abc1234',
      latestCommit: 'def5678',
      recommendedCommit: 'def5678'
    });

    mockWails.updatePluginToCommit.and.returnValue(Promise.resolve());

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

    mockWails.updatePluginToCommit.and.returnValue(Promise.reject('LOCAL_MODIFICATIONS'));

    const mockConfirmDialogRef = jasmine.createSpyObj('MatDialogRef', ['afterClosed']);
    mockConfirmDialogRef.afterClosed.and.returnValue(of(true));
    mockDialog.open.and.returnValue(mockConfirmDialogRef);
    mockWails.updatePluginWithForce.and.returnValue(Promise.resolve());

    await component.proceedWithUpdate();

    expect(mockDialog.open).toHaveBeenCalled();
    expect(mockWails.updatePluginWithForce).toHaveBeenCalledWith(
      mockPlugin.repository,
      'def5678'
    );
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: true });
  });

  it('should close dialog without updating', () => {
    component.close();
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: false });
  });
});
