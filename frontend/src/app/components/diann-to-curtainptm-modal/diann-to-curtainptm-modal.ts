import {Component, Inject} from '@angular/core';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from "@angular/forms";
import {Wails} from "../../core/services/wails";
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from "@angular/material/dialog";
import {ImportedFileSelection} from "../imported-file-selection/imported-file-selection";
import {MatButtonModule} from "@angular/material/button";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";

export interface DiannToCurtainPtmConfig {
  pr_file_path: string;
  report_file_path: string;
  modification_of_interests: string;
}

@Component({
  selector: 'app-diann-to-curtainptm-modal',
  imports: [
    ReactiveFormsModule,
    ImportedFileSelection,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule
  ],
  templateUrl: './diann-to-curtainptm-modal.html',
  styleUrl: './diann-to-curtainptm-modal.scss'
})
export class DiannToCurtainptmModal {
  form!: FormGroup;
  columns: string[] = [];

  constructor(
    private fb: FormBuilder,
    public dialogRef: MatDialogRef<DiannToCurtainptmModal>,
    private wails: Wails,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {
    this.form = this.fb.group({
      pr_file_path: new FormControl(null, Validators.required),
      report_file_path: new FormControl(null, Validators.required),
      modification_of_interests: new FormControl("UniMod:21", Validators.required),
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
        const preview = await this.wails.parseDataFile(path, 1);
        if (preview && preview.headers) {
          this.columns = preview.headers;
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  updateFormWithSelected(e: string, control: string) {
    this.form.controls[control].setValue(e);
  }
}
