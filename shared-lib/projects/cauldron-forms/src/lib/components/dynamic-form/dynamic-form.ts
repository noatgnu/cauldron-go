import {
  Component,
  Input,
  Output,
  EventEmitter,
  OnInit,
  OnChanges,
  OnDestroy,
  SimpleChanges,
  signal,
  Inject,
  Optional
} from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Subscription } from 'rxjs';
import {
  PluginV2,
  PluginInputV2,
  FieldGroup,
  FieldOption,
  SelectOption,
  TableColumn
} from '../../models/plugin.models';
import { FileHandler, ImportedFile } from '../../interfaces/file-handler.interface';
import { NotificationHandler } from '../../interfaces/notification.interface';
import { LogHandler } from '../../interfaces/log.interface';
import { FILE_HANDLER, NOTIFICATION_HANDLER, LOG_HANDLER } from '../../tokens/injection-tokens';
import { SampleAnnotation } from '../sample-annotation/sample-annotation';
import { GenericTableEditor } from '../generic-table-editor/generic-table-editor';
import { ImportedFileSelectionDialog } from '../imported-file-selection/imported-file-selection-dialog';

export interface ExampleFilePathResolver {
  getPluginExampleFilePath(pluginId: string, filename: string): Promise<string>;
}

export const EXAMPLE_FILE_PATH_RESOLVER = Symbol('EXAMPLE_FILE_PATH_RESOLVER');

@Component({
  selector: 'cld-dynamic-form',
  standalone: true,
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
    MatTooltipModule,
    MatDialogModule
  ],
  templateUrl: './dynamic-form.html',
  styleUrl: './dynamic-form.scss'
})
export class DynamicFormComponent implements OnInit, OnChanges, OnDestroy {
  @Input() plugin!: PluginV2;
  @Input() disabled = false;
  @Input() exampleFilePathResolver?: ExampleFilePathResolver;
  @Output() formSubmit = new EventEmitter<Record<string, unknown>>();
  @Output() formChange = new EventEmitter<Record<string, unknown>>();

  form!: FormGroup;
  formKey = signal(0);
  columnOptions = new Map<string, string[]>();
  selectOptions = new Map<string, SelectOption[]>();
  groupedOptions = new Map<string, FieldGroup[]>();
  loading = signal(false);
  formValues = signal<Record<string, unknown>>({});
  validationErrors = signal<string[]>([]);
  lastSelectedIndex = new Map<string, number>();
  private valueChangesSubscription?: Subscription;

  constructor(
    private fb: FormBuilder,
    @Inject(FILE_HANDLER) private fileHandler: FileHandler,
    @Inject(NOTIFICATION_HANDLER) private notification: NotificationHandler,
    @Optional() @Inject(LOG_HANDLER) private logger: LogHandler | null,
    private dialog: MatDialog
  ) {}

  async ngOnInit() {
    await this.initializeForm();
  }

  async ngOnChanges(changes: SimpleChanges) {
    if (changes['plugin'] && !changes['plugin'].firstChange) {
      await this.log(`[DynamicForm] Plugin changed from ${changes['plugin'].previousValue?.definition.plugin.id} to ${changes['plugin'].currentValue?.definition.plugin.id}`);
      this.formKey.update(k => k + 1);
      await this.initializeForm();
    }
  }

  ngOnDestroy() {
    this.valueChangesSubscription?.unsubscribe();
  }

  private async log(message: string): Promise<void> {
    if (this.logger) {
      await this.logger.log(message);
    }
  }

