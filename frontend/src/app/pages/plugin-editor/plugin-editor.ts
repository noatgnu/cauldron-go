import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { MatStepperModule } from '@angular/material/stepper';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatChipsModule } from '@angular/material/chips';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatTooltipModule } from '@angular/material/tooltip';

import { PluginEditorService } from '../../core/services/plugin-editor.service';
import { Wails } from '../../core/services/wails';
import { models } from '../../../wailsjs/go/models';

@Component({
  selector: 'app-plugin-editor',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatStepperModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatCardModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatChipsModule,
    MatExpansionModule,
    MatTooltipModule
  ],
  templateUrl: './plugin-editor.html',
  styleUrls: ['./plugin-editor.scss']
})
export class PluginEditor implements OnInit {
  mode = signal<'create' | 'edit'>('create');
  loading = signal(false);
  saving = signal(false);
  validating = signal(false);
  validationResult = signal<{valid: boolean, errors: string[]} | null>(null);
  yamlPreview = signal('');

  pluginForm!: FormGroup;
  categories: string[] = [];
  runtimeTypes: Array<{value: string, label: string}> = [];
  inputTypes: Array<{value: string, label: string, description: string}> = [];

  constructor(
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router,
    private pluginEditorService: PluginEditorService,
    private wails: Wails
  ) {
    this.categories = this.pluginEditorService.getAvailableCategories();
    this.runtimeTypes = this.pluginEditorService.getAvailableRuntimeTypes();
    this.inputTypes = this.pluginEditorService.getAvailableInputTypes();
    this.initializeForm();
  }

  async ngOnInit() {
    const id = this.route.snapshot.paramMap.get('id');
    const templateId = this.route.snapshot.queryParamMap.get('template');

    if (id && id !== 'new') {
      await this.loadPlugin(id);
    } else if (templateId) {
      await this.cloneFromTemplate(templateId);
    } else {
      this.mode.set('create');
    }
  }

  private initializeForm() {
    this.pluginForm = this.fb.group({
      plugin: this.fb.group({
        id: ['', [Validators.required, Validators.pattern(/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/)]],
        name: ['', Validators.required],
        description: ['', [Validators.required, Validators.minLength(10)]],
        version: ['1.0.0', [Validators.required, Validators.pattern(/^\d+\.\d+\.\d+$/)]],
        author: ['CauldronGO Team'],
        category: ['analysis', Validators.required],
        icon: ['']
      }),
      runtime: this.fb.group({
        type: ['python', Validators.required],
        script: ['', Validators.required]
      }),
      inputs: this.fb.array([]),
      outputs: this.fb.array([]),
      plots: this.fb.array([]),
      execution: this.fb.group({
        argsMapping: this.fb.group({}),
        outputDir: ['--output_folder', Validators.required],
        requirements: this.fb.group({
          python: [''],
          r: [''],
          packages: this.fb.array([])
        })
      }),
      annotation: this.fb.group({
        samplesFrom: [''],
        annotationFile: ['']
      }),
      example: this.fb.group({
        enabled: [false],
        values: this.fb.group({})
      })
    });
  }

  get inputs(): FormArray {
    return this.pluginForm.get('inputs') as FormArray;
  }

  get outputs(): FormArray {
    return this.pluginForm.get('outputs') as FormArray;
  }

  get plots(): FormArray {
    return this.pluginForm.get('plots') as FormArray;
  }

  get packages(): FormArray {
    return this.pluginForm.get('execution.requirements.packages') as FormArray;
  }

  async loadPlugin(pluginID: string) {
    this.mode.set('edit');
    this.loading.set(true);

    try {
      const id = parseInt(pluginID, 10);
      if (isNaN(id)) {
        throw new Error('Invalid plugin ID');
      }
      const plugin = await this.wails.getPlugin(id);
      this.populateForm(plugin.definition);
      await this.wails.logToFile(`[PluginEditor] Loaded plugin: ${pluginID}`);
    } catch (error) {
      await this.wails.logToFile(`[PluginEditor] Error loading plugin: ${error}`);
    } finally {
      this.loading.set(false);
    }
  }

  async cloneFromTemplate(templateID: string) {
    this.mode.set('create');
    this.loading.set(true);

    try {
      const definition = await this.pluginEditorService.createFromTemplate(
        templateID,
        '',
        `Copy of ${templateID}`
      );
      this.populateForm(definition);
      await this.wails.logToFile(`[PluginEditor] Cloned from template: ${templateID}`);
    } catch (error) {
      await this.wails.logToFile(`[PluginEditor] Error cloning template: ${error}`);
    } finally {
      this.loading.set(false);
    }
  }

