import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { firstValueFrom } from 'rxjs';
import { Wails, Recipe, CompatibilityReport } from '../../core/services/wails';
import { RecipeService } from '../../core/services/recipe';
import { CompatibilityBadge } from '../../components/compatibility-badge/compatibility-badge';
import { ConfirmDialogComponent } from '../../components/confirm-dialog/confirm-dialog';

interface RecipeRow {
  recipe: Recipe;
  compatibility: CompatibilityReport | null;
}

@Component({
  selector: 'app-recipes',
  imports: [
    CommonModule,
    MatCardModule,
    MatIconModule,
    MatButtonModule,
    MatTableModule,
    MatTooltipModule,
    CompatibilityBadge
  ],
  templateUrl: './recipes.html',
  styleUrl: './recipes.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Recipes implements OnInit {
  protected rows = signal<RecipeRow[]>([]);
  protected loading = signal(false);
  protected displayedColumns: string[] = ['label', 'compatibility', 'createdAt', 'actions'];

  constructor(
    private wails: Wails,
    private recipeService: RecipeService,
    private router: Router,
    private dialog: MatDialog
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadRecipes();
  }

  async loadRecipes(): Promise<void> {
    this.loading.set(true);
    try {
      const list = await this.recipeService.getAllRecipes();
      const rows = await Promise.all(
        list.map(async recipe => {
          let compatibility: CompatibilityReport | null = null;
          try {
            compatibility = await this.recipeService.checkCompatibility(recipe.id);
          } catch {
            compatibility = null;
          }
          return { recipe, compatibility };
        })
      );
      this.rows.set(rows);
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to load recipes: ${error?.message || String(error)}`);
    } finally {
      this.loading.set(false);
    }
  }

  newRecipe(): void {
    this.router.navigate(['/recipe/new']);
  }

  editRecipe(id: string): void {
    this.router.navigate(['/recipe', id]);
  }

  async runRecipe(event: Event, id: string): Promise<void> {
    event.stopPropagation();
    try {
      const chain = await this.recipeService.runRecipe(id);
      await this.router.navigate(['/job-chain', chain.id]);
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to run recipe: ${error?.message || String(error)}`);
    }
  }

  async exportRecipe(event: Event, row: RecipeRow): Promise<void> {
    event.stopPropagation();
    try {
      const defaultName = `${row.recipe.label.replace(/[^a-z0-9-_]+/gi, '_')}.json`;
      const path = await this.wails.saveFileDialog('Export Recipe', defaultName);
      if (!path) return;

      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        width: '480px',
        disableClose: true,
        data: {
          title: 'Include plugin install info?',
          message: 'Embedding each plugin\'s repository, commit, and dependency requirements lets someone reinstall exactly these plugins if they\'re missing on their machine, but it puts those source repository URLs in the exported file.',
          confirmText: 'Include install info',
          cancelText: 'Export without it'
        }
      });
      const includeInstallInfo = await firstValueFrom(dialogRef.afterClosed());

      await this.recipeService.exportRecipe(row.recipe.id, path, !!includeInstallInfo);
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to export recipe: ${error?.message || String(error)}`);
    }
  }

  async exportRecipeNextflow(event: Event, row: RecipeRow): Promise<void> {
    event.stopPropagation();
    try {
      const outputDir = await this.wails.openDirectoryDialog('Export Nextflow Pipeline To');
      if (!outputDir) return;

      await this.recipeService.exportRecipeNextflow(row.recipe.id, outputDir);
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to export recipe as a Nextflow pipeline: ${error?.message || String(error)}`);
    }
  }

  async importRecipe(): Promise<void> {
    try {
      const path = await this.wails.openFileDialog('Import Recipe');
      if (!path) return;
      const result = await this.recipeService.importRecipeFromFile(path);
      if (!result.recipe) throw new Error('Import did not return a recipe');
      await this.router.navigate(['/recipe', result.recipe.id]);
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to import recipe: ${error?.message || String(error)}`);
    }
  }

  async deleteRecipe(event: Event, id: string): Promise<void> {
    event.stopPropagation();
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      disableClose: true,
      data: {
        title: 'Delete this recipe?',
        message: 'This deletes the saved recipe. It does not affect any chains already started from it.',
        confirmText: 'Yes, Delete',
        cancelText: 'Cancel'
      }
    });

    const confirmed = await firstValueFrom(dialogRef.afterClosed());
    if (!confirmed) return;

    try {
      await this.recipeService.deleteRecipe(id);
      this.rows.update(rows => rows.filter(r => r.recipe.id !== id));
    } catch (error: any) {
      await this.wails.logToFile(`[Recipes] Failed to delete recipe: ${error?.message || String(error)}`);
    }
  }
}
