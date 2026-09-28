import { ComponentFixture, TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { RecipeRegistry } from './recipe-registry';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';

describe('RecipeRegistry', () => {
  let component: RecipeRegistry;
  let fixture: ComponentFixture<RecipeRegistry>;
  let wailsMock: any;
  let notificationMock: any;
  let routerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      listRegistryRecipes: vi.fn().mockResolvedValue({ count: 0, results: [] }),
      getRecipeRegistryFilterOptions: vi.fn().mockResolvedValue({ categories: [], authors: [], tags: [] }),
      downloadRecipeFromRegistry: vi.fn().mockResolvedValue({ recipe: { id: 'local-1' }, compatibility: { allOk: true, stages: [] } }),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };
    routerMock = {
      navigate: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(true) })
    };

    await TestBed.configureTestingModule({
      imports: [RecipeRegistry, NoopAnimationsModule],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock },
        { provide: Router, useValue: routerMock }
      ]
    })
      .overrideProvider(MatDialog, { useValue: dialogMock })
      .compileComponents();

    fixture = TestBed.createComponent(RecipeRegistry);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads filter options on init', () => {
    expect(wailsMock.getRecipeRegistryFilterOptions).toHaveBeenCalled();
  });

  describe('filter changes', () => {
    it('passes category and tag filters to listRegistryRecipes', async () => {
      component.selectedCategory = 'preprocessing';
      component.selectedTag = 'proteomics';

      await component.loadRecipes();

      expect(wailsMock.listRegistryRecipes).toHaveBeenCalledWith(
        '', 'preprocessing', '', 'proteomics', (component as any).pageSize, 0
      );
    });

    it('resets to the first page when a filter changes', async () => {
      (component as any).pageIndex = 3;

      await component.onTagChange();

      expect((component as any).pageIndex).toBe(0);
    });

    it('clearFilters resets all filter fields and reloads', async () => {
      component.searchQuery = 'x';
      component.selectedCategory = 'a';
      component.selectedTag = 'd';

      await component.clearFilters();

      expect(component.searchQuery).toBe('');
      expect(component.selectedCategory).toBe('');
      expect(component.selectedTag).toBe('');
    });
  });

  describe('downloadRecipe', () => {
    const recipe: any = {
      id: 'recipe-1',
      label: 'Test Recipe',
      latest_version: { revision: 2 }
    };

    it('downloads and navigates to the local recipe editor on confirm', async () => {
      await component.downloadRecipe(recipe);

      expect(wailsMock.downloadRecipeFromRegistry).toHaveBeenCalledWith('recipe-1');
      expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe', 'local-1']);
    });

    it('does not download when the user cancels the confirmation', async () => {
      dialogMock.open.mockReturnValue({ afterClosed: () => of(false) });

      await component.downloadRecipe(recipe);

      expect(wailsMock.downloadRecipeFromRegistry).not.toHaveBeenCalled();
    });

    it('warns when the downloaded recipe is not fully compatible', async () => {
      wailsMock.downloadRecipeFromRegistry = vi.fn().mockResolvedValue({
        recipe: { id: 'local-1' },
        compatibility: { allOk: false, stages: [{ stageIndex: 0, pluginId: 'missing', status: 'missing' }] }
      });

      await component.downloadRecipe(recipe);

      expect(notificationMock.showInfo).toHaveBeenCalled();
      expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe', 'local-1']);
    });

    it('clears downloadingId even when the download fails', async () => {
      wailsMock.downloadRecipeFromRegistry = vi.fn().mockRejectedValue(new Error('network error'));

      await component.downloadRecipe(recipe);

      expect((component as any).downloadingId()).toBeNull();
      expect(notificationMock.showError).toHaveBeenCalled();
    });
  });
});
