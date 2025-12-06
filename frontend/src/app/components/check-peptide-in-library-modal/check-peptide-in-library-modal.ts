import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";

export interface CheckPeptideInLibraryConfig {
  fasta_file: string | null;
  miss_cleavage: number;
  min_length: number;
  file_path: string | null;
  peptide_column: string | null;
}

@Component({
  selector: 'app-check-peptide-in-library-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './check-peptide-in-library-modal.html',
  styleUrl: './check-peptide-in-library-modal.scss',
})
export class CheckPeptideInLibraryModal {
  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<CheckPeptideInLibraryModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      fasta_file: new FormControl<string | null>(null, Validators.required),
      miss_cleavage: new FormControl<number>(2, Validators.required),
      min_length: new FormControl<number>(5, Validators.required),
      file_path: new FormControl<string | null>(null, Validators.required),
      peptide_column: new FormControl<string | null>(null, Validators.required),
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
