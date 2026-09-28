import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RecipeRegistryDetail } from './recipe-registry-detail';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

describe('RecipeRegistryDetail', () => {
  let component: RecipeRegistryDetail;
  let fixture: ComponentFixture<RecipeRegistryDetail>;
  let activatedRouteMock: any;
  let routerMock: any;
  let wailsMock: any;
  let notificationMock: any;
  let dialogMock: any;

  const registryRecipe = {
    id: 'recipe-1',
    label: 'Test Recipe',
    description: 'A test recipe',
    author: { name: 'Jane' },
    category: { name: 'preprocessing' },
    tags: [],
    status: 'approved',
    latest_version: {
      revision: 2,
      data: {
        version: 1,
        label: 'Test Recipe',
        stages: [
          { pluginId: 'wide-to-long', pluginVersion: '1.0.0', bindings: {} },
          { pluginId: 'long-to-wide', pluginVersion: '1.0.0', bindings: { input_file: { stage: 0, output: 'long_data' } } }
        ]
      }
    }
  };

  beforeEach(async () => {
    activatedRouteMock = {
      snapshot: { paramMap: { get: vi.fn().mockReturnValue('recipe-1') } }
    };
    routerMock = { navigate: vi.fn() };
    wailsMock = {
      getRegistryRecipe: vi.fn().mockResolvedValue(registryRecipe),
      downloadRecipeFromRegistry: vi.fn().mockResolvedValue({ recipe: { id: 'local-1' }, compatibility: { allOk: true, stages: [] } }),
      logToFile: vi.fn().mockResolvedValue(undefined),
      generateRecipeDiagramFromData: vi.fn().mockResolvedValue('flowchart TD\n')
    };

    vi.doMock('mermaid', () => ({
      default: {
        initialize: vi.fn(),
        render: vi.fn().mockResolvedValue({ svg: '<svg></svg>' })
      }
    }));
    vi.doMock('svg-pan-zoom', () => ({
      default: vi.fn().mockReturnValue({ destroy: vi.fn() })
    }));
    notificationMock = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showInfo: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(true) })
    };

    await TestBed.configureTestingModule({
      imports: [RecipeRegistryDetail, NoopAnimationsModule],
      providers: [
        { provide: ActivatedRoute, useValue: activatedRouteMock },
        { provide: Router, useValue: routerMock },
        { provide: Wails, useValue: wailsMock },
        { provide: NotificationService, useValue: notificationMock }
      ]
    })
      .overrideProvider(MatDialog, { useValue: dialogMock })
      .compileComponents();

    fixture = TestBed.createComponent(RecipeRegistryDetail);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads the recipe by id from the route', async () => {
    await component.ngOnInit();

    expect(wailsMock.getRegistryRecipe).toHaveBeenCalledWith('recipe-1');
    expect(component['recipe']()?.label).toBe('Test Recipe');
  });

  it('redirects to the registry list when no id is present', async () => {
    activatedRouteMock.snapshot.paramMap.get = vi.fn().mockReturnValue(null);

    await component.ngOnInit();

    expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe-registry']);
  });

  it('derives stage rows including which stages have bound inputs', async () => {
    await component.ngOnInit();

    const stages = component['stages']();
    expect(stages.length).toBe(2);
    expect(stages[0].bound).toBe(false);
    expect(stages[1].bound).toBe(true);
    expect(stages[1].pluginId).toBe('long-to-wide');
  });

  it('generates the diagram from the recipe\'s latest version data', async () => {
    await component.ngOnInit();

    await component['diagramGenerator']([0, 1]);

    expect(wailsMock.generateRecipeDiagramFromData).toHaveBeenCalledWith(
      JSON.stringify(registryRecipe.latest_version.data),
      [0, 1]
    );
  });

  describe('downloadRecipe', () => {
    it('downloads and navigates to the local recipe editor on confirm', async () => {
      await component.ngOnInit();
      await component.downloadRecipe();

      expect(wailsMock.downloadRecipeFromRegistry).toHaveBeenCalledWith('recipe-1');
      expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe', 'local-1']);
    });

    it('does not download when the user cancels the confirmation', async () => {
      dialogMock.open.mockReturnValue({ afterClosed: () => of(false) });
      await component.ngOnInit();

      await component.downloadRecipe();

      expect(wailsMock.downloadRecipeFromRegistry).not.toHaveBeenCalled();
    });

    it('clears downloading state even when the download fails', async () => {
      wailsMock.downloadRecipeFromRegistry = vi.fn().mockRejectedValue(new Error('network error'));
      await component.ngOnInit();

      await component.downloadRecipe();

      expect(component['downloading']()).toBe(false);
      expect(notificationMock.showError).toHaveBeenCalled();
    });
  });
});
