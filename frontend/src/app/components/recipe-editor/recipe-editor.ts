import { Component, Input, OnInit, signal, inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import {
  RecipeStageRowDialog,
  RecipeStageRowDialogData,
  RecipeStageRowDialogResult,
  RecipeEarlierStage
} from '../recipe-stage-row-dialog/recipe-stage-row-dialog';
import { CompatibilityBadge } from '../compatibility-badge/compatibility-badge';
import { RecipeService } from '../../core/services/recipe';
import { PluginV2Service } from '../../core/services/plugin-v2';
import { NotificationService } from '../../core/services/notification.service';
import { RecipeStageRequest, CompatibilityReport } from '../../core/services/wails';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

interface RecipeStageBindingValue {
  stageIndex: number;
  outputName: string;
}

interface StageRow {
  id: string;
  plugin: models.PluginV2;
  values: Record<string, any>;
  bindings: Record<string, RecipeStageBindingValue>;
}

@Component({
  selector: 'app-recipe-editor',
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTableModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatTooltipModule,
    CompatibilityBadge
  ],
  templateUrl: './recipe-editor.html',
  styleUrl: './recipe-editor.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeEditor implements OnInit {
  @Input() recipeId: string | null = null;

  private dialog = inject(MatDialog);
  private recipeService = inject(RecipeService);
  private pluginService = inject(PluginV2Service);
  private notification = inject(NotificationService);
  private router = inject(Router);

  label = signal('');
  description = signal('');
  stages = signal<StageRow[]>([]);
  availablePlugins = signal<models.PluginV2[]>([]);
  selectedNewPluginId = signal<string | null>(null);
  saving = signal(false);
  running = signal(false);
  compatibility = signal<CompatibilityReport | null>(null);
  loading = signal(true);

  columns = ['summary', 'actions'];

  async ngOnInit() {
    try {
      this.availablePlugins.set(await this.pluginService.getAllPlugins());

      if (this.recipeId) {
        const recipe = await this.recipeService.getRecipe(this.recipeId);
        this.label.set(recipe.label);
        this.description.set(recipe.description || '');

        const stageRows = await this.recipeService.getRecipeStages(this.recipeId);
        const resolved: StageRow[] = [];
        for (const s of stageRows) {
          const plugin = this.availablePlugins().find(p => p.definition.plugin.id === s.pluginId);
          if (!plugin) {
            this.notification.showWarning(`Plugin "${s.pluginId}" from this recipe is no longer installed; that stage was skipped.`);
            continue;
          }
          const bindings: Record<string, RecipeStageBindingValue> = {};
          for (const [inputName, raw] of Object.entries(s.bindings || {})) {
            const b = raw as any;
            if (b && typeof b.stage === 'number' && typeof b.output === 'string') {
              bindings[inputName] = { stageIndex: b.stage, outputName: b.output };
            }
          }
          resolved.push({ id: crypto.randomUUID(), plugin, values: s.params || {}, bindings });
        }
        this.stages.set(resolved);

        this.compatibility.set(await this.recipeService.checkCompatibility(this.recipeId));
      }
    } catch (err) {
      this.notification.showError(`Failed to load recipe: ${err}`);
    } finally {
      this.loading.set(false);
    }
  }

  stageLabel(row: StageRow): string {
    const index = this.stages().findIndex(s => s.id === row.id);
    return `Stage ${index + 1}: ${row.plugin.definition.plugin.name}`;
  }

  bindingSummaryLines(row: StageRow): string[] {
    return Object.entries(row.bindings).map(([inputName, b]) => {
      const sourceStage = this.stages()[b.stageIndex];
      const sourceLabel = sourceStage ? this.stageLabel(sourceStage) : `Stage ${b.stageIndex + 1}`;
      return `← ${inputName}: ${sourceLabel} (${b.outputName})`;
    });
  }

  addStage() {
    const pluginId = this.selectedNewPluginId();
    if (!pluginId) return;
    const plugin = this.availablePlugins().find(p => p.definition.plugin.id === pluginId);
    if (!plugin) return;
    this.stages.update(rows => [...rows, { id: crypto.randomUUID(), plugin, values: {}, bindings: {} }]);
    this.selectedNewPluginId.set(null);
  }

  removeStage(id: string) {
    const removedIndex = this.stages().findIndex(s => s.id === id);
    if (removedIndex === -1) return;

    let anyBindingDropped = false;
    this.stages.update(rows => {
      const next = rows.filter(r => r.id !== id);
      return next.map(row => {
        const newBindings: Record<string, RecipeStageBindingValue> = {};
        for (const [inputName, b] of Object.entries(row.bindings)) {
          if (b.stageIndex === removedIndex) {
            anyBindingDropped = true;
            continue;
          }
          newBindings[inputName] = b.stageIndex > removedIndex ? { ...b, stageIndex: b.stageIndex - 1 } : b;
        }
        return { ...row, bindings: newBindings };
      });
    });

    if (anyBindingDropped) {
      this.notification.showWarning('Removed a stage that a later stage referenced; that binding was cleared.');
    }
  }

  async editStage(id: string) {
    const index = this.stages().findIndex(s => s.id === id);
    const row = this.stages()[index];
    if (!row) return;

    const earlierStages: RecipeEarlierStage[] = this.stages().slice(0, index).map((s, i) => ({
      index: i,
      label: this.stageLabel(s),
      outputs: (s.plugin.definition.outputs || []).map(o => ({ name: o.name }))
    }));

    const dialogRef = this.dialog.open<RecipeStageRowDialog, RecipeStageRowDialogData, RecipeStageRowDialogResult>(RecipeStageRowDialog, {
      width: '900px',
      maxHeight: '90vh',
      data: { plugin: row.plugin, initialValues: row.values, initialBindings: row.bindings, earlierStages }
    });

    const result = await firstValueFrom(dialogRef.afterClosed());
    if (result) {
      this.stages.update(rows => rows.map(r => (r.id === id ? { ...r, values: result.values, bindings: result.bindings } : r)));
    }
  }

  private buildStageRequests(): RecipeStageRequest[] {
    return this.stages().map(row => {
      const bindings: Record<string, models.RecipeStageBinding> = {};
      for (const [inputName, b] of Object.entries(row.bindings)) {
        bindings[inputName] = new models.RecipeStageBinding({ stage: b.stageIndex, output: b.outputName });
      }
      return {
        pluginId: row.plugin.definition.plugin.id,
        pluginVersion: row.plugin.definition.plugin.version,
        params: row.values,
        bindings
      } as RecipeStageRequest;
    });
  }

  private async persistRecipe(): Promise<string> {
    const stages = this.buildStageRequests();
    const recipe = this.recipeId
      ? await this.recipeService.updateRecipe(this.recipeId, this.label(), this.description(), stages)
      : await this.recipeService.saveRecipe(this.label(), this.description(), stages);
    this.recipeId = recipe.id;
    return recipe.id;
  }

  async saveRecipe() {
    if (!this.label().trim()) {
      this.notification.showError('Give the recipe a label first.');
      return;
    }
    if (this.stages().length === 0) {
      this.notification.showError('Add at least one stage to the recipe.');
      return;
    }
    this.saving.set(true);
    try {
      const id = await this.persistRecipe();
      this.notification.showSuccess('Recipe saved.');
      await this.router.navigate(['/recipe', id]);
    } catch (err) {
      this.notification.showError(`Failed to save recipe: ${err}`);
    } finally {
      this.saving.set(false);
    }
  }

  async runNow() {
    if (!this.label().trim()) {
      this.notification.showError('Give the recipe a label first.');
      return;
    }
    if (this.stages().length === 0) {
      this.notification.showError('Add at least one stage to run.');
      return;
    }
    this.running.set(true);
    try {
      const id = await this.persistRecipe();
      const chain = await this.recipeService.runRecipe(id);
      this.notification.showSuccess('Recipe started.');
      await this.router.navigate(['/job-chain', chain.id]);
    } catch (err) {
      this.notification.showError(`Failed to run recipe: ${err}`);
    } finally {
      this.running.set(false);
    }
  }
}
