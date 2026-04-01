import { Component, Inject, OnInit, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Wails } from '../../core/services/wails';

export interface UninstallPluginDialogData {
  pluginId: string;
  pluginName: string;
  repositoryURL: string;
}

export interface UninstallPluginResult {
  confirmed: boolean;
  removeGitAuth: boolean;
  deleteJobHistory: boolean;
  deleteEnvironments: boolean;
}

@Component({
  selector: 'app-uninstall-plugin-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatCheckboxModule,
    MatProgressSpinnerModule
  ],
  templateUrl: './uninstall-plugin-dialog.html',
  styleUrl: './uninstall-plugin-dialog.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class UninstallPluginDialog implements OnInit {
  form: FormGroup;
  loading = signal(true);
  jobCount = signal(0);
  envCount = signal(0);

  constructor(
    public dialogRef: MatDialogRef<UninstallPluginDialog>,
    @Inject(MAT_DIALOG_DATA) public data: UninstallPluginDialogData,
    private fb: FormBuilder,
    private wails: Wails
  ) {
    this.form = this.fb.group({
      removeGitAuth: [false],
      deleteJobHistory: [false],
      deleteEnvironments: [false]
    });
  }

  async ngOnInit() {
    try {
      const [jobs, envs] = await Promise.all([
        this.wails.getPluginJobCount(this.data.pluginId),
        this.wails.getPluginEnvironmentCount(this.data.pluginId)
      ]);
      this.jobCount.set(jobs);
      this.envCount.set(envs);
    } catch (err) {
      console.error('Failed to get plugin information:', err);
    } finally {
      this.loading.set(false);
    }
  }

  confirm() {
    const result: UninstallPluginResult = {
      confirmed: true,
      removeGitAuth: this.form.value.removeGitAuth,
      deleteJobHistory: this.form.value.deleteJobHistory,
      deleteEnvironments: this.form.value.deleteEnvironments
    };
    this.dialogRef.close(result);
  }

  cancel() {
    this.dialogRef.close({ confirmed: false });
  }
}
