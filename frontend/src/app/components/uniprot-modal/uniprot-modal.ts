import {Component, Inject, Input} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatSelectModule} from "@angular/material/select";
import {getUniprotFromFields, uniprotColumns, uniprotSections} from "uniprotparserjs";

export interface UniprotModalData {
  columns: string[];
  fieldParameters?: string;
}

export interface UniprotConfig {
  from: string;
  to: string;
  column: string;
  selectedUniProtColumns: string[];
}

@Component({
  selector: 'app-uniprot-modal',
  imports: [
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatSelectModule
  ],
  templateUrl: './uniprot-modal.html',
  styleUrl: './uniprot-modal.scss',
})
export class UniprotModal {
  uniprotFromFields: {groupName: string, items: any[]}[] = [];
  columns: string[] = [];
  dataMap: {[key: string]: {label: string, fieldId: string, section: string}[]} = {};
  columnFormMap: {[key: string]: FormGroup<{columns: FormControl<string[] | null>}>} = {};
  sections = uniprotSections;

  uniprotCols = uniprotColumns;
  uniprotSectionMap: any = {};
  uniprotSections = uniprotSections;

  form!: FormGroup;

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<UniprotModal>,
    @Inject(MAT_DIALOG_DATA) public data: UniprotModalData
  ) {
    this.form = this.fb.group({
      from: new FormControl<string>('UniProtKB_AC-ID', Validators.required),
      to: new FormControl<string>('UniProtKB', Validators.required),
      column: new FormControl<string>('', Validators.required),
      selectedUniProtColumns: new FormControl<string[]>([], Validators.required),
    });

    if (data) {
      this.columns = data.columns || [];
      if (data.fieldParameters) {
        this.initializeFieldParameters(data.fieldParameters);
      }
    }

    for (const section of this.uniprotCols) {
      if (!this.uniprotSectionMap[section.section]) {
        this.uniprotSectionMap[section.section] = [];
      }
      this.uniprotSectionMap[section.section].push(section);
    }

    for (const section of this.uniprotSections) {
      this.columnFormMap[section] = this.fb.group({
        columns: new FormControl<string[]>([]),
      });
    }

    getUniprotFromFields().then((res: any) => {
      this.uniprotFromFields = res;
    });
  }

  initializeFieldParameters(fieldParameters: string) {
    const fields = fieldParameters.split(',');
    for (const col of uniprotColumns) {
      if (!this.dataMap[col.section]) {
        this.dataMap[col.section] = [];
      }
      this.dataMap[col.section].push(col);
      if (fields.includes(col.fieldId)) {
        const currentValues = this.columnFormMap[col.section].controls['columns'].value || [];
        this.columnFormMap[col.section].controls['columns'].setValue([...currentValues, col.fieldId]);
      }
    }
  }

  cancel() {
    this.dialogRef.close();
  }

  submit() {
    let selectedColumns: string[] = [];
    for (const s of uniprotSections) {
      if (this.columnFormMap[s]) {
        if (this.columnFormMap[s].value.columns) {
          for (const fieldId of this.columnFormMap[s].value.columns) {
            selectedColumns.push(fieldId);
          }
        }
      }
    }
    this.form.controls['selectedUniProtColumns'].setValue(selectedColumns);
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }
}
