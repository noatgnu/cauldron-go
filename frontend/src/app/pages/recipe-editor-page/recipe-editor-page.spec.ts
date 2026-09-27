import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ActivatedRoute } from '@angular/router';
import { of } from 'rxjs';
import { convertToParamMap } from '@angular/router';
import { RecipeEditorPage } from './recipe-editor-page';

describe('RecipeEditorPage', () => {
  let component: RecipeEditorPage;
  let fixture: ComponentFixture<RecipeEditorPage>;

  async function setup(id: string | null) {
    await TestBed.configureTestingModule({
      imports: [RecipeEditorPage],
      providers: [
        {
          provide: ActivatedRoute,
          useValue: { paramMap: of(convertToParamMap(id ? { id } : {})) }
        }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(RecipeEditorPage);
    component = fixture.componentInstance;
  }

  it('reads a recipe id from the route', async () => {
    await setup('recipe-1');
    component.ngOnInit();
    expect(component['recipeId']()).toBe('recipe-1');
  });

  it('has no recipe id for a new recipe route', async () => {
    await setup(null);
    component.ngOnInit();
    expect(component['recipeId']()).toBeNull();
  });
});
