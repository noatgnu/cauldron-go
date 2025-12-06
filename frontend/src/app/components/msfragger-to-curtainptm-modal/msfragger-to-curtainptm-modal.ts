import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";
import {MatSelectModule} from "@angular/material/select";

export interface MsfraggerToCurtainPtmConfig {
  file_path: string;
  index_col: string;
  peptide_col: string;
  fasta_file: string | null;
}

@Component({
  selector: 'app-msfragger-to-curtainptm-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule
  ],
  templateUrl: './msfragger-to-curtainptm-modal.html',
  styleUrl: './msfragger-to-curtainptm-modal.scss',
})
export class MsfraggerToCurtainptmModal {
  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<MsfraggerToCurtainptmModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      file_path: new FormControl(null, Validators.required),
      index_col: new FormControl(null, Validators.required),
      peptide_col: new FormControl(null, Validators.required),
      fasta_file: new FormControl(null),
    });
  }

  close() {
    this.dialogRef.close();
  }

  submit() {
    if (this.form.valid) {
      this.dialogRef.close(this.form.value);
    }
  }

  async openFile(formPath: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[formPath].setValue(path);
        if (formPath === 'file_path') {
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

  updateFormWithSelected(e: string, control: string) {
    this.form.controls[control].setValue(e);
  }

  updateColumns(cols: string[]) {
    this.columns = cols;
  }
}
