import { Component, Inject, OnInit, OnDestroy, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { Wails } from '../../core/services/wails';
import { Subscription } from 'rxjs';

export interface PluginInstallProgressData {
  repoURL: string;
  commitHash?: string;
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
  styleUrl: './plugin-install-progress.scss'
})
export class PluginInstallProgress implements OnInit, OnDestroy {
  protected currentStatus = signal('Initializing installation...');
  protected progress = signal(0);
  protected stages = signal<string[]>([]);
  protected completed = signal(false);
  protected error = signal('');
  
  private subscription: Subscription = new Subscription();

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: PluginInstallProgressData,
    private dialogRef: MatDialogRef<PluginInstallProgress>,
    private wails: Wails
  ) {}

  ngOnInit() {
    this.startInstallation();
    
    const sub = this.wails.listen('plugin:install:progress', (event: any) => {
      if (event.repo === this.data.repoURL) {
        this.currentStatus.set(event.status);
        this.stages.update(s => [...s, event.status]);
        this.updateProgress();
      }
    });
    this.subscription.add(sub);
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  private async startInstallation() {
    try {
      await this.wails.installPluginFromRepo(this.data.repoURL, this.data.commitHash || '');
      this.completed.set(true);
      this.currentStatus.set('Installation completed successfully!');
      this.progress.set(100);
    } catch (err: any) {
      this.error.set(err.toString() || 'An unknown error occurred during installation');
      this.currentStatus.set('Installation failed');
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