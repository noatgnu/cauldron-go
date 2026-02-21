import { Component, Inject, Optional } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatDividerModule } from '@angular/material/divider';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { NotificationHandler } from '../../interfaces/notification.interface';
import { NOTIFICATION_HANDLER } from '../../tokens/injection-tokens';
import { ColorPaletteProvider, COLOR_PALETTE_PROVIDER, DEFAULT_COLOR_PALETTE } from '../../interfaces/color-palette.interface';
import { Annotation, SampleAnnotationData } from '../../models/plugin.models';

interface AnnotationRow extends Annotation {
  selected?: boolean;
}

interface RegexRule {
  pattern: string;
  condition: string;
  bioreplicate: string;
  batch: string;
}

@Component({
  selector: 'cld-sample-annotation',
  standalone: true,
  imports: [
    ReactiveFormsModule,
    FormsModule,
    MatDialogModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatIconModule,
    MatTableModule,
    MatCheckboxModule,
    MatExpansionModule,
    MatDividerModule,
    MatTooltipModule,
    MatMenuModule
  ],
  templateUrl: './sample-annotation.html',
  styleUrl: './sample-annotation.scss'
})
export class SampleAnnotation {
  _annotation: AnnotationRow[] = [];
  mode: 'edit' | 'create' = 'edit';
  displayedColumns: string[] = ['Sample', 'Condition', 'BioReplicate', 'Color', 'Batch'];

  private get defaultColorList(): string[] {
    if (this.colorPaletteProvider) {
      return this.colorPaletteProvider.getColorPalette();
    }
    return DEFAULT_COLOR_PALETTE;
  }

  regexRules: RegexRule[] = [];
  batchCondition = '';
  batchBioReplicate = '';
  batchBatch = '';
  selectAllCheckbox = false;

  constructor(
    public dialogRef: MatDialogRef<SampleAnnotation>,
    @Inject(MAT_DIALOG_DATA) public data: SampleAnnotationData,
    @Inject(NOTIFICATION_HANDLER) private notification: NotificationHandler,
    @Optional() @Inject(COLOR_PALETTE_PROVIDER) private colorPaletteProvider: ColorPaletteProvider | null
  ) {
    try {
      this.mode = data.mode;
      if (data.samples) {
        this._annotation = data.samples.map(s => ({
          sample: s,
          condition: '',
          bioreplicate: '',
          batch: '',
          color: this.defaultColorList[0],
          selected: false
        }));
      } else if (data.annotation) {
        const annotationArray = Array.isArray(data.annotation) ? data.annotation : [];
        this._annotation = annotationArray.map(a => ({
          sample: a.sample || '',
          condition: a.condition || '',
          bioreplicate: a.bioreplicate || '',
          batch: a.batch || '',
          color: a.color || '',
          selected: false
        }));
      } else {
        this._annotation = [];
      }
    } catch (error) {
      this._annotation = [];
    }
  }

  onConditionChange(changedRow: AnnotationRow) {
    if (!changedRow.condition) {
      changedRow.color = '';
      return;
    }

    const existingRowWithSameCondition = this._annotation.find(row => row !== changedRow && row.condition === changedRow.condition && row.color);
    if (existingRowWithSameCondition) {
      changedRow.color = existingRowWithSameCondition.color;
    } else {
      const usedColors = new Set(this._annotation.map(a => a.color).filter(c => c));
      let newColor = '';
      for (const color of this.defaultColorList) {
        if (!usedColors.has(color)) {
          newColor = color;
          break;
        }
      }
      changedRow.color = newColor || this.defaultColorList[0];
    }
  }

  get annotation(): AnnotationRow[] {
    return this._annotation;
  }

  save() {
    const annotationsToSave: Annotation[] = this._annotation.map(({ selected, ...rest }) => rest);
    this.dialogRef.close(annotationsToSave);
  }

  close() {
    this.dialogRef.close();
  }

  addSample() {
    this._annotation.push({ sample: '', condition: '', bioreplicate: '', batch: '', color: '', selected: false });
  }

  parseFromClipboard(column: 'sample' | 'condition' | 'bioreplicate' | 'batch' | 'color') {
    navigator.clipboard.readText().then(text => {
      text = text.trim();
      let lines = text.split(/\r?\n/).filter(l => l.length > 0);
      if (text.includes('\t')) {
        lines = text.split(/\t/);
      }

      if (this._annotation.length === lines.length) {
        for (let i = 0; i < lines.length; i++) {
          this._annotation[i][column] = lines[i];
          if (column === 'condition') {
            this.onConditionChange(this._annotation[i]);
          }
        }
      } else {
        for (let i = 0; i < lines.length; i++) {
          if (this._annotation[i]) {
            this._annotation[i][column] = lines[i];
          } else {
            const newRow: AnnotationRow = { sample: '', condition: '', bioreplicate: '', batch: '', color: '', selected: false };
            newRow[column] = lines[i];
            this._annotation.push(newRow);
          }
          if (column === 'condition') {
            this.onConditionChange(this._annotation[i]);
          }
        }
      }
      this.notification.showSuccess(`Parsed ${lines.length} ${column.toLowerCase()} value(s)`);
    });
  }

