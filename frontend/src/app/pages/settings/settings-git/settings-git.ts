import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatDividerModule } from '@angular/material/divider';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Wails } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';

interface GitAuthConfig {
  id: number;
  repositoryURL: string;
  sshKeyPath: string;
  hasPassphrase: boolean;
  createdAt: number;
  updatedAt: number;
}

@Component({
  selector: 'app-settings-git',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatListModule,
    MatDividerModule,
    MatProgressSpinnerModule,
    MatTooltipModule
  ],
  templateUrl: './settings-git.html',
  styleUrl: './settings-git.scss',
})
export class SettingsGit implements OnInit {
  private wails = inject(Wails);
  private notification = inject(NotificationService);

  loading = signal(true);
  configs = signal<GitAuthConfig[]>([]);

  newRepoURL = signal('');
  newSSHKeyPath = signal('');
  newPassphrase = signal('');
  showPassphrase = signal(false);
  validatingKey = signal(false);

  async ngOnInit() {
    await this.loadConfigs();
  }

  async loadConfigs() {
    this.loading.set(true);
    try {
      const configs = await this.wails.getAllGitAuthConfigs();
      this.configs.set(configs || []);
    } catch (error) {
      this.notification.showError('Failed to load Git authentication configs');
    } finally {
      this.loading.set(false);
    }
  }

  async browseSSHKey() {
    try {
      const path = await this.wails.openFile('Select SSH Private Key');
      if (path) {
        this.newSSHKeyPath.set(path);
      }
    } catch (error) {
      this.notification.showError('Failed to open file dialog');
    }
  }

  async validateKey() {
    const keyPath = this.newSSHKeyPath().trim();
    if (!keyPath) {
      this.notification.showError('Please select an SSH key file');
      return;
    }

    this.validatingKey.set(true);
    try {
      await this.wails.validateSSHKey(keyPath, this.newPassphrase());
      this.notification.showSuccess('SSH key is valid');
    } catch (error) {
      this.notification.showError(`Invalid SSH key: ${error}`);
    } finally {
      this.validatingKey.set(false);
    }
  }

  async saveConfig() {
    const repoURL = this.newRepoURL().trim();
    const keyPath = this.newSSHKeyPath().trim();

    if (!repoURL) {
      this.notification.showError('Please enter a repository URL');
      return;
    }

    if (!keyPath) {
      this.notification.showError('Please select an SSH key file');
      return;
    }

    try {
      await this.wails.saveGitAuthConfig(repoURL, keyPath, this.newPassphrase());
      this.notification.showSuccess('Git authentication config saved');

      this.newRepoURL.set('');
      this.newSSHKeyPath.set('');
      this.newPassphrase.set('');
      this.showPassphrase.set(false);

      await this.loadConfigs();
    } catch (error) {
      this.notification.showError(`Failed to save config: ${error}`);
    }
  }

  async deleteConfig(repoURL: string) {
    try {
      await this.wails.deleteGitAuthConfig(repoURL);
      this.notification.showSuccess('Git authentication config deleted');
      await this.loadConfigs();
    } catch (error) {
      this.notification.showError(`Failed to delete config: ${error}`);
    }
  }

  togglePassphraseVisibility() {
    this.showPassphrase.set(!this.showPassphrase());
  }

  formatTimestamp(timestamp: number): string {
    return new Date(timestamp * 1000).toLocaleString();
  }

  getRepoDisplayName(url: string): string {
    const parts = url.split('/');
    return parts.length >= 2 ? parts.slice(-2).join('/') : url;
  }
}
