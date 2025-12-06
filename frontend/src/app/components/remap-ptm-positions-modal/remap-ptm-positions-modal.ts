import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";

export interface RemapPtmConfig {
  fasta_file: string | null;
  file_path: string;
  peptide_column: string;
  position_in_peptide_column: string;
  uniprot_acc_column: string;
}

@Component({
  selector: 'app-remap-ptm-positions-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './remap-ptm-positions-modal.html',
  styleUrl: './remap-ptm-positions-modal.scss',
})
export class RemapPtmPositionsModal {
  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<RemapPtmPositionsModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      fasta_file: new FormControl<string | null>(null),
      file_path: new FormControl<string | null>(null, Validators.required),
      peptide_column: new FormControl<string | null>(null, Validators.required),
      position_in_peptide_column: new FormControl<string | null>(null, Validators.required),
      uniprot_acc_column: new FormControl<string | null>(null, Validators.required),
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
