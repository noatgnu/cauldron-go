import { ComponentFixture, TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { of } from 'rxjs';
import { PluginRegistry } from './plugin-registry';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('PluginRegistry', () => {
  let component: PluginRegistry;
  let fixture: ComponentFixture<PluginRegistry>;
  let wailsMock: any;
  let notificationMock: any;
  let routerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      listRegistryPlugins: vi.fn().mockResolvedValue({ plugins: [], total: 0 }),
      listRegistryCategories: vi.fn().mockResolvedValue([]),
      getRegistryFilterOptions: vi.fn().mockResolvedValue({
        categories: [], subcategories: [], authors: [], tags: [], languages: []
      }),
      getPluginsV2: vi.fn().mockResolvedValue([]),
      progress: signal(null),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn()
    };
    routerMock = {
      navigate: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(undefined) })
    };

    await TestBed.configureTestingModule({
      imports: [PluginRegistry, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: Router, useValue: routerMock }
      ]
    })
    // overrideProvider reliably replaces MatDialog even though the component imports MatDialogModule itself.
    .overrideProvider(MatDialog, { useValue: dialogMock })
    .compileComponents();

    fixture = TestBed.createComponent(PluginRegistry);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads filter options on init', () => {
    expect(wailsMock.getRegistryFilterOptions).toHaveBeenCalled();
  });

  describe('filter changes', () => {
    it('passes subcategory, language and tag filters to listRegistryPlugins', async () => {
      component.selectedSubcategory = 'quality-control';
      component.selectedLanguage = 'python';
      component.selectedTag = 'proteomics';

      await component.loadPlugins();

      expect(wailsMock.listRegistryPlugins).toHaveBeenCalledWith(
        '', '', '', 'quality-control', 'python', 'proteomics', (component as any).pageSize, 0
      );
    });

    it('resets to the first page when a filter changes', async () => {
      (component as any).pageIndex = 3;

      await component.onLanguageChange();

      expect((component as any).pageIndex).toBe(0);
    });

    it('clearFilters resets all filter fields and reloads', async () => {
      component.searchQuery = 'x';
      component.selectedCategory = 'a';
      component.selectedSubcategory = 'b';
      component.selectedLanguage = 'c';
      component.selectedTag = 'd';

      await component.clearFilters();

      expect(component.searchQuery).toBe('');
      expect(component.selectedCategory).toBe('');
      expect(component.selectedSubcategory).toBe('');
      expect(component.selectedLanguage).toBe('');
      expect(component.selectedTag).toBe('');
    });
  });

  describe('installPlugin', () => {
    const plugin: any = {
      id: 'plugin-1',
      name: 'Test Plugin',
      description: 'Test',
      repository: 'https://github.com/test/repo',
      author: { name: 'Test' }
    };

    it('sets checkingPluginId to the plugin being checked and clears it on success', async () => {
      let sawCheckingId: string | null = null;
      wailsMock.fetchPluginDependencies = vi.fn().mockImplementation(async () => {
        sawCheckingId = (component as any).checkingPluginId();
        return { hasPythonDeps: false, hasRDeps: true, runtimeEnvironments: ['r'] };
      });

      await component.installPlugin(plugin);

      expect(sawCheckingId).toBe('plugin-1');
      expect((component as any).checkingPluginId()).toBeNull();
      expect(dialogMock.open).toHaveBeenCalled();
    });

    it('clears checkingPluginId even when the dependency fetch fails', async () => {
      wailsMock.fetchPluginDependencies = vi.fn().mockRejectedValue(new Error('network error'));

      await component.installPlugin(plugin);

      expect((component as any).checkingPluginId()).toBeNull();
      expect(dialogMock.open).toHaveBeenCalled();
    });
  });
});
