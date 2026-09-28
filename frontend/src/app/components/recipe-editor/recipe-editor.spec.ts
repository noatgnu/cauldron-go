import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi } from 'vitest';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RecipeEditor } from './recipe-editor';
import { RecipeService } from '../../core/services/recipe';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { NotificationService } from '../../core/services/notification.service';

vi.doMock('svg-pan-zoom', () => ({
  default: vi.fn().mockReturnValue({ destroy: vi.fn() })
}));

describe('RecipeEditor', () => {
  let component: RecipeEditor;
  let fixture: ComponentFixture<RecipeEditor>;
  let mockRecipeService: any;
  let mockPluginService: any;
  let mockNotification: any;
  let mockRouter: any;
  let mockDialog: any;

  const pluginA = {
    id: 1,
    definition: {
      plugin: { id: 'plugin-a', name: 'Plugin A', version: '1.0.0' },
      inputs: [],
      outputs: [{ name: 'out_a', path: 'out_a.tsv', type: 'data' }],
      runtime: { environments: ['python'], entrypoint: 'a.py' },
      execution: {}
    },
    folderPath: '/a',
    scriptPath: '/a/a.py',
    installSource: 'builtin',
    commitHash: '',
    repository: '',
    enabled: true
  };

  const pluginB = {
    id: 2,
    definition: {
      plugin: { id: 'plugin-b', name: 'Plugin B', version: '1.0.0' },
      inputs: [{ name: 'input_file', label: 'Input', type: 'file', required: true }],
      outputs: [],
      runtime: { environments: ['python'], entrypoint: 'b.py' },
      execution: {}
    },
    folderPath: '/b',
    scriptPath: '/b/b.py',
    installSource: 'builtin',
    commitHash: '',
    repository: '',
    enabled: true
  };

  beforeEach(async () => {
    mockRecipeService = {
      getRecipe: vi.fn(),
      getRecipeStages: vi.fn(),
      checkCompatibility: vi.fn(),
      saveRecipe: vi.fn().mockResolvedValue({ id: 'recipe-1', label: 'My Recipe' }),
      updateRecipe: vi.fn().mockResolvedValue({ id: 'recipe-1', label: 'My Recipe' }),
      runRecipe: vi.fn().mockResolvedValue({ id: 'chain-1' }),
      generateDiagram: vi.fn().mockResolvedValue('flowchart TD\n')
    };
    mockPluginService = {
      getAllPlugins: vi.fn().mockResolvedValue([pluginA, pluginB])
    };
    mockNotification = {
      showError: vi.fn(),
      showSuccess: vi.fn(),
      showWarning: vi.fn()
    };
    mockRouter = {
      navigate: vi.fn().mockResolvedValue(true)
    };
    mockDialog = {
      open: vi.fn()
    };

    await TestBed.configureTestingModule({
      imports: [RecipeEditor, NoopAnimationsModule],
      providers: [
        { provide: RecipeService, useValue: mockRecipeService },
        { provide: PluginV2Service, useValue: mockPluginService },
        { provide: NotificationService, useValue: mockNotification },
        { provide: Router, useValue: mockRouter },
        { provide: MatDialog, useValue: mockDialog }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(RecipeEditor);
    component = fixture.componentInstance;
  });

  async function init() {
    fixture.detectChanges();
    await fixture.whenStable();
  }

  it('should create with no stages when starting a new recipe', async () => {
    await init();
    expect(component).toBeTruthy();
    expect(component.stages().length).toBe(0);
    expect(component.availablePlugins().map(p => p.definition.plugin.id)).toEqual(['plugin-a', 'plugin-b']);
  });

  it('adds a stage for the selected plugin', async () => {
    await init();
    component.selectedNewPluginId.set('plugin-a');
    component.addStage();
    expect(component.stages().length).toBe(1);
    expect(component.stages()[0].plugin.definition.plugin.id).toBe('plugin-a');
    expect(component.selectedNewPluginId()).toBeNull();
  });

  it('removes a stage and clears later bindings that pointed to it', async () => {
    await init();
    component.stages.set([
      { id: 's1', plugin: pluginA as any, values: {}, bindings: {} },
      { id: 's2', plugin: pluginB as any, values: {}, bindings: { input_file: { stageIndex: 0, outputName: 'out_a' } } }
    ]);

    component.removeStage('s1');

    expect(component.stages().length).toBe(1);
    expect(component.stages()[0].id).toBe('s2');
    expect(component.stages()[0].bindings).toEqual({});
    expect(mockNotification.showWarning).toHaveBeenCalled();
  });

  it('shifts later binding indices down when an earlier stage is removed', async () => {
    await init();
    component.stages.set([
      { id: 's1', plugin: pluginA as any, values: {}, bindings: {} },
      { id: 's2', plugin: pluginA as any, values: {}, bindings: {} },
      { id: 's3', plugin: pluginB as any, values: {}, bindings: { input_file: { stageIndex: 1, outputName: 'out_a' } } }
    ]);

    component.removeStage('s1');

    expect(component.stages().map(s => s.id)).toEqual(['s2', 's3']);
    expect(component.stages()[1].bindings['input_file']).toEqual({ stageIndex: 0, outputName: 'out_a' });
  });

  it('loads an existing recipe with resolved stages, bindings, and compatibility', async () => {
    mockRecipeService.getRecipe.mockResolvedValue({ id: 'recipe-1', label: 'Loaded Recipe', description: 'desc' });
    mockRecipeService.getRecipeStages.mockResolvedValue([
      { id: 1, recipeId: 'recipe-1', stageIndex: 0, pluginId: 'plugin-a', params: {}, bindings: {} },
      {
        id: 2, recipeId: 'recipe-1', stageIndex: 1, pluginId: 'plugin-b',
        params: {}, bindings: { input_file: { stage: 0, output: 'out_a' } }
      }
    ]);
    mockRecipeService.checkCompatibility.mockResolvedValue({ recipeId: 'recipe-1', allOk: true, stages: [] });

    component.recipeId = 'recipe-1';
    await component.ngOnInit();

    expect(component.label()).toBe('Loaded Recipe');
    expect(component.stages().length).toBe(2);
    expect(component.stages()[1].bindings['input_file']).toEqual({ stageIndex: 0, outputName: 'out_a' });
    expect(component.compatibility()?.allOk).toBe(true);
  });

  it('refuses to save without a label', async () => {
    await init();
    component.selectedNewPluginId.set('plugin-a');
    component.addStage();
    component.label.set('');

    await component.saveRecipe();

    expect(mockRecipeService.saveRecipe).not.toHaveBeenCalled();
    expect(mockNotification.showError).toHaveBeenCalled();
  });

  it('saves the recipe and navigates to its editor page', async () => {
    await init();
    component.label.set('My Recipe');
    component.selectedNewPluginId.set('plugin-a');
    component.addStage();

    await component.saveRecipe();

    expect(mockRecipeService.saveRecipe).toHaveBeenCalledWith('My Recipe', '', [
      { pluginId: 'plugin-a', pluginVersion: '1.0.0', params: {}, bindings: {} }
    ]);
    expect(mockRouter.navigate).toHaveBeenCalledWith(['/recipe', 'recipe-1']);
  });

  it('saves then runs the recipe and navigates to the chain detail page', async () => {
    await init();
    component.label.set('My Recipe');
    component.selectedNewPluginId.set('plugin-a');
    component.addStage();

    await component.runNow();

    expect(mockRecipeService.saveRecipe).toHaveBeenCalled();
    expect(mockRecipeService.runRecipe).toHaveBeenCalledWith('recipe-1');
    expect(mockRouter.navigate).toHaveBeenCalledWith(['/job-chain', 'chain-1']);
  });
});
