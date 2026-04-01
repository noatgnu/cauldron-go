import { Component, Inject, OnInit, OnDestroy, signal, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { Wails } from '../../core/services/wails';
import { NotificationService } from '../../core/services/notification.service';
import { Subscription } from 'rxjs';

export interface PluginInstallProgressData {
  repoURL: string;
  commitHash?: string;
  sshKeyPath?: string;
  passphrase?: string;
  createVenv?: boolean;
  basePythonPath?: string;
  createRenv?: boolean;
  renvName?: string;
}

@Component({
  selector: 'app-plugin-install-progress',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatProgressBarModule,
    MatIconModule,
    MatButtonModule
  ],
  templateUrl: './plugin-install-progress.html',
  styleUrl: './plugin-install-progress.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class PluginInstallProgress implements OnInit, OnDestroy {
  protected currentStatus = signal('Initializing installation...');
  protected progress = signal(0);
  protected stages = signal<string[]>([]);
  protected completed = signal(false);
  protected error = signal('');
  protected installedPluginId = signal<string | null>(null);

  private subscription: Subscription = new Subscription();

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: PluginInstallProgressData,
    private dialogRef: MatDialogRef<PluginInstallProgress>,
    private wails: Wails,
    private notification: NotificationService
  ) {}

  ngOnInit() {
    this.startInstallation();

    const pluginProgressSub = this.wails.listen('plugin:install:progress', (event: any) => {
      if (event.repo === this.data.repoURL) {
        this.currentStatus.set(event.status);
        this.stages.update(s => [...s, event.status]);
        this.updateProgress();
      }
    });
    this.subscription.add(pluginProgressSub);

    const envProgressSub = this.wails.listen('progress', (event: any) => {
      if (event.type === 'install' &&
          (event.id === 'python-venv' ||
           event.id === 'python-packages' ||
           event.id === 'python-requirements' ||
           event.id === 'r-packages' ||
           event.id === 'renv-init')) {
        this.currentStatus.set(event.message);
        if (event.percentage) {
          this.progress.set(event.percentage);
        }
      }
    });
    this.subscription.add(envProgressSub);
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  private async startInstallation() {
    try {
      if (this.data.sshKeyPath) {
        this.currentStatus.set('Saving Git authentication configuration...');
        await this.wails.saveGitAuthConfig(
          this.data.repoURL,
          this.data.sshKeyPath,
          this.data.passphrase || ''
        );
      }

      this.currentStatus.set('Installing plugin from repository...');
      const result = await this.wails.installPluginFromRepo(this.data.repoURL, this.data.commitHash || '');

      if (result && typeof result === 'object' && 'pluginId' in result) {
        this.installedPluginId.set((result as any).pluginId);

        if (this.data.createVenv || this.data.createRenv) {
          await this.createEnvironments((result as any).pluginId);
        }

        await this.checkAndPromptForRequirements((result as any).pluginId);
      }

      this.completed.set(true);
      this.currentStatus.set('Installation completed successfully!');
      this.progress.set(100);

      if (window.runtime) {
        window.runtime.EventsEmit('plugin:install:success');
      }
    } catch (err: any) {
      this.error.set(err.toString() || 'An unknown error occurred during installation');
      this.currentStatus.set('Installation failed');
    }
  }

  private async createEnvironments(pluginId: string) {
    try {
      if (this.data.createVenv && this.data.basePythonPath) {
        const venvPath = await this.wails.getDefaultVenvPath(pluginId);
        await this.wails.createPythonVirtualEnv(this.data.basePythonPath, venvPath, pluginId);

        const venvs = await this.wails.getVirtualEnvironments();
        const newVenv = venvs.find(v => v.Path.includes(venvPath) || venvPath.includes(v.Name));

        if (newVenv) {
          await this.wails.bindPluginToEnvironment(pluginId, 'python', newVenv.ID, newVenv.Path);
          this.notification.showSuccess('Python virtual environment created and bound');
        }
      }

      if (this.data.createRenv && this.data.renvName) {
        await this.wails.createRenvEnvironment(this.data.renvName, [], pluginId, false);

        const renvs = await this.wails.getRenvEnvironments();
        const newRenv = renvs.find(r => r.Name === this.data.renvName);

        if (newRenv) {
          await this.wails.bindPluginToEnvironment(pluginId, 'r', newRenv.ID, newRenv.Path);
          this.notification.showSuccess('R environment created and bound');
        }
      }
    } catch (err) {
      await this.wails.logToFile(`[PluginInstallProgress] Failed to create environments: ${err}`);
      this.notification.showWarning('Plugin installed but environment creation failed. You can create environments later from the plugin list.');
    }
  }

  private async checkAndPromptForRequirements(pluginId: string) {
    try {
      if (!this.data.createVenv && !this.data.createRenv) {
        return;
      }

      const requirements = await this.wails.getPluginRequirements(pluginId);

      if (requirements && requirements.requirementsExist) {
        await this.installRequirements(pluginId);
      }
    } catch (err) {
      await this.wails.logToFile(`[PluginInstallProgress] Failed to check requirements: ${err}`);
    }
  }

  private async installRequirements(pluginId: string) {
    try {
      await this.wails.installPluginRequirements(pluginId);
      this.notification.showSuccess('Requirements installed successfully!');
    } catch (err: any) {
      await this.wails.logToFile(`[PluginInstallProgress] Failed to install requirements: ${err}`);
      this.notification.showError(`Failed to install requirements: ${err}`);
    }
  }

  private updateProgress() {
    const totalStages = 6;
    const current = this.stages().length;
    this.progress.set(Math.min(95, (current / totalStages) * 100));
  }

  close() {
    this.dialogRef.close(this.completed());
  }
}