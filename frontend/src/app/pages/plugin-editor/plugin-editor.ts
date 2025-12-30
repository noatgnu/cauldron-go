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
        subcategory: [''],
        icon: [''],
        repository: ['']
      }),
      runtime: this.fb.group({
        environments: this.fb.array([this.fb.control('python')]),
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
          packages: this.fb.array([]),
          pythonRequirementsFile: [''],
          rPackagesFile: ['']
        }),
        envVariables: this.fb.array([])
      }),
      annotation: this.fb.group({
        samplesFrom: [''],
        annotationFile: ['']
      }),
      example: this.fb.group({
        enabled: [false],
        values: this.fb.group({})
      }),
      diagram: this.fb.group({
        enabled: [false]
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

  get environments(): FormArray {
    return this.pluginForm.get('runtime.environments') as FormArray;
  }

  get envVariables(): FormArray {
    return this.pluginForm.get('execution.envVariables') as FormArray;
  }

  async loadPlugin(pluginID: string) {
    this.mode.set('edit');
    this.loading.set(true);

    try {
      const plugins = await this.wails.getPluginsV2();
      const plugin = plugins.find((p: any) => p.id === parseInt(pluginID, 10));

      if (!plugin) {
        throw new Error(`Plugin not found: ${pluginID}`);
      }

      if (plugin.definition) {
        this.populateForm(plugin.definition);
      } else {
        throw new Error('Plugin definition not available');
      }

      await this.wails.logToFile(`[PluginEditor] Loaded pluginV2: ${pluginID}`);
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
      execution: {
        outputDir: definition.execution?.outputDir || '--output_folder',
        requirements: definition.execution?.requirements || {}
      },
      annotation: definition.annotation || {},
      example: definition.example || { enabled: false },
      diagram: { enabled: false }
    });

    this.environments.clear();
    if (definition.runtime?.environments && definition.runtime.environments.length > 0) {
      definition.runtime.environments.forEach((env: string) => {
        this.environments.push(this.fb.control(env));
      });
    } else {
      this.environments.push(this.fb.control('python'));
    }

    this.pluginForm.patchValue({
      runtime: {
        script: definition.runtime?.script || ''
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

    this.packages.clear();
    if (definition.execution?.requirements?.packages) {
      definition.execution.requirements.packages.forEach((pkg: string) => {
        this.packages.push(this.fb.control(pkg));
      });
    }

    this.envVariables.clear();
    if (definition.execution?.envVariables) {
      definition.execution.envVariables.forEach((envVar: any) => {
        this.envVariables.push(this.createEnvVariableFormGroup(envVar));
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
      placeholder: [input?.placeholder || ''],
      accept: [input?.accept || ''],
      options: this.fb.array(input?.options?.map((opt: string) => this.fb.control(opt)) || []),
      optionsFromFile: [input?.optionsFromFile || ''],
      groups: this.fb.array(input?.groups?.map((g: any) => this.createGroupFormGroup(g)) || []),
      groupsFromFile: [input?.groupsFromFile || ''],
      multiple: [input?.multiple || false],
      sourceFile: [input?.sourceFile || ''],
      min: [input?.min ?? null],
      max: [input?.max ?? null],
      step: [input?.step ?? null],
      visibleWhen: this.fb.group({
        field: [input?.visibleWhen?.field || ''],
        equals: [input?.visibleWhen?.equals ?? null],
        equalsAny: this.fb.array(input?.visibleWhen?.equalsAny?.map((v: any) => this.fb.control(v)) || [])
      }),
      disableAnnotationManagement: [input?.disableAnnotationManagement || false],
      tableColumns: this.fb.array(input?.tableColumns?.map((col: any) => this.createTableColumnFormGroup(col)) || [])
    });
  }

  private createGroupFormGroup(group?: any): FormGroup {
    return this.fb.group({
      name: [group?.name || '', Validators.required],
      options: this.fb.array(group?.options?.map((opt: any) => this.createFieldOptionFormGroup(opt)) || [])
    });
  }

  private createFieldOptionFormGroup(option?: any): FormGroup {
    return this.fb.group({
      value: [option?.value || '', Validators.required],
      label: [option?.label || '', Validators.required]
    });
  }

  private createTableColumnFormGroup(column?: any): FormGroup {
    return this.fb.group({
      name: [column?.name || '', [Validators.required, Validators.pattern(/^[a-zA-Z][a-zA-Z0-9_]*$/)]],
      label: [column?.label || '', Validators.required],
      type: [column?.type || 'text'],
      required: [column?.required || false],
      description: [column?.description || '']
    });
  }

  private createEnvVariableFormGroup(envVar?: any): FormGroup {
    return this.fb.group({
      name: [envVar?.name || '', [Validators.required, Validators.pattern(/^[a-zA-Z][a-zA-Z0-9_]*$/)]],
      label: [envVar?.label || '', Validators.required],
      type: [envVar?.type || 'text', Validators.required],
      required: [envVar?.required || false],
      default: [envVar?.default || null],
      description: [envVar?.description || '']
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

  addEnvironment() {
    this.environments.push(this.fb.control('python'));
  }

  removeEnvironment(index: number) {
    if (this.environments.length > 1) {
      this.environments.removeAt(index);
    }
  }

  addPackage() {
    this.packages.push(this.fb.control(''));
  }

  removePackage(index: number) {
    this.packages.removeAt(index);
  }

  addEnvVariable() {
    this.envVariables.push(this.createEnvVariableFormGroup());
  }

  removeEnvVariable(index: number) {
    this.envVariables.removeAt(index);
  }

  getInputOptions(inputIndex: number): FormArray {
    return this.inputs.at(inputIndex).get('options') as FormArray;
  }

  addInputOption(inputIndex: number) {
    this.getInputOptions(inputIndex).push(this.fb.control(''));
  }

  removeInputOption(inputIndex: number, optionIndex: number) {
    this.getInputOptions(inputIndex).removeAt(optionIndex);
  }

  getInputGroups(inputIndex: number): FormArray {
    return this.inputs.at(inputIndex).get('groups') as FormArray;
  }

  addInputGroup(inputIndex: number) {
    this.getInputGroups(inputIndex).push(this.createGroupFormGroup());
  }

  removeInputGroup(inputIndex: number, groupIndex: number) {
    this.getInputGroups(inputIndex).removeAt(groupIndex);
  }

  getGroupOptions(inputIndex: number, groupIndex: number): FormArray {
    return this.getInputGroups(inputIndex).at(groupIndex).get('options') as FormArray;
  }

  addGroupOption(inputIndex: number, groupIndex: number) {
    this.getGroupOptions(inputIndex, groupIndex).push(this.createFieldOptionFormGroup());
  }

  removeGroupOption(inputIndex: number, groupIndex: number, optionIndex: number) {
    this.getGroupOptions(inputIndex, groupIndex).removeAt(optionIndex);
  }

  getInputTableColumns(inputIndex: number): FormArray {
    return this.inputs.at(inputIndex).get('tableColumns') as FormArray;
  }

  addInputTableColumn(inputIndex: number) {
    this.getInputTableColumns(inputIndex).push(this.createTableColumnFormGroup());
  }

  removeInputTableColumn(inputIndex: number, columnIndex: number) {
    this.getInputTableColumns(inputIndex).removeAt(columnIndex);
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
      runtime: new models.PluginRuntimeV2({
        environments: formValue.runtime.environments,
        script: formValue.runtime.script
      }),
      inputs: formValue.inputs.map((i: any) => new models.PluginInputV2(i)),
      outputs: formValue.outputs.map((o: any) => new models.PluginOutputV2(o)),
      plots: [],
      execution: new models.PluginExecution({
        argsMapping: {},
        outputDir: formValue.execution.outputDir,
        requirements: formValue.execution.requirements,
        envVariables: formValue.execution.envVariables
      }),
      annotation: formValue.annotation?.samplesFrom || formValue.annotation?.annotationFile ? formValue.annotation : undefined,
      example: formValue.example?.enabled ? formValue.example : undefined,
      diagram: formValue.diagram
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