  private populateForm(definition: models.PluginDefinition) {
    this.pluginForm.patchValue({
      plugin: definition.plugin,
      runtime: definition.runtime,
      execution: {
        outputDir: definition.execution?.outputDir || '--output_folder',
        requirements: definition.execution?.requirements || {}
      }
    });

    this.inputs.clear();
    if (definition.inputs) {
      definition.inputs.forEach(input => {
        this.inputs.push(this.createInputFormGroup(input));
      });
    }

    this.outputs.clear();
    if (definition.outputs) {
      definition.outputs.forEach(output => {
        this.outputs.push(this.createOutputFormGroup(output));
      });
    }
  }

  private createInputFormGroup(input?: any): FormGroup {
    return this.fb.group({
      name: [input?.name || '', [Validators.required, Validators.pattern(/^[a-zA-Z][a-zA-Z0-9_]*$/)]],
      label: [input?.label || '', Validators.required],
      type: [input?.type || 'text', Validators.required],
      required: [input?.required || false],
      default: [input?.default || null],
      description: [input?.description || ''],
      placeholder: [input?.placeholder || '']
    });
  }

  private createOutputFormGroup(output?: any): FormGroup {
    return this.fb.group({
      name: [output?.name || '', Validators.required],
      path: [output?.path || '', Validators.required],
      type: [output?.type || 'data', Validators.required],
      description: [output?.description || ''],
      format: [output?.format || 'tsv']
    });
  }

  addInput() {
    this.inputs.push(this.createInputFormGroup());
  }

  removeInput(index: number) {
    this.inputs.removeAt(index);
  }

  addOutput() {
    this.outputs.push(this.createOutputFormGroup());
  }

  removeOutput(index: number) {
    this.outputs.removeAt(index);
  }

  async generatePreview() {
    this.validating.set(true);

    try {
      const definition = this.buildPluginDefinition();
      const yaml = await this.pluginEditorService.definitionToYAML(definition);
      this.yamlPreview.set(yaml);

      const validation = await this.pluginEditorService.validatePlugin(definition);
      this.validationResult.set(validation);

      await this.wails.logToFile(`[PluginEditor] Generated preview, valid: ${validation.valid}`);
    } catch (error) {
      this.validationResult.set({
        valid: false,
        errors: [`Preview generation error: ${error}`]
      });
      await this.wails.logToFile(`[PluginEditor] Preview error: ${error}`);
    } finally {
      this.validating.set(false);
    }
  }

  private buildPluginDefinition(): models.PluginDefinition {
    const formValue = this.pluginForm.value;

    return new models.PluginDefinition({
      plugin: new models.PluginMetadata(formValue.plugin),
      runtime: new models.PluginRuntimeV2(formValue.runtime),
      inputs: formValue.inputs.map((i: any) => new models.PluginInputV2(i)),
      outputs: formValue.outputs.map((o: any) => new models.PluginOutputV2(o)),
      plots: [],
      execution: new models.PluginExecution({
        argsMapping: {},
        outputDir: formValue.execution.outputDir,
        requirements: formValue.execution.requirements
      })
    });
  }

  async savePlugin() {
    if (this.pluginForm.invalid) {
      await this.wails.logToFile('[PluginEditor] Form is invalid');
      return;
    }

    this.saving.set(true);

    try {
      const definition = this.buildPluginDefinition();
      const pluginID = definition.plugin.id;

      await this.pluginEditorService.savePlugin(pluginID, definition);
      await this.wails.logToFile(`[PluginEditor] Saved plugin: ${pluginID}`);

      this.router.navigate(['/plugin-list']);
    } catch (error) {
      await this.wails.logToFile(`[PluginEditor] Save error: ${error}`);
    } finally {
      this.saving.set(false);
    }
  }

  cancel() {
    this.router.navigate(['/plugin-list']);
  }

  async onPluginNameChange(name: string) {
    if (this.mode() === 'create' && !this.pluginForm.get('plugin.id')?.dirty) {
      const generatedID = this.pluginEditorService.generatePluginID(name);
      this.pluginForm.patchValue({
        plugin: { id: generatedID }
      });
    }
  }
}
