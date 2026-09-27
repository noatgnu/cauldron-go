import { Component, Inject, ViewChild, signal, computed, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogRef, MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatButtonToggleModule } from '@angular/material/button-toggle';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { DynamicFormComponent } from '../dynamic-form/dynamic-form';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

export interface RecipeStageOutputRef {
  name: string;
}

export interface RecipeEarlierStage {
  index: number;
  label: string;
  outputs: RecipeStageOutputRef[];
}

export interface RecipeStageBinding {
  stageIndex: number;
  outputName: string;
}

export interface RecipeStageRowDialogData {
  plugin: models.PluginV2;
  initialValues: Record<string, any>;
  initialBindings: Record<string, RecipeStageBinding>;
  earlierStages: RecipeEarlierStage[];
}

export interface RecipeStageRowDialogResult {
  values: Record<string, any>;
  bindings: Record<string, RecipeStageBinding>;
}

@Component({
  selector: 'app-recipe-stage-row-dialog',
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatButtonToggleModule, MatFormFieldModule, MatSelectModule, DynamicFormComponent],
  templateUrl: './recipe-stage-row-dialog.html',
  styleUrl: './recipe-stage-row-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class RecipeStageRowDialog {
  @ViewChild(DynamicFormComponent) dynamicForm?: DynamicFormComponent;

  bindings = signal<Record<string, RecipeStageBinding>>({});

  fileInputs = computed(() => (this.data.plugin?.definition.inputs || []).filter(i => i.type === 'file'));

  formPlugin = computed<models.PluginV2>(() => {
    const bound = new Set(Object.keys(this.bindings()));
    return new models.PluginV2({
      ...this.data.plugin,
      definition: new models.PluginDefinition({
        ...this.data.plugin.definition,
        inputs: this.data.plugin.definition.inputs.filter(i => !bound.has(i.name))
      })
    });
  });

  constructor(
    private dialogRef: MatDialogRef<RecipeStageRowDialog, RecipeStageRowDialogResult>,
    @Inject(MAT_DIALOG_DATA) public data: RecipeStageRowDialogData
  ) {
    this.bindings.set({ ...data.initialBindings });
  }

  ngAfterViewInit() {
    setTimeout(async () => {
      await this.dynamicForm?.loadFromJobParameters(this.data.initialValues);
    }, 500);
  }

  isBound(inputName: string): boolean {
    return inputName in this.bindings();
  }

  bindingFor(inputName: string): RecipeStageBinding | undefined {
    return this.bindings()[inputName];
  }

  setMode(inputName: string, mode: 'file' | 'bound') {
    if (mode === 'file') {
      this.bindings.update(b => {
        const next = { ...b };
        delete next[inputName];
        return next;
      });
      return;
    }

    const firstStage = this.data.earlierStages[0];
    const firstOutput = firstStage?.outputs[0];
    if (!firstStage || !firstOutput) return;
    this.bindings.update(b => ({
      ...b,
      [inputName]: { stageIndex: firstStage.index, outputName: firstOutput.name }
    }));
  }

  outputsForStage(stageIndex: number): RecipeStageOutputRef[] {
    return this.data.earlierStages.find(s => s.index === stageIndex)?.outputs || [];
  }

  setBindingStage(inputName: string, stageIndex: number) {
    const outputs = this.outputsForStage(stageIndex);
    this.bindings.update(b => ({
      ...b,
      [inputName]: { stageIndex, outputName: outputs[0]?.name || '' }
    }));
  }

  setBindingOutput(inputName: string, outputName: string) {
    this.bindings.update(b => {
      const current = b[inputName];
      if (!current) return b;
      return { ...b, [inputName]: { ...current, outputName } };
    });
  }

  save() {
    if (this.dynamicForm) {
      this.dynamicForm.submit();
    } else {
      this.dialogRef.close({ values: {}, bindings: this.bindings() });
    }
  }

  onFormSubmit(values: Record<string, any>) {
    this.dialogRef.close({ values, bindings: this.bindings() });
  }

  cancel() {
    this.dialogRef.close();
  }
}
