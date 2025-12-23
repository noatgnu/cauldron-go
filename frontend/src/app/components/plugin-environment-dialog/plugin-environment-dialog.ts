import { Component, OnInit, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatCardModule } from '@angular/material/card';
import { MatDividerModule } from '@angular/material/divider';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';
import { Wails, PythonEnvironment, REnvironment, PluginEnvironmentBinding, VirtualEnvironment, RenvEnvironment } from '../../core/services/wails';

export interface PluginEnvironmentDialogData {
  pluginId: string;
  pluginName: string;
  runtimeType: string;
}

@Component({
  selector: 'app-plugin-environment-dialog',
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatCardModule,
    MatDividerModule,
    MatListModule,
    MatTooltipModule,
    MatSelectModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule
  ],
  templateUrl: './plugin-environment-dialog.html',
  styleUrl: './plugin-environment-dialog.scss',
})
export class PluginEnvironmentDialog implements OnInit {
  data = inject<PluginEnvironmentDialogData>(MAT_DIALOG_DATA);
  dialogRef = inject(MatDialogRef<PluginEnvironmentDialog>);
  wails = inject(Wails);
  snackBar = inject(MatSnackBar);

  loading = signal(true);
  pythonBinding = signal<PluginEnvironmentBinding | null>(null);
  rBinding = signal<PluginEnvironmentBinding | null>(null);
  pythonEnvironments = signal<PythonEnvironment[]>([]);
  rEnvironments = signal<REnvironment[]>([]);
  activePythonEnv = signal<PythonEnvironment | null>(null);
  activeREnv = signal<REnvironment | null>(null);

  showVenvCreation = signal(false);
  showRenvCreation = signal(false);
  selectedBasePython = signal<string>('');
  renvName = signal<string>('');
  renvPackages = signal<string>('');
  creatingEnvironment = signal(false);
  creationProgress = signal<string>('');

  basePythonEnvironments = computed(() => {
    return this.pythonEnvironments().filter(env => !env.isVirtual);
  });

  async ngOnInit() {
    await this.loadData();
    this.setupProgressListener();
  }

  private setupProgressListener() {
    this.wails.progress$.subscribe(progress => {
      if (progress && (progress.type === 'install' || progress.type === 'generic')) {
        this.creationProgress.set(progress.message);
      }
    });
  }

  async loadData() {
    this.loading.set(true);
    try {
      const getPythonBinding = async () => {
        try {
          return await this.wails.getPluginEnvironmentBinding(this.data.pluginId, 'python');
        } catch {
          return null;
        }
      };

      const getRBinding = async () => {
        try {
          return await this.wails.getPluginEnvironmentBinding(this.data.pluginId, 'r');
        } catch {
          return null;
        }
      };

      const [pythonBinding, rBinding, pythonEnvs, rEnvs, activePython, activeR] = await Promise.all([
        getPythonBinding(),
        getRBinding(),
        this.wails.detectPythonEnvironments(),
        this.wails.detectREnvironments(),
        this.wails.getActivePythonEnvironment(),
        this.wails.getActiveREnvironment()
      ]);

      this.pythonBinding.set(pythonBinding);
      this.rBinding.set(rBinding);
      this.pythonEnvironments.set(pythonEnvs || []);
      this.rEnvironments.set(rEnvs || []);
      this.activePythonEnv.set(activePython);
      this.activeREnv.set(activeR);
    } catch (error) {
      console.error('Failed to load environment data:', error);
      this.showError('Failed to load environment data');
    } finally {
      this.loading.set(false);
    }
  }

  needsPython(): boolean {
    return this.data.runtimeType === 'python' || this.data.runtimeType === 'pythonWithR';
  }

