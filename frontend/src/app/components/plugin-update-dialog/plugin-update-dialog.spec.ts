import { ComponentFixture, TestBed } from '@angular/core/testing';
import { MatDialogRef } from '@angular/material/dialog';
import { PluginUpdateDialog } from './plugin-update-dialog';
import { Wails } from '../../core/services/wails';
import { models } from '../../../wailsjs/go/models';

describe('PluginUpdateDialog', () => {
  let component: PluginUpdateDialog;
  let fixture: ComponentFixture<PluginUpdateDialog>;
  let mockDialogRef: jasmine.SpyObj<MatDialogRef<PluginUpdateDialog>>;
  let mockWails: jasmine.SpyObj<Wails>;

  beforeEach(async () => {
    mockDialogRef = jasmine.createSpyObj('MatDialogRef', ['close']);
    mockWails = jasmine.createSpyObj('Wails', [
      'getPluginsV2',
      'checkPluginUpdate',
      'updatePluginToCommit',
      'logToFile'
    ]);

    await TestBed.configureTestingModule({
      imports: [PluginUpdateDialog],
      providers: [
        { provide: MatDialogRef, useValue: mockDialogRef },
        { provide: Wails, useValue: mockWails }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PluginUpdateDialog);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with checking state', () => {
    expect(component.checking()).toBe(true);
    expect(component.updating()).toBe(false);
    expect(component.availableUpdates().length).toBe(0);
  });

  it('should check for updates on init', async () => {
    const mockPlugin = {
      id: 1,
      installSource: 'remote',
      repository: 'https://github.com/test/plugin',
      commitHash: 'abc123',
      definition: {
        plugin: {
          name: 'Test Plugin',
          icon: 'test'
        }
      }
    } as any;

    mockWails.getPluginsV2.and.returnValue(Promise.resolve([mockPlugin]));
    mockWails.checkPluginUpdate.and.returnValue(Promise.resolve({
      has_update: true,
      current_commit: 'abc123',
      recommended_commit: 'def456',
      latest_commit: 'ghi789',
      changelog_url: 'https://github.com/test/plugin/compare/abc123...def456'
    }));

    await component.ngOnInit();

    expect(component.checking()).toBe(false);
    expect(component.availableUpdates().length).toBe(1);
    expect(component.selection.selected.length).toBe(1);
  });

  it('should not show plugins without updates', async () => {
    const mockPlugin = {
      id: 1,
      installSource: 'remote',
      repository: 'https://github.com/test/plugin',
      commitHash: 'abc123',
      definition: {
        plugin: {
          name: 'Test Plugin',
          icon: 'test'
        }
      }
    } as any;

    mockWails.getPluginsV2.and.returnValue(Promise.resolve([mockPlugin]));
    mockWails.checkPluginUpdate.and.returnValue(Promise.resolve({
      has_update: false,
      current_commit: 'abc123'
    }));

    await component.ngOnInit();

    expect(component.checking()).toBe(false);
    expect(component.availableUpdates().length).toBe(0);
  });

  it('should filter out builtin plugins', async () => {
    const mockPlugins = [
      {
        id: 1,
        installSource: 'builtin',
        definition: { plugin: { name: 'Builtin Plugin' } }
      },
      {
        id: 2,
        installSource: 'remote',
        repository: 'https://github.com/test/plugin',
        commitHash: 'abc123',
        definition: { plugin: { name: 'Remote Plugin' } }
      }
    ] as any;

    mockWails.getPluginsV2.and.returnValue(Promise.resolve(mockPlugins));
    mockWails.checkPluginUpdate.and.returnValue(Promise.resolve({
      has_update: true,
      current_commit: 'abc123',
      recommended_commit: 'def456',
      latest_commit: 'ghi789'
    }));

    await component.ngOnInit();

    expect(mockWails.checkPluginUpdate).toHaveBeenCalledTimes(1);
  });

  it('should handle master toggle', async () => {
    const mockUpdates = [
      { plugin: { id: 1 } as any, currentCommit: 'a', recommendedCommit: 'b', latestCommit: 'c' },
      { plugin: { id: 2 } as any, currentCommit: 'd', recommendedCommit: 'e', latestCommit: 'f' }
    ];

    component.availableUpdates.set(mockUpdates);
    mockUpdates.forEach(u => component.selection.select(u));

    expect(component.isAllSelected()).toBe(true);

    component.masterToggle();
    expect(component.selection.selected.length).toBe(0);

    component.masterToggle();
    expect(component.selection.selected.length).toBe(2);
  });

  it('should get short commit hash', () => {
    const fullCommit = 'abc123def456ghi789';
    expect(component.getShortCommit(fullCommit)).toBe('abc123d');
  });

  it('should update selected plugins', async () => {
    const mockUpdate = {
      plugin: {
        id: 1,
        repository: 'https://github.com/test/plugin',
        definition: { plugin: { name: 'Test Plugin' } }
      } as any,
      currentCommit: 'abc123',
      recommendedCommit: 'def456',
      latestCommit: 'ghi789'
    };

    component.availableUpdates.set([mockUpdate]);
    component.selection.select(mockUpdate);

    mockWails.updatePluginToCommit.and.returnValue(Promise.resolve());
    mockWails.logToFile.and.returnValue(Promise.resolve());

    await component.updateSelected();

    expect(mockWails.updatePluginToCommit).toHaveBeenCalledWith(
      'https://github.com/test/plugin',
      'def456'
    );
    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: true, count: 1 });
  });

  it('should not update if no plugins selected', async () => {
    await component.updateSelected();

    expect(mockWails.updatePluginToCommit).not.toHaveBeenCalled();
    expect(mockDialogRef.close).not.toHaveBeenCalled();
  });

  it('should close dialog without updating', () => {
    component.close();

    expect(mockDialogRef.close).toHaveBeenCalledWith({ updated: false });
  });

  it('should handle update errors gracefully', async () => {
    const mockUpdate = {
      plugin: {
        id: 1,
        repository: 'https://github.com/test/plugin',
        definition: { plugin: { name: 'Test Plugin' } }
      } as any,
      currentCommit: 'abc123',
      recommendedCommit: 'def456',
      latestCommit: 'ghi789'
    };

    component.availableUpdates.set([mockUpdate]);
    component.selection.select(mockUpdate);

    mockWails.updatePluginToCommit.and.returnValue(Promise.reject('Network error'));
    mockWails.logToFile.and.returnValue(Promise.resolve());

    await component.updateSelected();

    expect(mockWails.logToFile).toHaveBeenCalledWith(
      jasmine.stringContaining('Failed to update')
    );
    expect(component.updating()).toBe(false);
  });

  it('should set updating state during update', async () => {
    const mockUpdate = {
      plugin: {
        id: 1,
        repository: 'https://github.com/test/plugin',
        definition: { plugin: { name: 'Test Plugin' } }
      } as any,
      currentCommit: 'abc123',
      recommendedCommit: 'def456',
      latestCommit: 'ghi789'
    };

    component.availableUpdates.set([mockUpdate]);
    component.selection.select(mockUpdate);

    mockWails.updatePluginToCommit.and.returnValue(
      new Promise(resolve => setTimeout(resolve, 100))
    );
    mockWails.logToFile.and.returnValue(Promise.resolve());

    const updatePromise = component.updateSelected();
    expect(component.updating()).toBe(true);

    await updatePromise;
    expect(component.updating()).toBe(false);
  });

  it('should open changelog in new window', () => {
    spyOn(window, 'open');
    const url = 'https://github.com/test/plugin/compare/abc...def';

    component.openChangelog(url);

    expect(window.open).toHaveBeenCalledWith(url, '_blank');
  });

  it('should not open changelog if url is empty', () => {
    spyOn(window, 'open');

    component.openChangelog('');

    expect(window.open).not.toHaveBeenCalled();
  });
});
