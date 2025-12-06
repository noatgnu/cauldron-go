import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";

export interface CoverageMapConfig {
  fasta_file: string | null;
  file_path: string;
  sequence_column: string;
  index_column: string;
  uniprot_acc_column: string;
  value_columns: string[];
}

@Component({
  selector: 'app-coverage-map-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './coverage-map-modal.html',
  styleUrl: './coverage-map-modal.scss',
})
export class CoverageMapModal {
  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<CoverageMapModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      fasta_file: new FormControl<string | null>(null),
      file_path: new FormControl<string | null>(null, Validators.required),
      sequence_column: new FormControl<string | null>(null, Validators.required),
      index_column: new FormControl<string | null>(null, Validators.required),
      uniprot_acc_column: new FormControl<string | null>(null, Validators.required),
      value_columns: new FormControl<string[]>([], Validators.required),
    });
  }

  async openFile(fileType: 'fasta_file' | 'file_path') {
    try {
      const path = await this.wails.openDataFileDialog();

      if (path) {
        this.form.controls[fileType].setValue(path);
        if (fileType === 'file_path') {
          const preview = await this.wails.parseDataFile(path, 1);
          if (preview && preview.headers) {
            this.columns = preview.headers;
          }
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(e: string, formControl: string) {
    this.form.controls[formControl].setValue(e);
  }

  updateColumns(cols: string[]) {
    this.columns = cols;
  }

  close() {
    this.dialogRef.close();
  }

  submit() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }
}
