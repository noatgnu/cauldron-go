import {Component, Inject, Input} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {FormsModule, ReactiveFormsModule} from "@angular/forms";
import {DataFrame, IDataFrame} from "data-forge";
import {MatButtonModule} from "@angular/material/button";
import {MatInputModule} from "@angular/material/input";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatIconModule} from "@angular/material/icon";
import {MatTableModule} from "@angular/material/table";
import {MatCheckboxModule} from "@angular/material/checkbox";
import {MatExpansionModule} from "@angular/material/expansion";
import {MatDividerModule} from "@angular/material/divider";

export interface SampleAnnotationData {
  samples?: string[];
  annotation?: IDataFrame<number, {Sample: string, Condition: string, Batch: string}>;
  mode: 'edit' | 'create';
}

interface RegexRule {
  pattern: string;
  condition: string;
  batch: string;
}

@Component({
  selector: 'app-sample-annotation',
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
    MatDividerModule
  ],
  templateUrl: './sample-annotation.html',
  styleUrl: './sample-annotation.scss'
})
export class SampleAnnotation {
  _annotation: {Sample: string, Condition: string, Batch: string, selected?: boolean}[] = [];
  mode: 'edit' | 'create' = 'edit';
  displayedColumns: string[] = ['Sample', 'Condition', 'Batch'];

  regexRules: RegexRule[] = [];
  batchCondition: string = '';
  batchBatch: string = '';
  selectAllCheckbox: boolean = false;

  constructor(
    public dialogRef: MatDialogRef<SampleAnnotation>,
    @Inject(MAT_DIALOG_DATA) public data: SampleAnnotationData
  ) {
    this.mode = data.mode;

    if (data.samples) {
      const annotationData: any[] = [];
      for (const s of data.samples) {
        annotationData.push({Sample: s, Condition: '', Batch: '', selected: false});
      }
      this._annotation = annotationData;
    } else if (data.annotation) {
      this._annotation = data.annotation.toArray().map(a => ({...a, selected: false}));
    }
  }

  get annotation(): {Sample: string, Condition: string, Batch: string, selected?: boolean}[] {
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
    this._annotation.push({Sample: '', Condition: '', Batch: '', selected: false});
  }

  parseFromClipboard(column: 'Sample'|'Condition'|'Batch') {
    navigator.clipboard.readText().then(text => {
      text = text.trim();
      let lines = text.split(/\r?\n/).filter(l => l.length > 0);
      if (text.includes('\t')) {
        lines = text.split(/\t/);
      }

      if (this._annotation.length === lines.length) {
        for (let i = 0; i < lines.length; i++) {
          this._annotation[i][column] = lines[i];
        }
      } else {
        for (let i = 0; i < lines.length; i++) {
          if (this._annotation[i]) {
            this._annotation[i][column] = lines[i];
          } else {
            this._annotation.push({Sample: lines[i], Condition: '', Batch: '', selected: false});
          }
        }
      }
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
            if (rule.batch) {
              annotation.Batch = rule.batch;
            }
            matchCount++;
          }
        }
      } catch (err) {
        console.error(`Invalid regex pattern: ${rule.pattern}`, err);
        alert(`Invalid regex pattern: ${rule.pattern}`);
      }
    }

    alert(`Applied rules to ${matchCount} sample(s)`);
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
      alert('No samples selected');
      return;
    }

    if (!this.batchCondition) {
      alert('Please enter a condition');
      return;
    }

    for (const annotation of this._annotation) {
      if (annotation.selected) {
        annotation.Condition = this.batchCondition;
        if (this.batchBatch) {
          annotation.Batch = this.batchBatch;
        }
      }
    }

    alert(`Applied condition "${this.batchCondition}" to ${selectedCount} sample(s)`);
    this.deselectAll();
    this.batchCondition = '';
    this.batchBatch = '';
  }
}
