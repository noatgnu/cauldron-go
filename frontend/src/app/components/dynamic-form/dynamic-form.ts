import { Component, Input, Output, EventEmitter, OnInit, OnChanges, OnDestroy, SimpleChanges, signal, computed } from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatDialog } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { models } from '../../../wailsjs/go/models';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { SampleAnnotation } from '../sample-annotation/sample-annotation';
import { GenericTableEditor, TableColumn } from '../generic-table-editor/generic-table-editor';
import { ImportedFileSelectionDialog } from '../imported-file-selection/imported-file-selection-dialog';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-dynamic-form',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTooltipModule
  ],
  templateUrl: './dynamic-form.html',
  styleUrl: './dynamic-form.scss'
})
export class DynamicFormComponent implements OnInit, OnChanges, OnDestroy {
  @Input() plugin!: models.PluginV2;
  @Input() disabled = false;
  @Output() formSubmit = new EventEmitter<Record<string, any>>();
  @Output() formChange = new EventEmitter<Record<string, any>>();

  form!: FormGroup;
  formKey = signal(0);
  columnOptions = new Map<string, string[]>();
  selectOptions = new Map<string, string[]>();
  groupedOptions = new Map<string, models.FieldGroup[]>();
  loading = signal(false);
  formValues = signal<Record<string, any>>({});
  validationErrors = signal<string[]>([]);
  lastSelectedIndex = new Map<string, number>();
  private valueChangesSubscription?: Subscription;

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private notificationService: NotificationService,
    private dialog: MatDialog
  ) {}

  async ngOnInit() {
    await this.initializeForm();
  }

  async ngOnChanges(changes: SimpleChanges) {
    if (changes['plugin'] && !changes['plugin'].firstChange) {
      await this.wails.logToFile(`[DynamicForm] Plugin changed from ${changes['plugin'].previousValue?.definition.plugin.id} to ${changes['plugin'].currentValue?.definition.plugin.id}`);
      this.formKey.update(k => k + 1);
      await this.initializeForm();
    }
  }

  ngOnDestroy() {
    this.valueChangesSubscription?.unsubscribe();
  }

  private async initializeForm() {
    this.valueChangesSubscription?.unsubscribe();

    this.columnOptions.clear();
    this.selectOptions.clear();
    this.groupedOptions.clear();
    this.lastSelectedIndex.clear();
    this.validationErrors.set([]);

    this.buildForm();
    await this.loadExternalOptions();

    this.valueChangesSubscription = this.form.valueChanges.subscribe((values) => {
      this.formValues.set(values);
      this.formChange.emit(this.getFormValue());
      if (this.validationErrors().length > 0) {
        this.validationErrors.set([]);
      }
    });
  }

  private buildForm() {
    const group: any = {};

    for (const input of this.plugin.definition.inputs) {
      const validators = [];

      if (input.required) {
        validators.push(Validators.required);
      }

      if (input.type === 'number') {
        if (input.min !== undefined && input.min !== null) {
          validators.push(Validators.min(input.min));
        }
        if (input.max !== undefined && input.max !== null) {
          validators.push(Validators.max(input.max));
        }
      }

      const defaultValue = this.getDefaultValue(input);
      group[input.name] = [defaultValue, validators];
    }

    this.form = this.fb.group(group);
  }

  private getDefaultValue(input: models.PluginInputV2): any {
    if (input.default !== undefined && input.default !== null) {
      return input.default;
    }

    switch (input.type) {
      case 'boolean':
        return false;
      case 'number':
        return input.min || 0;
      case 'column-selector':
        return input.multiple ? [] : '';
      case 'multiselect-grouped':
        return [];
      default:
        return '';
    }
  }

  async openFile(inputName: string) {
    const filePath = await this.wails.openFileDialog('Select File');
    if (filePath) {
      this.form.patchValue({ [inputName]: filePath });

      const input = this.plugin.definition.inputs.find(i => i.name === inputName);
      if (input) {
        await this.loadColumnsForDependents(inputName, filePath);
      }
    }
  }

  async openImportedFileSelector(inputName: string) {
    const dialogRef = this.dialog.open(ImportedFileSelectionDialog, {
      width: '600px',
      disableClose: false
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && result.filePath) {
        this.form.patchValue({ [inputName]: result.filePath });

        const input = this.plugin.definition.inputs.find(i => i.name === inputName);
        if (input && result.columns) {
          await this.loadColumnsForDependentsWithHeaders(inputName, result.filePath, result.columns);
        } else if (input) {
          await this.loadColumnsForDependents(inputName, result.filePath);
        }
      }
    });
  }

  private async loadColumnsForDependentsWithHeaders(sourceInputName: string, filePath: string, headers: string[]) {
    const dependentInputs = this.plugin.definition.inputs.filter(
      i => i.sourceFile === sourceInputName
    );

    for (const input of dependentInputs) {
      await this.wails.logToFile(`[DynamicForm] Setting columns for ${input.name} from imported file headers`);
      this.columnOptions.set(input.name, headers);
    }
  }

  private async loadColumnsForDependents(sourceInputName: string, filePath: string) {
    const dependentInputs = this.plugin.definition.inputs.filter(
      i => i.sourceFile === sourceInputName
    );

    for (const input of dependentInputs) {
      await this.loadColumns(input.name, filePath);
    }
  }

  private async loadColumns(inputName: string, filePath: string) {
    try {
      this.loading.set(true);
      const content = await this.wails.readFile(filePath);
      if (!content) return;

      const lines = content.split('\n');
      if (lines.length === 0) return;

      const firstLine = lines[0].trim();
      const delimiter = firstLine.includes('\t') ? '\t' : ',';
      const headers = firstLine.split(delimiter).map(h => h.trim());

      await this.wails.logToFile(`[DynamicForm] Loaded ${headers.length} columns for ${inputName}: ${headers.join(', ')}`);
      this.columnOptions.set(inputName, headers);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`Error loading columns: ${errorMsg}`);
      this.notificationService.showError(`Failed to load columns: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  getColumns(inputName: string): string[] {
    return this.columnOptions.get(inputName) || [];
  }

  private async loadExternalOptions() {
    for (const input of this.plugin.definition.inputs) {
      if (input.type === 'select' && input.optionsFromFile) {
        await this.loadOptionsFromFile(input.name, input.optionsFromFile);
      } else if (input.type === 'multiselect-grouped' && input.groupsFromFile) {
        await this.loadGroupsFromFile(input.name, input.groupsFromFile);
      }
    }
  }

  private async loadOptionsFromFile(inputName: string, filePath: string) {
    try {
      this.loading.set(true);

      const fullPath = this.plugin.folderPath
        ? `${this.plugin.folderPath}/${filePath}`
        : filePath;

      await this.wails.logToFile(`[DynamicForm] Loading options from: ${fullPath}`);
      const content = await this.wails.readFile(fullPath);
      if (!content) return;

      const options = content
        .split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      this.selectOptions.set(inputName, options);
      await this.wails.logToFile(`[DynamicForm] Loaded ${options.length} options for ${inputName}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`[DynamicForm] Error loading options from file: ${errorMsg}`);
      this.notificationService.showError(`Failed to load options: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  private async loadGroupsFromFile(inputName: string, filePath: string) {
    try {
      this.loading.set(true);

      const fullPath = this.plugin.folderPath
        ? `${this.plugin.folderPath}/${filePath}`
        : filePath;

      await this.wails.logToFile(`[DynamicForm] Loading groups from: ${fullPath}`);
      const content = await this.wails.readFile(fullPath);
      if (!content) return;

      const groups: models.FieldGroup[] = JSON.parse(content);
      this.groupedOptions.set(inputName, groups);
      await this.wails.logToFile(`[DynamicForm] Loaded ${groups.length} groups for ${inputName}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`[DynamicForm] Error loading groups from file: ${errorMsg}`);
      this.notificationService.showError(`Failed to load grouped options: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  getSelectOptions(input: models.PluginInputV2): string[] {
    if (input.optionsFromFile) {
      return this.selectOptions.get(input.name) || [];
    }
    return input.options || [];
  }

  getGroupedOptions(inputName: string): models.FieldGroup[] {
    return this.groupedOptions.get(inputName) || [];
  }

  getInputsByType(type: string): models.PluginInputV2[] {
    return this.plugin.definition.inputs.filter(i => i.type === type);
  }

  isInputVisible(input: models.PluginInputV2): boolean {
    if (!input.visibleWhen) {
      return true;
    }

    const condition = input.visibleWhen;
    const currentValue = this.formValues()[condition.field];

    if (condition.equals !== undefined && condition.equals !== null) {
      return currentValue === condition.equals;
    }

    if (condition.equalsAny && Array.isArray(condition.equalsAny)) {
      return condition.equalsAny.includes(currentValue);
    }

    return true;
  }

  submit() {
    if (this.form.valid) {
      this.validationErrors.set([]);
      this.formSubmit.emit(this.getFormValue());
    } else {
      const errors: string[] = [];
      Object.keys(this.form.controls).forEach(key => {
        const control = this.form.get(key);
        if (control && control.invalid && control.errors) {
          const input = this.plugin.definition.inputs.find(i => i.name === key);
          const label = input?.label || key;

          if (control.errors['required']) {
            errors.push(`${label} is required`);
          }
          if (control.errors['min']) {
            errors.push(`${label} must be at least ${control.errors['min'].min}`);
          }
          if (control.errors['max']) {
            errors.push(`${label} must be at most ${control.errors['max'].max}`);
          }
        }
      });
      this.validationErrors.set(errors);
      this.notificationService.showError('Please fix validation errors before submitting');
    }
  }

  reset() {
    this.form.reset();
    this.columnOptions.clear();
    this.selectOptions.clear();
    this.groupedOptions.clear();
    this.validationErrors.set([]);
  }

  async loadExample() {
    const example = this.plugin.definition.example;
    if (!example || !example.enabled) {
      this.notificationService.showError('No example data available for this plugin');
      return;
    }

    try {
      this.loading.set(true);
      const valuesToSet: Record<string, any> = {};

      for (const [key, value] of Object.entries(example.values)) {
        if (key.endsWith('_source')) {
          const targetField = key.replace('_source', '');
          const input = this.plugin.definition.inputs.find(i => i.name === targetField);

          if (input && input.type === 'column-selector') {
            const filePath = await this.wails.getPluginExampleFilePath(this.plugin.definition.plugin.id, value as string);
            await this.loadColumns(targetField, filePath);

            if (!example.values[targetField]) {
              const content = await this.wails.readFile(filePath);
              if (content) {
                const lines = content.split('\n');
                if (lines.length > 0) {
                  const headers = lines[0].split('\t').map(h => h.trim());
                  const sampleColumns = input.multiple ? headers.slice(0, 10) : headers[0];
                  valuesToSet[targetField] = sampleColumns;
                }
              }
            }
          }
        } else {
          const input = this.plugin.definition.inputs.find(i => i.name === key);

          if (input && input.type === 'file') {
            const filePath = await this.wails.getPluginExampleFilePath(this.plugin.definition.plugin.id, value as string);
            valuesToSet[key] = filePath;

            await this.loadColumnsForDependents(key, filePath);
          } else {
            valuesToSet[key] = value;
          }
        }
      }

      this.form.patchValue(valuesToSet);
      this.notificationService.showSuccess('Example data loaded successfully');
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`Error loading example: ${errorMsg}`);
      this.notificationService.showError(`Failed to load example data: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadFromJobParameters(parameters: Record<string, any>) {
    try {
      this.loading.set(true);
      await this.wails.logToFile(`[DynamicForm] Loading parameters from job: ${JSON.stringify(parameters)}`);
      const valuesToSet: Record<string, any> = {};

      for (const [key, value] of Object.entries(parameters)) {
        const input = this.plugin.definition.inputs.find(i => i.name === key);

        if (input) {
          await this.wails.logToFile(`[DynamicForm] Processing parameter ${key}: type=${input.type}, value=${value} (${typeof value})`);

          if (input.type === 'file' && typeof value === 'string') {
            valuesToSet[key] = value;
            await this.loadColumnsForDependents(key, value);
          } else if (input.type === 'column' || input.type === 'column-selector') {
            const sourceFile = parameters[`${key}_source`];
            if (sourceFile && typeof sourceFile === 'string') {
              await this.loadColumns(key, sourceFile);
            }
            valuesToSet[key] = value;
          } else if (input.type === 'boolean' || input.type === 'checkbox') {
            let boolValue: boolean;
            if (typeof value === 'string') {
              boolValue = value === 'true' || value === '1' || value === 'True';
            } else if (typeof value === 'number') {
              boolValue = value !== 0;
            } else {
              boolValue = !!value;
            }
            valuesToSet[key] = boolValue;
            await this.wails.logToFile(`[DynamicForm] Converted boolean parameter ${key}: ${value} (${typeof value}) -> ${boolValue}`);
          } else {
            valuesToSet[key] = value;
          }
        } else {
          await this.wails.logToFile(`[DynamicForm] Skipping parameter ${key}: no matching input found`);
        }
      }

      await this.wails.logToFile(`[DynamicForm] Values to set: ${JSON.stringify(valuesToSet)}`);
      this.form.patchValue(valuesToSet);
      this.notificationService.showSuccess('Job parameters loaded successfully');
      await this.wails.logToFile(`[DynamicForm] Successfully loaded job parameters`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.wails.logToFile(`[DynamicForm] Error loading job parameters: ${errorMsg}`);
      this.notificationService.showError(`Failed to load job parameters: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  private getFormValue(): Record<string, any> {
    const value: Record<string, any> = {};

    for (const key of Object.keys(this.form.value)) {
      const val = this.form.value[key];
      if (val !== null && val !== undefined && val !== '') {
        value[key] = val;
      }
    }

    return value;
  }

  handleOptionClick(event: MouseEvent, fieldName: string, optionValue: string, allOptions: string[]) {
    const currentIndex = allOptions.indexOf(optionValue);
    const control = this.form.get(fieldName);

    if (!control) return;

    const currentValue: string[] = Array.isArray(control.value) ? [...control.value] : [];

    if (event.shiftKey && this.lastSelectedIndex.has(fieldName)) {
      const lastIndex = this.lastSelectedIndex.get(fieldName)!;
      const start = Math.min(lastIndex, currentIndex);
      const end = Math.max(lastIndex, currentIndex);

      const rangeValues = allOptions.slice(start, end + 1);

      const newValue = [...new Set([...currentValue, ...rangeValues])];
      control.setValue(newValue);
    } else {
      if (currentValue.includes(optionValue)) {
        const newValue = currentValue.filter(v => v !== optionValue);
        control.setValue(newValue);
      } else {
        control.setValue([...currentValue, optionValue]);
      }
    }

    this.lastSelectedIndex.set(fieldName, currentIndex);

    event.preventDefault();
    event.stopPropagation();
  }

  isAnnotationFileInput(input: models.PluginInputV2): boolean {
    if (input.type !== 'file') return false;
    if ((input as any).disableAnnotationManagement === true) return false;
    const name = input.name.toLowerCase();
    const label = input.label?.toLowerCase() || '';
    return name.includes('annotation') ||
           name.includes('metadata') ||
           label.includes('annotation') ||
           label.includes('metadata') ||
           label.includes('sample annotation');
  }

  hasTableColumns(input: models.PluginInputV2): boolean {
    return input.type === 'file' && !!input.tableColumns && input.tableColumns.length > 0;
  }

  async openAnnotationManager(inputName: string) {
    const sampleNames = this.getSampleNamesForAnnotation();
    if (!sampleNames || sampleNames.length === 0) {
      this.notificationService.showWarning('Please select sample columns first to create an annotation file');
      return;
    }

    const currentFilePath = this.form.get(inputName)?.value;
    let existingAnnotation: any[] | undefined;

    if (currentFilePath) {
      try {
        const content = await this.wails.readFile(currentFilePath);
        existingAnnotation = this.parseAnnotationFile(content, sampleNames);
      } catch (error) {
        await this.wails.logToFile(`Error reading annotation file: ${error}`);
      }
    }

    const dialogRef = this.dialog.open(SampleAnnotation, {
      width: '90vw',
      maxWidth: '1400px',
      height: '80vh',
      disableClose: true,
      data: {
        samples: existingAnnotation ? undefined : sampleNames,
        annotation: existingAnnotation,
        mode: existingAnnotation ? 'edit' : 'create'
      }
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && Array.isArray(result)) {
        try {
          const filePath = await this.saveAnnotationFile(result);
          this.form.patchValue({ [inputName]: filePath });
          this.notificationService.showSuccess('Annotation file saved successfully');
        } catch (error) {
          const errorMsg = error instanceof Error ? error.message : String(error);
          await this.wails.logToFile(`Error saving annotation file: ${errorMsg}`);
          this.notificationService.showError(`Failed to save annotation file: ${errorMsg}`);
        }
      }
    });
  }

  private getSampleNamesForAnnotation(): string[] {
    const columnSelectors = this.plugin.definition.inputs.filter(
      i => i.type === 'column-selector' && i.multiple
    );

    for (const selector of columnSelectors) {
      const value = this.form.get(selector.name)?.value;
      if (Array.isArray(value) && value.length > 0) {
        return value;
      }
    }

    return [];
  }

  private parseAnnotationFile(content: string, sampleNames: string[]): any[] {
    const lines = content.trim().split('\n');
    if (lines.length < 2) {
      console.log('[DynamicForm] Annotation file has less than 2 lines');
      return [];
    }

    const headers = lines[0].split('\t').map(h => h.trim());
    console.log('[DynamicForm] Annotation headers:', headers);
    const sampleIdx = headers.findIndex(h => h.toLowerCase().includes('sample'));
    const conditionIdx = headers.findIndex(h => h.toLowerCase().includes('condition') || h.toLowerCase().includes('group'));
    const bioreplicateIdx = headers.findIndex(h => h.toLowerCase().includes('bioreplicate') || h.toLowerCase().includes('replicate'));
    const batchIdx = headers.findIndex(h => h.toLowerCase().includes('batch'));
    const colorIdx = headers.findIndex(h => h.toLowerCase().includes('color'));

    if (sampleIdx === -1) {
      console.log('[DynamicForm] No Sample column found in annotation file');
      return [];
    }

    const annotations = [];
    for (let i = 1; i < lines.length; i++) {
      const values = lines[i].split('\t');
      annotations.push({
        Sample: values[sampleIdx] || '',
        Condition: conditionIdx !== -1 ? (values[conditionIdx] || '') : '',
        BioReplicate: bioreplicateIdx !== -1 ? (values[bioreplicateIdx] || '') : '',
        Batch: batchIdx !== -1 ? (values[batchIdx] || '') : '',
        Color: colorIdx !== -1 ? (values[colorIdx] || '') : ''
      });
    }

    console.log('[DynamicForm] Parsed annotations:', annotations.length);
    return annotations;
  }

  private async saveAnnotationFile(annotations: any[]): Promise<string> {
    const headers = ['Sample', 'Condition', 'BioReplicate', 'Batch', 'Color'];
    const rows = annotations.map(a =>
      [a.Sample || '', a.Condition || '', a.BioReplicate || '', a.Batch || '', a.Color || ''].join('\t')
    );
    const content = [headers.join('\t'), ...rows].join('\n');

    const timestamp = Date.now();
    const filename = `annotation_${timestamp}.txt`;
    const filePath = await this.wails.saveTempFile(filename, content);

    return filePath;
  }

  async openTableEditor(inputName: string) {
    const input = this.plugin.definition.inputs.find(i => i.name === inputName);
    if (!input || !input.tableColumns) return;

    const currentFilePath = this.form.get(inputName)?.value;
    let existingData: any[] | undefined;

    if (currentFilePath) {
      try {
        const content = await this.wails.readFile(currentFilePath);
        existingData = this.parseTableFile(content, input.tableColumns);
      } catch (error) {
        await this.wails.logToFile(`Error reading table file: ${error}`);
      }
    }

    const dialogRef = this.dialog.open(GenericTableEditor, {
      width: '90vw',
      maxWidth: '1400px',
      height: '80vh',
      disableClose: true,
      data: {
        columns: input.tableColumns,
        data: existingData,
        title: input.label || 'Edit Table',
        mode: existingData ? 'edit' : 'create'
      }
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && Array.isArray(result) && input.tableColumns) {
        try {
          const filePath = await this.saveTableFile(result, input.tableColumns, inputName);
          this.form.patchValue({ [inputName]: filePath });
          this.notificationService.showSuccess('Table file saved successfully');
        } catch (error) {
          const errorMsg = error instanceof Error ? error.message : String(error);
          await this.wails.logToFile(`Error saving table file: ${errorMsg}`);
          this.notificationService.showError(`Failed to save table file: ${errorMsg}`);
        }
      }
    });
  }

  private parseTableFile(content: string, columns: models.TableColumn[]): any[] {
    const lines = content.trim().split('\n');
    if (lines.length < 2) return [];

    const headers = lines[0].split('\t').map(h => h.trim());
    const data: any[] = [];

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line) continue;

      const values = lines[i].split('\t');
      const row: any = {};
      columns.forEach((col, idx) => {
        const headerIdx = headers.findIndex(h => h.toLowerCase() === col.name.toLowerCase());
        row[col.name] = headerIdx !== -1 ? (values[headerIdx] || '').trim() : '';
      });

      const hasNonEmptyValue = columns.some(col => {
        const value = row[col.name];
        return value !== null && value !== undefined && value !== '';
      });

      if (hasNonEmptyValue) {
        data.push(row);
      }
    }

    return data;
  }

  private async saveTableFile(data: any[], columns: models.TableColumn[], inputName: string): Promise<string> {
    const headers = columns.map(c => c.name);

    const filteredData = data.filter(row => {
      return columns.some(col => {
        const value = row[col.name];
        return value !== null && value !== undefined && value !== '';
      });
    });

    const rows = filteredData.map(row =>
      columns.map(col => row[col.name] || '').join('\t')
    );
    const content = [headers.join('\t'), ...rows].join('\n');

    const timestamp = Date.now();
    const filename = `${inputName}_${timestamp}.txt`;
    const filePath = await this.wails.saveTempFile(filename, content);

    return filePath;
  }
}
