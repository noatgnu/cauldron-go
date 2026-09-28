import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router } from '@angular/router';
import { MatDialog } from '@angular/material/dialog';
import { of } from 'rxjs';
import { vi } from 'vitest';
import { Recipes } from './recipes';
import { Wails } from '../../core/services/wails';
import { RecipeService } from '../../core/services/recipe';

describe('Recipes', () => {
  let component: Recipes;
  let fixture: ComponentFixture<Recipes>;
  let wailsMock: any;
  let recipeServiceMock: any;
  let routerMock: any;
  let dialogMock: any;

  beforeEach(async () => {
    wailsMock = {
      logToFile: vi.fn().mockResolvedValue(undefined),
      saveFileDialog: vi.fn().mockResolvedValue('/tmp/recipe.json'),
      openFileDialog: vi.fn().mockResolvedValue('/tmp/recipe.json'),
      openDirectoryDialog: vi.fn().mockResolvedValue('/tmp/nextflow-pipeline')
    };
    recipeServiceMock = {
      getAllRecipes: vi.fn().mockResolvedValue([{ id: 'recipe-1', label: 'Recipe 1', createdAt: new Date().toISOString() }]),
      checkCompatibility: vi.fn().mockResolvedValue({ recipeId: 'recipe-1', allOk: true, stages: [] }),
      runRecipe: vi.fn().mockResolvedValue({ id: 'chain-1' }),
      exportRecipe: vi.fn().mockResolvedValue(undefined),
      exportRecipeNextflow: vi.fn().mockResolvedValue(undefined),
      importRecipeFromFile: vi.fn().mockResolvedValue({ recipe: { id: 'recipe-2' }, compatibility: { allOk: true, stages: [] } }),
      deleteRecipe: vi.fn().mockResolvedValue(undefined)
    };
    routerMock = {
      navigate: vi.fn()
    };
    dialogMock = {
      open: vi.fn().mockReturnValue({ afterClosed: () => of(true) })
    };

    await TestBed.configureTestingModule({
      imports: [Recipes],
      providers: [
        { provide: Wails, useValue: wailsMock },
        { provide: RecipeService, useValue: recipeServiceMock },
        { provide: Router, useValue: routerMock },
        { provide: MatDialog, useValue: dialogMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(Recipes);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads recipes with their compatibility report', async () => {
    await component.loadRecipes();
    expect(component['rows']().length).toBe(1);
    expect(component['rows']()[0].recipe.id).toBe('recipe-1');
    expect(component['rows']()[0].compatibility?.allOk).toBe(true);
  });

  it('navigates to a new recipe', () => {
    component.newRecipe();
    expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe/new']);
  });

  it('navigates to an existing recipe editor', () => {
    component.editRecipe('recipe-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe', 'recipe-1']);
  });

  it('runs a recipe and navigates to the chain detail page', async () => {
    await component.runRecipe(new Event('click'), 'recipe-1');
    expect(recipeServiceMock.runRecipe).toHaveBeenCalledWith('recipe-1');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/job-chain', 'chain-1']);
  });

  it('exports a recipe with install info when the user confirms it', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(true) });
    await component.loadRecipes();
    await component.exportRecipe(new Event('click'), component['rows']()[0]);
    expect(wailsMock.saveFileDialog).toHaveBeenCalled();
    expect(recipeServiceMock.exportRecipe).toHaveBeenCalledWith('recipe-1', '/tmp/recipe.json', true);
  });

  it('exports a recipe without install info when the user declines it', async () => {
    dialogMock.open.mockReturnValue({ afterClosed: () => of(false) });
    await component.loadRecipes();
    await component.exportRecipe(new Event('click'), component['rows']()[0]);
    expect(recipeServiceMock.exportRecipe).toHaveBeenCalledWith('recipe-1', '/tmp/recipe.json', false);
  });

  it('exports a recipe as a Nextflow pipeline to the chosen directory', async () => {
    await component.loadRecipes();
    await component.exportRecipeNextflow(new Event('click'), component['rows']()[0]);
    expect(wailsMock.openDirectoryDialog).toHaveBeenCalled();
    expect(recipeServiceMock.exportRecipeNextflow).toHaveBeenCalledWith('recipe-1', '/tmp/nextflow-pipeline');
  });

  it('does not export a Nextflow pipeline when the user cancels the directory dialog', async () => {
    wailsMock.openDirectoryDialog.mockResolvedValue('');
    await component.loadRecipes();
    await component.exportRecipeNextflow(new Event('click'), component['rows']()[0]);
    expect(recipeServiceMock.exportRecipeNextflow).not.toHaveBeenCalled();
  });

  it('imports a recipe and navigates to its editor', async () => {
    await component.importRecipe();
    expect(recipeServiceMock.importRecipeFromFile).toHaveBeenCalledWith('/tmp/recipe.json');
    expect(routerMock.navigate).toHaveBeenCalledWith(['/recipe', 'recipe-2']);
  });

  it('deletes a recipe after confirmation and removes it from the list', async () => {
    await component.loadRecipes();
    await component.deleteRecipe(new Event('click'), 'recipe-1');
    expect(recipeServiceMock.deleteRecipe).toHaveBeenCalledWith('recipe-1');
    expect(component['rows']().length).toBe(0);
  });
});
