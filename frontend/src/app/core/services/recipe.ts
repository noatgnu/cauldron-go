import { Injectable, inject } from '@angular/core';
import { Wails, Recipe, RecipeStage, RecipeStageRequest, RecipeImportResult, CompatibilityReport, JobChain } from './wails';

@Injectable({
  providedIn: 'root'
})
export class RecipeService {
  private wails = inject(Wails);

  async saveRecipe(label: string, description: string, stages: RecipeStageRequest[]): Promise<Recipe> {
    return this.wails.saveRecipe({ label, description, stages });
  }

  async updateRecipe(id: string, label: string, description: string, stages: RecipeStageRequest[]): Promise<Recipe> {
    return this.wails.updateRecipe(id, { label, description, stages });
  }

  async getRecipe(id: string): Promise<Recipe> {
    return this.wails.getRecipe(id);
  }

  async getRecipeStages(id: string): Promise<RecipeStage[]> {
    return this.wails.getRecipeStages(id);
  }

  async getAllRecipes(limit = 100, offset = 0): Promise<Recipe[]> {
    return this.wails.getAllRecipes(limit, offset);
  }

  async deleteRecipe(id: string): Promise<void> {
    return this.wails.deleteRecipe(id);
  }

  async exportRecipe(id: string, path: string, includeInstallInfo: boolean): Promise<void> {
    return this.wails.exportRecipe(id, path, includeInstallInfo);
  }

  async exportRecipeNextflow(id: string, outputDir: string): Promise<void> {
    return this.wails.exportRecipeNextflow(id, outputDir);
  }

  async importRecipeFromFile(path: string): Promise<RecipeImportResult> {
    return this.wails.importRecipeFromFile(path);
  }

  async checkCompatibility(id: string): Promise<CompatibilityReport> {
    return this.wails.checkRecipeCompatibility(id);
  }

  async runRecipe(id: string): Promise<JobChain> {
    return this.wails.runRecipe(id);
  }

  async generateDiagram(id: string, expandedStages: number[]): Promise<string> {
    return this.wails.generateRecipeDiagram(id, expandedStages);
  }
}