  private async logError(message: string): Promise<void> {
    if (this.logger) {
      await this.logger.error(message);
    }
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
    const group: Record<string, unknown[]> = {};

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

  private getDefaultValue(input: PluginInputV2): unknown {
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
    const filePath = await this.fileHandler.openFileDialog('Select File');
    if (filePath) {
      this.form.patchValue({ [inputName]: filePath });

      const input = this.plugin.definition.inputs.find(i => i.name === inputName);
      if (input) {
        await this.loadColumnsForDependents(inputName, filePath);
      }
    }
  }

  async openDirectory(inputName: string) {
    const directoryPath = await this.fileHandler.openDirectoryDialog('Select Directory');
    if (directoryPath) {
      this.form.patchValue({ [inputName]: directoryPath });
    }
  }

  async openImportedFileSelector(inputName: string) {
    const dialogRef = this.dialog.open(ImportedFileSelectionDialog, {
      width: '600px',
      disableClose: false
    });

    dialogRef.afterClosed().subscribe(async (result: { filePath?: string; columns?: string[] } | undefined) => {
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
      await this.log(`[DynamicForm] Setting columns for ${input.name} from imported file headers`);
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
      const headers = await this.fileHandler.getFileHeaders(filePath);
      if (headers && headers.length > 0) {
        await this.log(`[DynamicForm] Loaded ${headers.length} columns for ${inputName}: ${headers.join(', ')}`);
        this.columnOptions.set(inputName, headers);
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.logError(`Error loading columns: ${errorMsg}`);
      this.notification.showError(`Failed to load columns: ${errorMsg}`);
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

      await this.log(`[DynamicForm] Loading options from: ${fullPath}`);
      const content = await this.fileHandler.readFile(fullPath);
      if (!content) return;

      const options = content
        .split('\n')
        .map(line => line.trim())
        .filter(line => line.length > 0);

      this.selectOptions.set(inputName, options);
      await this.log(`[DynamicForm] Loaded ${options.length} options for ${inputName}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.logError(`[DynamicForm] Error loading options from file: ${errorMsg}`);
      this.notification.showError(`Failed to load options: ${errorMsg}`);
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

      await this.log(`[DynamicForm] Loading groups from: ${fullPath}`);
      const content = await this.fileHandler.readFile(fullPath);
      if (!content) return;

      const groups: FieldGroup[] = JSON.parse(content);
      this.groupedOptions.set(inputName, groups);
      await this.log(`[DynamicForm] Loaded ${groups.length} groups for ${inputName}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.logError(`[DynamicForm] Error loading groups from file: ${errorMsg}`);
      this.notification.showError(`Failed to load grouped options: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  getSelectOptions(input: PluginInputV2): FieldOption[] {
    let rawOptions: SelectOption[];

    if (input.optionsFromFile) {
      rawOptions = this.selectOptions.get(input.name) || [];
    } else {
      rawOptions = input.options || [];
    }

    return this.normalizeOptions(rawOptions);
  }

  private normalizeOptions(options: SelectOption[]): FieldOption[] {
    return options.map(opt => {
      if (typeof opt === 'string') {
        return { value: opt, label: opt };
      }
      return opt;
    });
  }

  getGroupedOptions(inputName: string): FieldGroup[] {
    return this.groupedOptions.get(inputName) || [];
  }

  getInputsByType(type: string): PluginInputV2[] {
    return this.plugin.definition.inputs.filter(i => i.type === type);
  }

  isInputVisible(input: PluginInputV2): boolean {
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
      this.notification.showError('Please fix validation errors before submitting');
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
      this.notification.showError('No example data available for this plugin');
      return;
    }

    if (!this.exampleFilePathResolver) {
      this.notification.showError('Example file path resolver not configured');
      return;
    }

    try {
      this.loading.set(true);
      const valuesToSet: Record<string, unknown> = {};

      for (const [key, value] of Object.entries(example.values)) {
        if (key.endsWith('_source')) {
          const targetField = key.replace('_source', '');
          const input = this.plugin.definition.inputs.find(i => i.name === targetField);

          if (input && input.type === 'column-selector') {
            const filePath = await this.exampleFilePathResolver.getPluginExampleFilePath(
              this.plugin.definition.plugin.id,
              value as string
            );
            await this.loadColumns(targetField, filePath);

            if (!example.values[targetField]) {
              const content = await this.fileHandler.readFile(filePath);
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
            const filePath = await this.exampleFilePathResolver.getPluginExampleFilePath(
              this.plugin.definition.plugin.id,
              value as string
            );
            valuesToSet[key] = filePath;

            await this.loadColumnsForDependents(key, filePath);
          } else {
            valuesToSet[key] = value;
          }
        }
      }

      this.form.patchValue(valuesToSet);
      this.notification.showSuccess('Example data loaded successfully');
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.logError(`Error loading example: ${errorMsg}`);
      this.notification.showError(`Failed to load example data: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadFromJobParameters(parameters: Record<string, unknown>) {
    try {
      this.loading.set(true);
      await this.log(`[DynamicForm] Loading parameters from job: ${JSON.stringify(parameters)}`);
      const valuesToSet: Record<string, unknown> = {};

      for (const [key, value] of Object.entries(parameters)) {
        const input = this.plugin.definition.inputs.find(i => i.name === key);

        if (input) {
          await this.log(`[DynamicForm] Processing parameter ${key}: type=${input.type}, value=${value} (${typeof value})`);

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
            await this.log(`[DynamicForm] Converted boolean parameter ${key}: ${value} (${typeof value}) -> ${boolValue}`);
          } else {
            valuesToSet[key] = value;
          }
        } else {
          await this.log(`[DynamicForm] Skipping parameter ${key}: no matching input found`);
        }
      }

      await this.log(`[DynamicForm] Values to set: ${JSON.stringify(valuesToSet)}`);
      this.form.patchValue(valuesToSet);
      this.notification.showSuccess('Job parameters loaded successfully');
      await this.log(`[DynamicForm] Successfully loaded job parameters`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      await this.logError(`[DynamicForm] Error loading job parameters: ${errorMsg}`);
      this.notification.showError(`Failed to load job parameters: ${errorMsg}`);
    } finally {
      this.loading.set(false);
    }
  }

  private getFormValue(): Record<string, unknown> {
    const value: Record<string, unknown> = {};

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

    if (event.shiftKey && this.lastSelectedIndex.has(fieldName)) {
      const currentValue: string[] = Array.isArray(control.value) ? [...control.value] : [];
      const lastIndex = this.lastSelectedIndex.get(fieldName)!;
      const start = Math.min(lastIndex, currentIndex);
      const end = Math.max(lastIndex, currentIndex);

      const rangeValues = allOptions.slice(start, end + 1);
      const newValue = [...new Set([...currentValue, ...rangeValues])];
      control.setValue(newValue);

      event.preventDefault();
      event.stopPropagation();
    }

    this.lastSelectedIndex.set(fieldName, currentIndex);
  }

  isAnnotationFileInput(input: PluginInputV2): boolean {
    if (input.type !== 'file') return false;
    if (input.disableAnnotationManagement === true) return false;
    const name = input.name.toLowerCase();
    const label = input.label?.toLowerCase() || '';
    return name.includes('annotation') ||
           name.includes('metadata') ||
           label.includes('annotation') ||
           label.includes('metadata') ||
           label.includes('sample annotation');
  }

  hasTableColumns(input: PluginInputV2): boolean {
    return input.type === 'file' && !!input.tableColumns && input.tableColumns.length > 0;
  }

  async openAnnotationManager(inputName: string) {
    const sampleNames = this.getSampleNamesForAnnotation();
    if (!sampleNames || sampleNames.length === 0) {
      this.notification.showWarning('Please select sample columns first to create an annotation file');
      return;
    }

    const currentFilePath = this.form.get(inputName)?.value;
    let existingAnnotation: Record<string, string>[] | undefined;

    if (currentFilePath) {
      try {
        const content = await this.fileHandler.readFile(currentFilePath);
        existingAnnotation = this.parseAnnotationFile(content, sampleNames);
      } catch (error) {
        await this.logError(`Error reading annotation file: ${error}`);
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
          this.notification.showSuccess('Annotation file saved successfully');
        } catch (error) {
          const errorMsg = error instanceof Error ? error.message : String(error);
          await this.logError(`Error saving annotation file: ${errorMsg}`);
          this.notification.showError(`Failed to save annotation file: ${errorMsg}`);
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

  private parseAnnotationFile(content: string, sampleNames: string[]): Record<string, string>[] {
    const lines = content.trim().split('\n');
    if (lines.length < 2) {
      return [];
    }

    const headers = lines[0].split('\t').map(h => h.trim());
    const sampleIdx = headers.findIndex(h => h.toLowerCase().includes('sample'));
    const conditionIdx = headers.findIndex(h => h.toLowerCase().includes('condition') || h.toLowerCase().includes('group'));
    const bioreplicateIdx = headers.findIndex(h => h.toLowerCase().includes('bioreplicate') || h.toLowerCase().includes('replicate'));
    const batchIdx = headers.findIndex(h => h.toLowerCase().includes('batch'));
    const colorIdx = headers.findIndex(h => h.toLowerCase().includes('color'));

    if (sampleIdx === -1) {
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

    return annotations;
  }

  private async saveAnnotationFile(annotations: Record<string, string>[]): Promise<string> {
    const headers = ['Sample', 'Condition', 'BioReplicate', 'Batch', 'Color'];
    const rows = annotations.map(a =>
      [a['Sample'] || '', a['Condition'] || '', a['BioReplicate'] || '', a['Batch'] || '', a['Color'] || ''].join('\t')
    );
    const content = [headers.join('\t'), ...rows].join('\n');

    const timestamp = Date.now();
    const filename = `annotation_${timestamp}.txt`;
    const filePath = await this.fileHandler.saveTempFile(filename, content);

    return filePath;
  }

  async openTableEditor(inputName: string) {
    const input = this.plugin.definition.inputs.find(i => i.name === inputName);
    if (!input || !input.tableColumns) return;

    const currentFilePath = this.form.get(inputName)?.value;
    let existingData: Record<string, unknown>[] | undefined;

    if (currentFilePath) {
      try {
        const content = await this.fileHandler.readFile(currentFilePath);
        existingData = this.parseTableFile(content, input.tableColumns);
      } catch (error) {
        await this.logError(`Error reading table file: ${error}`);
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
          this.notification.showSuccess('Table file saved successfully');
        } catch (error) {
          const errorMsg = error instanceof Error ? error.message : String(error);
          await this.logError(`Error saving table file: ${errorMsg}`);
          this.notification.showError(`Failed to save table file: ${errorMsg}`);
        }
      }
    });
  }

  private parseTableFile(content: string, columns: TableColumn[]): Record<string, unknown>[] {
    const lines = content.trim().split('\n');
    if (lines.length < 2) return [];

    const headers = lines[0].split('\t').map(h => h.trim());
    const data: Record<string, unknown>[] = [];

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line) continue;

      const values = lines[i].split('\t');
      const row: Record<string, unknown> = {};
      columns.forEach((col) => {
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

  private async saveTableFile(data: Record<string, unknown>[], columns: TableColumn[], inputName: string): Promise<string> {
    const headers = columns.map(c => c.name);

    const filteredData = data.filter(row => {
      return columns.some(col => {
        const value = row[col.name];
        return value !== null && value !== undefined && value !== '';
      });
    });

    const rows = filteredData.map(row =>
      columns.map(col => String(row[col.name] || '')).join('\t')
    );
    const content = [headers.join('\t'), ...rows].join('\n');

    const timestamp = Date.now();
    const filename = `${inputName}_${timestamp}.txt`;
    const filePath = await this.fileHandler.saveTempFile(filename, content);

    return filePath;
  }
}