  addRegexRule() {
    this.regexRules.push({ pattern: '', condition: '', bioreplicate: '', batch: '' });
  }

  removeRegexRule(index: number) {
    this.regexRules.splice(index, 1);
  }

  applyRegexRules() {
    let matchCount = 0;
    for (const rule of this.regexRules) {
      if (!rule.pattern || !rule.condition) continue;
      try {
        const regex = new RegExp(rule.pattern, 'i');
        for (const annotation of this._annotation) {
          if (regex.test(annotation.sample)) {
            annotation.condition = rule.condition;
            this.onConditionChange(annotation);
            if (rule.bioreplicate) {
              annotation.bioreplicate = rule.bioreplicate;
            }
            if (rule.batch) {
              annotation.batch = rule.batch;
            }
            matchCount++;
          }
        }
      } catch (err) {
        this.notification.showError(`Invalid regex pattern: ${rule.pattern}`);
      }
    }
    this.notification.showSuccess(`Applied rules to ${matchCount} sample(s)`);
  }

  selectAll() {
    this._annotation.forEach(a => a.selected = true);
    this.selectAllCheckbox = true;
  }

  deselectAll() {
    this._annotation.forEach(a => a.selected = false);
    this.selectAllCheckbox = false;
  }

  toggleSelectAll() {
    if (this.selectAllCheckbox) {
      this.selectAll();
    } else {
      this.deselectAll();
    }
  }

  isSelected(index: number): boolean {
    return this._annotation[index]?.selected || false;
  }

  applyBatchAssignment() {
    const selectedCount = this._annotation.filter(a => a.selected).length;
    if (selectedCount === 0) {
      this.notification.showWarning('No samples selected');
      return;
    }
    if (!this.batchCondition) {
      this.notification.showWarning('Please enter a condition');
      return;
    }
    for (const annotation of this._annotation) {
      if (annotation.selected) {
        annotation.condition = this.batchCondition;
        this.onConditionChange(annotation);
        if (this.batchBioReplicate) {
          annotation.bioreplicate = this.batchBioReplicate;
        }
        if (this.batchBatch) {
          annotation.batch = this.batchBatch;
        }
      }
    }
    this.notification.showSuccess(`Applied condition "${this.batchCondition}" to ${selectedCount} sample(s)`);
    this.deselectAll();
    this.batchCondition = '';
    this.batchBioReplicate = '';
    this.batchBatch = '';
  }

  replaceColorForCondition(condition: string) {
    const firstSample = this._annotation.find(a => a.condition === condition);
    const currentColor = firstSample?.color || this.defaultColorList[0];

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = currentColor;
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      let count = 0;
      for (const annotation of this._annotation) {
        if (annotation.condition === condition) {
          annotation.color = newColor;
          count++;
        }
      }
      this.notification.showSuccess(`Updated color for ${count} sample(s) with condition "${condition}"`);
    };
    colorInput.click();
  }

  replaceColorForBatch(batch: string) {
    const firstSample = this._annotation.find(a => a.batch === batch);
    const currentColor = firstSample?.color || this.defaultColorList[0];

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = currentColor;
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      let count = 0;
      for (const annotation of this._annotation) {
        if (annotation.batch === batch) {
          annotation.color = newColor;
          count++;
        }
      }
      this.notification.showSuccess(`Updated color for ${count} sample(s) in batch "${batch}"`);
    };
    colorInput.click();
  }

  selectByCondition(condition: string) {
    this.deselectAll();
    for (const annotation of this._annotation) {
      if (annotation.condition === condition) {
        annotation.selected = true;
      }
    }
    const count = this._annotation.filter(a => a.selected).length;
    this.notification.showSuccess(`Selected ${count} sample(s) with condition "${condition}"`);
  }

  selectByBatch(batch: string) {
    this.deselectAll();
    for (const annotation of this._annotation) {
      if (annotation.batch === batch) {
        annotation.selected = true;
      }
    }
    const count = this._annotation.filter(a => a.selected).length;
    this.notification.showSuccess(`Selected ${count} sample(s) in batch "${batch}"`);
  }

  getUniqueConditions(): string[] {
    return [...new Set(this._annotation.map(a => a.condition).filter((c): c is string => !!c))];
  }

  getUniqueBatches(): string[] {
    return [...new Set(this._annotation.map(a => a.batch).filter((b): b is string => !!b))];
  }

  assignColorToSelected() {
    const selected = this._annotation.filter(a => a.selected);
    if (selected.length === 0) {
      this.notification.showWarning('No samples selected');
      return;
    }

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = selected[0].color || this.defaultColorList[0];
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      for (const annotation of this._annotation) {
        if (annotation.selected) {
          annotation.color = newColor;
        }
      }
      this.notification.showSuccess(`Updated color for ${selected.length} selected sample(s)`);
    };
    colorInput.click();
  }

  clearAllConditions() {
    for (const annotation of this._annotation) {
      annotation.condition = '';
      annotation.color = '';
    }
    this.notification.showSuccess('Cleared all conditions');
  }

  clearAllBatches() {
    for (const annotation of this._annotation) {
      annotation.batch = '';
    }
    this.notification.showSuccess('Cleared all batches');
  }
}
