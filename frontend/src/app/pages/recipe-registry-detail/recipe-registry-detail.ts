import { Component, OnInit, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { firstValueFrom } from 'rxjs';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { ConfirmDialogComponent } from '../../components/confirm-dialog/confirm-dialog';
import { RecipeDiagram } from '../../components/recipe-diagram/recipe-diagram';
import { RegistryRecipe } from '../../core/models/recipe-registry';

interface StageRow {
  index: number;
  pluginId: string;
  pluginVersion: string;
  bound: boolean;
}

@Component({
  selector: 'app-recipe-registry-detail',
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatDialogModule,
    MatTooltipModule,
    RecipeDiagram
  ],
  templateUrl: './recipe-registry-detail.html',
  styleUrl: './recipe-registry-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeRegistryDetail implements OnInit {
  protected recipe = signal<RegistryRecipe | null>(null);
  protected loading = signal(false);
  protected downloading = signal(false);

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private dialog: MatDialog,
    private notification: NotificationService
  ) {}

  async ngOnInit(): Promise<void> {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      await this.router.navigate(['/recipe-registry']);
      return;
    }
    await this.loadRecipeDetail(id);
  }

  async loadRecipeDetail(id: string): Promise<void> {
    this.loading.set(true);
    try {
      const recipe = await this.wails.getRegistryRecipe(id) as RegistryRecipe;
      this.recipe.set(recipe);
    } catch (error: any) {
      await this.wails.logToFile(`[RecipeRegistryDetail] Failed to load recipe: ${error?.message || String(error)}`);
      this.notification.showError('Failed to load recipe from registry');
    } finally {
      this.loading.set(false);
    }
  }

  protected diagramGenerator = (expandedStages: number[]) => {
    const data = this.recipe()?.latest_version?.data;
    return this.wails.generateRecipeDiagramFromData(JSON.stringify(data ?? {}), expandedStages);
  };

  protected stages = computed<StageRow[]>(() => {
    const data = this.recipe()?.latest_version?.data;
    const stages = data?.['stages'];
    if (!Array.isArray(stages)) return [];
    return stages.map((s: any, index: number) => ({
      index,
      pluginId: s?.pluginId ?? '-',
      pluginVersion: s?.pluginVersion ?? '-',
      bound: !!(s?.bindings && Object.keys(s.bindings).length > 0)
    }));
  });

  async downloadRecipe(): Promise<void> {
    const recipe = this.recipe();
    if (!recipe) return;

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '440px',
      disableClose: true,
      data: {
        title: 'Download this recipe?',
        message: `This downloads "${recipe.label}" (revision ${recipe.latest_version?.revision ?? '?'}) as a new local recipe. If any of its plugins aren't installed, you'll be able to see which ones after downloading.`,
        confirmText: 'Download',
        cancelText: 'Cancel'
      }
    });

    const confirmed = await firstValueFrom(dialogRef.afterClosed());
    if (!confirmed) return;

    this.downloading.set(true);
    try {
      const result = await this.wails.downloadRecipeFromRegistry(recipe.id);
      if (!result.recipe) throw new Error('Download did not return a recipe');
      if (!result.compatibility?.allOk) {
        this.notification.showInfo('Recipe downloaded, but some plugins are missing or incompatible. Check the compatibility badge before running it.');
      }
      await this.router.navigate(['/recipe', result.recipe.id]);
    } catch (error: any) {
      await this.wails.logToFile(`[RecipeRegistryDetail] Failed to download recipe: ${error?.message || String(error)}`);
      this.notification.showError('Failed to download recipe');
    } finally {
      this.downloading.set(false);
    }
  }

  goBack(): void {
    this.router.navigate(['/recipe-registry']);
  }

  getAuthorName(recipe: RegistryRecipe): string {
    return recipe.author?.name || 'Unknown';
  }

  getCategoryName(recipe: RegistryRecipe): string {
    return recipe.category?.name || 'Uncategorized';
  }
}
