import { Component, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSelectModule } from '@angular/material/select';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Router } from '@angular/router';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { ConfirmDialogComponent } from '../../components/confirm-dialog/confirm-dialog';
import { firstValueFrom } from 'rxjs';
import {
  RegistryRecipe,
  RegistryRecipeListResponse,
  RegistryRecipeFilterOptions
} from '../../core/models/recipe-registry';

@Component({
  selector: 'app-recipe-registry',
  imports: [
    FormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatSelectModule,
    MatPaginatorModule,
    MatDialogModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './recipe-registry.html',
  styleUrl: './recipe-registry.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeRegistry implements OnInit {
  protected recipes = signal<RegistryRecipe[]>([]);
  protected filterOptions = signal<RegistryRecipeFilterOptions>({ categories: [], authors: [], tags: [] });
  protected loading = signal(false);
  protected totalCount = signal(0);
  protected downloadingId = signal<string | null>(null);
  protected pageSize = 10;
  protected pageIndex = 0;

  searchQuery = '';
  selectedCategory = '';
  selectedTag = '';

  displayedColumns: string[] = ['label', 'revision', 'category', 'author', 'updated', 'actions'];

  constructor(
    private wails: Wails,
    private dialog: MatDialog,
    private notification: NotificationService,
    private router: Router
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadFilterOptions();
    await this.loadRecipes();
  }

  async loadFilterOptions(): Promise<void> {
    try {
      const options = await this.wails.getRecipeRegistryFilterOptions() as RegistryRecipeFilterOptions;
      this.filterOptions.set(options);
    } catch (error) {
      await this.wails.logToFile(`[RecipeRegistry] Failed to load filter options: ${error}`);
    }
  }

  async loadRecipes(): Promise<void> {
    this.loading.set(true);
    try {
      const offset = this.pageIndex * this.pageSize;
      const response = await this.wails.listRegistryRecipes(
        this.searchQuery,
        this.selectedCategory,
        '',
        this.selectedTag,
        this.pageSize,
        offset
      ) as RegistryRecipeListResponse;

      this.recipes.set(response.results || []);
      this.totalCount.set(response.count || 0);
    } catch (error) {
      await this.wails.logToFile(`[RecipeRegistry] Failed to load recipes: ${error}`);
      this.notification.showError('Failed to load recipes from registry. Please check your recipe registry URL in settings.');
      this.recipes.set([]);
      this.totalCount.set(0);
    } finally {
      this.loading.set(false);
    }
  }

  async onSearch(): Promise<void> {
    this.pageIndex = 0;
    await this.loadRecipes();
  }

  async onCategoryChange(): Promise<void> {
    this.pageIndex = 0;
    await this.loadRecipes();
  }

  async onTagChange(): Promise<void> {
    this.pageIndex = 0;
    await this.loadRecipes();
  }

  async clearFilters(): Promise<void> {
    this.searchQuery = '';
    this.selectedCategory = '';
    this.selectedTag = '';
    this.pageIndex = 0;
    await this.loadRecipes();
  }

  async onPageChange(event: PageEvent): Promise<void> {
    this.pageIndex = event.pageIndex;
    this.pageSize = event.pageSize;
    await this.loadRecipes();
  }

  async refreshRegistry(): Promise<void> {
    this.notification.showInfo('Refreshing registry data...');
    await this.loadFilterOptions();
    await this.loadRecipes();
    this.notification.showSuccess('Registry data refreshed successfully');
  }

  viewDetails(recipe: RegistryRecipe): void {
    this.router.navigate(['/recipe-registry', recipe.id]);
  }

  async downloadRecipe(recipe: RegistryRecipe): Promise<void> {
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

    this.downloadingId.set(recipe.id);
    try {
      const result = await this.wails.downloadRecipeFromRegistry(recipe.id);
      if (!result.recipe) throw new Error('Download did not return a recipe');
      if (!result.compatibility?.allOk) {
        this.notification.showInfo('Recipe downloaded, but some plugins are missing or incompatible. Check the compatibility badge before running it.');
      }
      await this.router.navigate(['/recipe', result.recipe.id]);
    } catch (error: any) {
      await this.wails.logToFile(`[RecipeRegistry] Failed to download recipe: ${error?.message || String(error)}`);
      this.notification.showError('Failed to download recipe');
    } finally {
      this.downloadingId.set(null);
    }
  }

  formatDate(dateString: string): string {
    if (!dateString) return 'N/A';
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) return 'N/A';
      return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
    } catch {
      return 'N/A';
    }
  }

  getAuthorName(recipe: RegistryRecipe): string {
    return recipe.author?.name || 'Unknown';
  }

  getCategoryName(recipe: RegistryRecipe): string {
    return recipe.category?.name || 'Uncategorized';
  }
}
