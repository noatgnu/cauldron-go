import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

export interface InstallPluginDialogData {
  repoURL?: string;
}

@Component({
  selector: 'app-install-plugin-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatInputModule,
    MatFormFieldModule,
    MatIconModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './install-plugin-dialog.html',
  styleUrl: './install-plugin-dialog.scss'
})
export class InstallPluginDialog {
  form: FormGroup;
  installing = false;
  error = '';

  urlPattern = /^(https?:\/\/|git@)[\w.-]+[:/][\w.-]+\/[\w.-]+(\.git)?$/;

  constructor(
    public dialogRef: MatDialogRef<InstallPluginDialog>,
    @Inject(MAT_DIALOG_DATA) public data: InstallPluginDialogData,
    private fb: FormBuilder
  ) {
    this.form = this.fb.group({
      repoURL: [data.repoURL || '', [Validators.required, Validators.pattern(this.urlPattern)]],
      commitHash: ['']
    });
  }

  getErrorMessage(): string {
    const repoURLControl = this.form.get('repoURL');
    if (repoURLControl?.hasError('required')) {
      return 'Repository URL is required';
    }
    if (repoURLControl?.hasError('pattern')) {
      return 'Invalid repository URL format. Expected: https://github.com/user/repo';
    }
    return '';
  }

  install() {
    if (this.form.valid) {
      this.dialogRef.close({
        repoURL: this.form.value.repoURL,
        commitHash: this.form.value.commitHash
      });
    }
  }

  cancel() {
    this.dialogRef.close();
  }
}
