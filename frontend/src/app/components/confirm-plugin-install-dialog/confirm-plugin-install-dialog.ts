import { Component, Inject, inject, signal, computed, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatSelectModule } from '@angular/material/select';
import { Wails, PythonEnvironment, REnvironment } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';

export interface PluginInstallConfirmData {
  repo: string;
  ref?: string;
  registry?: string;
  name: string;
  id: string;
  version: string;
  author: string;
  description: string;
  category: string;
  requiresAuthentication?: boolean;
  runtimeEnvironments?: string[];
  hasPythonDeps?: boolean;
  hasRDeps?: boolean;
}

export interface PluginInstallConfirmResult {
  confirmed: boolean;
  sshKeyPath?: string;
  passphrase?: string;
  createVenv?: boolean;
  basePythonPath?: string;
  createRenv?: boolean;
  renvName?: string;
}

@Component({
  selector: 'app-confirm-plugin-install-dialog',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatDividerModule,
    MatExpansionModule,
    MatFormFieldModule,
    MatInputModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    MatCheckboxModule,
    MatSelectModule
  ],
  templateUrl: './confirm-plugin-install-dialog.html',
  styleUrl: './confirm-plugin-install-dialog.scss',
})
export class ConfirmPluginInstallDialog implements OnInit {
  private wails = inject(Wails);
  private notification = inject(NotificationService);
  private fb = inject(FormBuilder);

  form: FormGroup;
  showPassphrase = signal(false);
  validatingKey = signal(false);

  pythonEnvironments = signal<PythonEnvironment[]>([]);
  basePythonEnvironments = computed(() => this.pythonEnvironments().filter(env => !env.isVirtual));
  rEnvironments = signal<REnvironment[]>([]);
  loadingEnvironments = signal(true);

  constructor(
    public dialogRef: MatDialogRef<ConfirmPluginInstallDialog>,
    @Inject(MAT_DIALOG_DATA) public data: PluginInstallConfirmData
  ) {
    this.form = this.fb.group({
      sshKeyPath: [''],
      passphrase: [''],
      createVenv: [false],
      basePythonPath: [''],
      createRenv: [false],
      renvName: ['']
    });
  }

  async ngOnInit() {
    await this.wails.logToFile(`[ConfirmPluginInstallDialog] Dialog initialized with data: hasPythonDeps=${this.data.hasPythonDeps}, hasRDeps=${this.data.hasRDeps}`);
    await this.loadEnvironments();
  }

  async loadEnvironments() {
    this.loadingEnvironments.set(true);
    try {
      const [pythonEnvs, rEnvs] = await Promise.all([
        this.wails.detectPythonEnvironments(),
        this.wails.detectREnvironments()
      ]);
      this.pythonEnvironments.set(pythonEnvs || []);
      this.rEnvironments.set(rEnvs || []);

      const baseEnvs = this.basePythonEnvironments();
      await this.wails.logToFile(`[ConfirmPluginInstallDialog] Detected ${baseEnvs.length} base Python environments and ${rEnvs.length} R environments`);

      if (baseEnvs.length > 0) {
        this.form.patchValue({ basePythonPath: baseEnvs[0].path });
      }

      await this.wails.logToFile(`[ConfirmPluginInstallDialog] Checking Python deps - hasPythonDeps: ${this.data.hasPythonDeps}, baseEnvsAvailable: ${baseEnvs.length > 0}`);

      if (this.data.hasPythonDeps && baseEnvs.length > 0) {
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Auto-enabling Python venv`);
        this.form.patchValue({ createVenv: true });
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Form value after patch - createVenv: ${this.form.value.createVenv}`);
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Auto-enabled Python venv for plugin with Python dependencies`);
      } else if (this.data.hasPythonDeps && baseEnvs.length === 0) {
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Python deps detected but no base Python environments available`);
      }

      await this.wails.logToFile(`[ConfirmPluginInstallDialog] Checking R deps - hasRDeps: ${this.data.hasRDeps}, rEnvsAvailable: ${rEnvs.length > 0}`);

      if (this.data.hasRDeps && rEnvs.length > 0) {
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Auto-enabling R renv`);
        this.form.patchValue({
          createRenv: true,
          renvName: `renv-${this.data.id}`
        });
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Form value after R patch - createRenv: ${this.form.value.createRenv}`);
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] Auto-enabled R renv for plugin with R dependencies`);
      } else if (this.data.hasRDeps && rEnvs.length === 0) {
        await this.wails.logToFile(`[ConfirmPluginInstallDialog] R deps detected but no R environments available`);
      }

      this.form.get('createRenv')?.valueChanges.subscribe((checked: boolean) => {
        if (checked && !this.form.value.renvName) {
          this.form.patchValue({ renvName: `renv-${this.data.id}` });
        }
      });
    } catch (error) {
      await this.wails.logToFile(`[ConfirmPluginInstallDialog] Failed to load environments: ${error}`);
    } finally {
      this.loadingEnvironments.set(false);
    }
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

  confirm() {
    const result: PluginInstallConfirmResult = {
      confirmed: true,
      sshKeyPath: this.form.value.sshKeyPath || undefined,
      passphrase: this.form.value.passphrase || undefined,
      createVenv: this.form.value.createVenv || false,
      basePythonPath: this.form.value.createVenv ? this.form.value.basePythonPath : undefined,
      createRenv: this.form.value.createRenv || false,
      renvName: this.form.value.createRenv ? this.form.value.renvName : undefined
    };
    this.dialogRef.close(result);
  }

  getEnvTypeLabel(type: string): string {
    switch (type) {
      case 'system': return 'System';
      case 'venv': return 'Virtual Env';
      case 'conda': return 'Conda';
      case 'portable': return 'Portable';
      default: return type;
    }
  }

  cancel() {
    this.dialogRef.close({ confirmed: false });
  }
}
