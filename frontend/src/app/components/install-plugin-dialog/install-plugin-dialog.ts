import { Component, Inject, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

export interface InstallPluginDialogData {
  repoURL?: string;
}

export interface InstallPluginResult {
  repoURL: string;
  commitHash: string;
  sshKeyPath?: string;
  passphrase?: string;
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
    MatProgressSpinnerModule,
    MatExpansionModule,
    MatTooltipModule
  ],
  templateUrl: './install-plugin-dialog.html',
  styleUrl: './install-plugin-dialog.scss'
})
export class InstallPluginDialog {
  private wails = inject(Wails);
  private notification = inject(NotificationService);

  form: FormGroup;
  installing = false;
  error = '';
  showPassphrase = signal(false);
  validatingKey = signal(false);

  urlPattern = /^(https?:\/\/|git@)[\w.-]+[:/][\w.-]+\/[\w.-]+(\.git)?$/;

  constructor(
    public dialogRef: MatDialogRef<InstallPluginDialog>,
    @Inject(MAT_DIALOG_DATA) public data: InstallPluginDialogData,
    private fb: FormBuilder
  ) {
    this.form = this.fb.group({
      repoURL: [data.repoURL || '', [Validators.required, Validators.pattern(this.urlPattern)]],
      commitHash: [''],
      sshKeyPath: [''],
      passphrase: ['']
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

  async browseSSHKey() {
    try {
      const path = await this.wails.openFile('Select SSH Private Key');
      if (path) {
        this.form.patchValue({ sshKeyPath: path });
      }
    } catch (err: any) {
      this.notification.showError('Failed to select SSH key file');
    }
  }

  async validateKey() {
    const keyPath = this.form.value.sshKeyPath;
    const passphrase = this.form.value.passphrase;

    if (!keyPath) {
      this.notification.showWarning('Please select an SSH key file first');
      return;
    }

    this.validatingKey.set(true);
    try {
      await this.wails.validateSSHKey(keyPath, passphrase || '');
      this.notification.showSuccess('SSH key is valid');
    } catch (err: any) {
      this.notification.showError(`Invalid SSH key: ${err}`);
    } finally {
      this.validatingKey.set(false);
    }
  }

  togglePassphraseVisibility() {
    this.showPassphrase.update(v => !v);
  }

  install() {
    if (this.form.valid) {
      const result: InstallPluginResult = {
        repoURL: this.form.value.repoURL,
        commitHash: this.form.value.commitHash,
        sshKeyPath: this.form.value.sshKeyPath || undefined,
        passphrase: this.form.value.passphrase || undefined
      };
      this.dialogRef.close(result);
    }
  }

  cancel() {
    this.dialogRef.close();
  }
}
