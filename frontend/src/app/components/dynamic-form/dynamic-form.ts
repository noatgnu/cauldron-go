import { Component, Input, Output, EventEmitter, OnInit, signal, computed } from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { models } from '../../../wailsjs/go/models';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

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
    MatProgressSpinnerModule
  ],
  templateUrl: './dynamic-form.html',
  styleUrl: './dynamic-form.scss'
})
export class DynamicFormComponent implements OnInit {
  @Input() plugin!: models.PluginV2;
  @Input() disabled = false;
  @Output() formSubmit = new EventEmitter<Record<string, any>>();
  @Output() formChange = new EventEmitter<Record<string, any>>();

  form!: FormGroup;
  columnOptions = new Map<string, string[]>();
  loading = signal(false);
  formValues = signal<Record<string, any>>({});
  validationErrors = signal<string[]>([]);
  lastSelectedIndex = new Map<string, number>();

  constructor(
    private fb: FormBuilder,
    private wails: Wails,
    private notificationService: NotificationService
  ) {}

  ngOnInit() {
    this.buildForm();
    this.form.valueChanges.subscribe((values) => {
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

      const headers = lines[0].split('\t').map(h => h.trim());
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
            const [category, filename] = (value as string).split('/');
            const filePath = await this.wails.getExampleFilePath(category, filename);
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
            const [category, filename] = (value as string).split('/');
            const filePath = await this.wails.getExampleFilePath(category, filename);
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
}
