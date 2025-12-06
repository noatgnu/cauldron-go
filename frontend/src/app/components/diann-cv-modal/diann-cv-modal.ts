import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from "@angular/forms";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {Wails} from "../../core/services/wails";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";

export interface DiannCvConfig {
  pr_matrix_file: string | null;
  pg_matrix_file: string | null;
  log_file: string | null;
  sample_annotations: string | null;
  annotation_file: string | null;
  samples: string | null;
}

@Component({
  selector: 'app-diann-cv-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule
  ],
  templateUrl: './diann-cv-modal.html',
  styleUrl: './diann-cv-modal.scss',
})
export class DiannCvModal {
  form!: FormGroup;

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<DiannCvModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      pr_matrix_file: new FormControl<string | null>(null),
      pg_matrix_file: new FormControl<string | null>(null),
      log_file: new FormControl<string | null>(null),
      sample_annotations: new FormControl<string | null>(null),
      annotation_file: new FormControl<string | null>(null),
      samples: new FormControl<string | null>(null),
    });
  }

  async browse(fileType: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        this.form.controls[fileType].setValue(path);
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(e: string, control: string) {
    this.form.controls[control].setValue(e);
  }

  close() {
    this.dialogRef.close();
  }

  submit() {
    this.dialogRef.close(this.form.value);
  }
}
