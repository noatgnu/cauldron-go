import { Component, OnInit, OnDestroy, signal, ChangeDetectionStrategy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { RecipeEditor } from '../../components/recipe-editor/recipe-editor';

@Component({
  selector: 'app-recipe-editor-page',
  imports: [MatCardModule, RecipeEditor],
  templateUrl: './recipe-editor-page.html',
  styleUrl: './recipe-editor-page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeEditorPage implements OnInit, OnDestroy {
  protected recipeId = signal<string | null>(null);
  private paramMapSubscription?: Subscription;

  constructor(private route: ActivatedRoute) {}

  ngOnInit(): void {
    this.paramMapSubscription = this.route.paramMap.subscribe(params => {
      this.recipeId.set(params.get('id'));
    });
  }

  ngOnDestroy(): void {
    this.paramMapSubscription?.unsubscribe();
  }
}