  needsR(): boolean {
    return this.data.runtimeType === 'r' || this.data.runtimeType === 'pythonWithR';
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

  async onPythonEnvSelected(event: any) {
    const selected = event.options[0];
    if (!selected || !selected.selected) return;

    const envPath = selected.value;
    try {
      await this.wails.bindPluginToEnvironment(this.data.pluginId, 'python', 0, envPath);
      await this.loadData();
      this.showSuccess('Python environment bound successfully');
    } catch (error) {
      console.error('Failed to bind Python environment:', error);
      this.showError('Failed to bind Python environment');
    }
  }

  async onREnvSelected(event: any) {
    const selected = event.options[0];
    if (!selected || !selected.selected) return;

    const envPath = selected.value;
    try {
      await this.wails.bindPluginToEnvironment(this.data.pluginId, 'r', 0, envPath);
      await this.loadData();
      this.showSuccess('R environment bound successfully');
    } catch (error) {
      console.error('Failed to bind R environment:', error);
      this.showError('Failed to bind R environment');
    }
  }

  async unbindEnvironment(envType: 'python' | 'r') {
    try {
      await this.wails.deletePluginEnvironmentBinding(this.data.pluginId, envType);
      if (envType === 'python') {
        this.pythonBinding.set(null);
      } else {
        this.rBinding.set(null);
      }
      this.showSuccess(`${envType === 'python' ? 'Python' : 'R'} environment unbound`);
    } catch (error) {
      console.error('Failed to unbind environment:', error);
      this.showError('Failed to unbind environment');
    }
  }

  startVenvCreation() {
    const baseEnvs = this.basePythonEnvironments();
    if (baseEnvs.length === 0) {
      this.showError('No base Python environments detected. Please configure Python in Settings first.');
      return;
    }

    const activePython = this.activePythonEnv();
    const defaultEnv = activePython && !activePython.isVirtual
      ? activePython.path
      : baseEnvs[0].path;

    this.selectedBasePython.set(defaultEnv);
    this.showVenvCreation.set(true);
  }

  cancelVenvCreation() {
    this.showVenvCreation.set(false);
    this.selectedBasePython.set('');
  }

  async confirmVenvCreation() {
    const basePython = this.selectedBasePython();
    if (!basePython) {
      this.showError('Please select a base Python environment');
      return;
    }

    const venvPath = await this.wails.openDirectoryDialog('Select location for virtual environment');
    if (!venvPath) return;

    this.creatingEnvironment.set(true);
    this.creationProgress.set('Creating virtual environment...');

    try {
      await this.wails.createPythonVirtualEnv(basePython, venvPath, this.data.pluginId);

      const venvs = await this.wails.getVirtualEnvironments();
      const newVenv = venvs.find(v => v.Path.includes(venvPath) || venvPath.includes(v.Name));

      if (newVenv) {
        this.creationProgress.set('Binding environment to plugin...');
        await this.wails.bindPluginToEnvironment(this.data.pluginId, 'python', newVenv.ID, newVenv.Path);
      }

      await this.loadData();
      this.showSuccess(`Virtual environment created and bound to ${this.data.pluginName}`);
      this.cancelVenvCreation();
    } catch (error) {
      console.error('Failed to create virtual environment:', error);
      this.showError(`Failed to create virtual environment: ${error}`);
    } finally {
      this.creatingEnvironment.set(false);
      this.creationProgress.set('');
    }
  }

  startRenvCreation() {
    const rEnvs = this.rEnvironments();
    if (rEnvs.length === 0) {
      this.showError('No R environments detected. Please configure R in Settings first.');
      return;
    }

    this.renvName.set('');
    this.renvPackages.set('');
    this.showRenvCreation.set(true);
  }

  cancelRenvCreation() {
    this.showRenvCreation.set(false);
    this.renvName.set('');
    this.renvPackages.set('');
  }

  async confirmRenvCreation() {
    const name = this.renvName().trim();
    if (!name) {
      this.showError('Please enter a name for the renv environment');
      return;
    }

    const packagesInput = this.renvPackages().trim();
    const packages = packagesInput ? packagesInput.split(',').map(p => p.trim()).filter(p => p) : [];

    this.creatingEnvironment.set(true);
    this.creationProgress.set('Creating renv environment...');

    try {
      await this.wails.createRenvEnvironment(name, packages, this.data.pluginId);

      const renvs = await this.wails.getRenvEnvironments();
      const newRenv = renvs.find(r => r.Name === name || r.ProjectPath.includes(name));

      if (newRenv) {
        this.creationProgress.set('Binding environment to plugin...');
        await this.wails.bindPluginToEnvironment(this.data.pluginId, 'r', newRenv.ID, newRenv.Path);
      }

      await this.loadData();
      this.showSuccess(`Renv environment created and bound to ${this.data.pluginName}`);
      this.cancelRenvCreation();
    } catch (error) {
      console.error('Failed to create renv environment:', error);
      this.showError(`Failed to create renv environment: ${error}`);
    } finally {
      this.creatingEnvironment.set(false);
      this.creationProgress.set('');
    }
  }

  private showSuccess(message: string) {
    this.snackBar.open(message, 'Close', {
      duration: 3000,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['success-snackbar']
    });
  }

  private showError(message: string) {
    this.snackBar.open(message, 'Close', {
      duration: 5000,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['error-snackbar']
    });
  }
}
