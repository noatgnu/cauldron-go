import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {FormsModule, ReactiveFormsModule} from "@angular/forms";
import {IDataFrame} from "data-forge";
import {MatButtonModule} from "@angular/material/button";
import {MatInputModule} from "@angular/material/input";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatIconModule} from "@angular/material/icon";
import {MatTableModule} from "@angular/material/table";
import {MatCheckboxModule} from "@angular/material/checkbox";
import {MatExpansionModule} from "@angular/material/expansion";
import {MatDividerModule} from "@angular/material/divider";
import {MatTooltipModule} from "@angular/material/tooltip";
import {MatMenuModule} from "@angular/material/menu";
import {NotificationService} from "../../core/services/notification.service";

export interface SampleAnnotationData {
  samples?: string[];
  annotation?: IDataFrame<number, {Sample: string, Condition: string, Batch: string, Color: string}>;
  mode: 'edit' | 'create';
}

interface RegexRule {
  pattern: string;
  condition: string;
  batch: string;
}

@Component({
  selector: 'app-sample-annotation',
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
  _annotation: {Sample: string, Condition: string, Batch: string, Color: string, selected?: boolean}[] = [];
  mode: 'edit' | 'create' = 'edit';
  displayedColumns: string[] = ['Sample', 'Condition', 'Color', 'Batch'];
  private readonly defaultColorList: string[] = [
    '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
    '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf'
  ];

  regexRules: RegexRule[] = [];
  batchCondition: string = '';
  batchBatch: string = '';
  selectAllCheckbox: boolean = false;

  constructor(
    public dialogRef: MatDialogRef<SampleAnnotation>,
    @Inject(MAT_DIALOG_DATA) public data: SampleAnnotationData,
    private notificationService: NotificationService
  ) {
    this.mode = data.mode;
    if (data.samples) {
      this._annotation = data.samples.map(s => ({Sample: s, Condition: '', Batch: '', Color: this.defaultColorList[0], selected: false}));
    } else if (data.annotation) {
      this._annotation = data.annotation.toArray().map(a => ({...a, selected: false, Color: a.Color || ''}));
    }
  }

  onConditionChange(changedRow: {Sample: string, Condition: string, Batch: string, Color: string, selected?: boolean}) {
    if (!changedRow.Condition) {
      changedRow.Color = '';
      return;
    }

    const existingRowWithSameCondition = this._annotation.find(row => row !== changedRow && row.Condition === changedRow.Condition && row.Color);
    if (existingRowWithSameCondition) {
      changedRow.Color = existingRowWithSameCondition.Color;
    } else {
      const usedColors = new Set(this._annotation.map(a => a.Color).filter(c => c));
      let newColor = '';
      for (const color of this.defaultColorList) {
        if (!usedColors.has(color)) {
          newColor = color;
          break;
        }
      }
      changedRow.Color = newColor || this.defaultColorList[0];
    }
  }

  get annotation(): {Sample: string, Condition: string, Batch: string, Color: string, selected?: boolean}[] {
    return this._annotation;
  }

  save() {
    const annotationsToSave = this._annotation.map(({selected, ...rest}) => rest);
    this.dialogRef.close(annotationsToSave);
  }

  close() {
    this.dialogRef.close();
  }

  addSample() {
    this._annotation.push({Sample: '', Condition: '', Batch: '', Color: '', selected: false});
  }

  parseFromClipboard(column: 'Sample'|'Condition'|'Batch'|'Color') {
    navigator.clipboard.readText().then(text => {
      text = text.trim();
      let lines = text.split(/\r?\n/).filter(l => l.length > 0);
      if (text.includes('\t')) {
        lines = text.split(/\t/);
      }

      if (this._annotation.length === lines.length) {
        for (let i = 0; i < lines.length; i++) {
          this._annotation[i][column] = lines[i];
          if (column === 'Condition') {
            this.onConditionChange(this._annotation[i]);
          }
        }
      } else {
        for (let i = 0; i < lines.length; i++) {
          if (this._annotation[i]) {
            this._annotation[i][column] = lines[i];
          } else {
            this._annotation.push({Sample: lines[i], Condition: '', Batch: '', Color: '', selected: false});
          }
           if (column === 'Condition') {
            this.onConditionChange(this._annotation[i]);
          }
        }
      }
      this.notificationService.showSuccess(`Parsed ${lines.length} ${column.toLowerCase()} value(s)`);
    });
  }

  addRegexRule() {
    this.regexRules.push({pattern: '', condition: '', batch: ''});
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
          if (regex.test(annotation.Sample)) {
            annotation.Condition = rule.condition;
            this.onConditionChange(annotation);
            if (rule.batch) {
              annotation.Batch = rule.batch;
            }
            matchCount++;
          }
        }
      } catch (err) {
        this.notificationService.showError(`Invalid regex pattern: ${rule.pattern}`);
      }
    }
    this.notificationService.showSuccess(`Applied rules to ${matchCount} sample(s)`);
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
      this.notificationService.showWarning('No samples selected');
      return;
    }
    if (!this.batchCondition) {
      this.notificationService.showWarning('Please enter a condition');
      return;
    }
    for (const annotation of this._annotation) {
      if (annotation.selected) {
        annotation.Condition = this.batchCondition;
        this.onConditionChange(annotation);
        if (this.batchBatch) {
          annotation.Batch = this.batchBatch;
        }
      }
    }
    this.notificationService.showSuccess(`Applied condition "${this.batchCondition}" to ${selectedCount} sample(s)`);
    this.deselectAll();
    this.batchCondition = '';
    this.batchBatch = '';
  }

  replaceColorForCondition(condition: string) {
    const firstSample = this._annotation.find(a => a.Condition === condition);
    const currentColor = firstSample?.Color || this.defaultColorList[0];

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = currentColor;
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      let count = 0;
      for (const annotation of this._annotation) {
        if (annotation.Condition === condition) {
          annotation.Color = newColor;
          count++;
        }
      }
      this.notificationService.showSuccess(`Updated color for ${count} sample(s) with condition "${condition}"`);
    };
    colorInput.click();
  }

  replaceColorForBatch(batch: string) {
    const firstSample = this._annotation.find(a => a.Batch === batch);
    const currentColor = firstSample?.Color || this.defaultColorList[0];

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = currentColor;
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      let count = 0;
      for (const annotation of this._annotation) {
        if (annotation.Batch === batch) {
          annotation.Color = newColor;
          count++;
        }
      }
      this.notificationService.showSuccess(`Updated color for ${count} sample(s) in batch "${batch}"`);
    };
    colorInput.click();
  }

  selectByCondition(condition: string) {
    this.deselectAll();
    for (const annotation of this._annotation) {
      if (annotation.Condition === condition) {
        annotation.selected = true;
      }
    }
    const count = this._annotation.filter(a => a.selected).length;
    this.notificationService.showSuccess(`Selected ${count} sample(s) with condition "${condition}"`);
  }

  selectByBatch(batch: string) {
    this.deselectAll();
    for (const annotation of this._annotation) {
      if (annotation.Batch === batch) {
        annotation.selected = true;
      }
    }
    const count = this._annotation.filter(a => a.selected).length;
    this.notificationService.showSuccess(`Selected ${count} sample(s) in batch "${batch}"`);
  }

  getUniqueConditions(): string[] {
    return [...new Set(this._annotation.map(a => a.Condition).filter(c => c))];
  }

  getUniqueBatches(): string[] {
    return [...new Set(this._annotation.map(a => a.Batch).filter(b => b))];
  }

  assignColorToSelected() {
    const selected = this._annotation.filter(a => a.selected);
    if (selected.length === 0) {
      this.notificationService.showWarning('No samples selected');
      return;
    }

    const colorInput = document.createElement('input');
    colorInput.type = 'color';
    colorInput.value = selected[0].Color || this.defaultColorList[0];
    colorInput.onchange = () => {
      const newColor = colorInput.value;
      for (const annotation of this._annotation) {
        if (annotation.selected) {
          annotation.Color = newColor;
        }
      }
      this.notificationService.showSuccess(`Updated color for ${selected.length} selected sample(s)`);
    };
    colorInput.click();
  }

  clearAllConditions() {
    for (const annotation of this._annotation) {
      annotation.Condition = '';
      annotation.Color = '';
    }
    this.notificationService.showSuccess('Cleared all conditions');
  }

  clearAllBatches() {
    for (const annotation of this._annotation) {
      annotation.Batch = '';
    }
    this.notificationService.showSuccess('Cleared all batches');
  }
}
