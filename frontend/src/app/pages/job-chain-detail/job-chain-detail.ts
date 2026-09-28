import { Component, OnInit, OnDestroy, signal, ChangeDetectionStrategy, effect, untracked } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription, firstValueFrom } from 'rxjs';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { Wails, ChainStatus } from '../../core/services/wails';
import { JobChainService } from '../../core/services/job-chain';
import { ConfirmDialogComponent } from '../../components/confirm-dialog/confirm-dialog';
import { PromptDialogComponent } from '../../components/prompt-dialog/prompt-dialog';
import { NotificationService } from '../../core/services/notification.service';

@Component({
  selector: 'app-job-chain-detail',
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './job-chain-detail.html',
  styleUrl: './job-chain-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobChainDetail implements OnInit, OnDestroy {
  protected chain = signal<ChainStatus | null>(null);
  protected loading = signal(true);
  protected savingRecipe = signal(false);
  protected displayedColumns: string[] = ['status', 'stage', 'createdAt', 'actions'];
  private chainId = '';
  private paramMapSubscription?: Subscription;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private wails: Wails,
    private jobChainService: JobChainService,
    private dialog: MatDialog,
    private notification: NotificationService
  ) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (!job || job.chainId !== untracked(() => this.chainId)) return;
      this.loadChain();
    });
  }

  ngOnInit(): void {
    this.paramMapSubscription = this.route.paramMap.subscribe(async params => {
      const id = params.get('id');
      if (!id) return;
      this.chainId = id;
      await this.loadChain();
    });
  }

  ngOnDestroy(): void {
    this.paramMapSubscription?.unsubscribe();
  }

  async loadChain(): Promise<void> {
    this.loading.set(true);
    try {
      const status = await this.jobChainService.getChainStatus(this.chainId);
      this.chain.set(status);
    } catch (error: any) {
      await this.wails.logToFile(`[JobChainDetail] Failed to load chain: ${error?.message || String(error)}`);
    } finally {
      this.loading.set(false);
    }
  }

  stageLabel(stage: any): string {
    if (stage.job?.name) return `Stage ${stage.stageIndex + 1}: ${stage.job.name}`;
    return `Stage ${stage.stageIndex + 1}`;
  }

  viewJob(id: string): void {
    this.router.navigate(['/jobs', id]);
  }

  async deleteChain(): Promise<void> {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      width: '400px',
      disableClose: true,
      data: {
        title: 'Delete this chain?',
        message: 'This deletes the chain and all of its jobs, including any still pending or in progress. This cannot be undone.',
        confirmText: 'Yes, Delete',
        cancelText: 'Cancel'
      }
    });

    const confirmed = await firstValueFrom(dialogRef.afterClosed());
    if (!confirmed) return;

    try {
      await this.jobChainService.deleteChain(this.chainId);
      await this.router.navigate(['/job-chains']);
    } catch (error: any) {
      await this.wails.logToFile(`[JobChainDetail] Failed to delete chain: ${error?.message || String(error)}`);
    }
  }

  async saveAsRecipe(): Promise<void> {
    const chain = this.chain();
    if (!chain) return;

    const dialogRef = this.dialog.open(PromptDialogComponent, {
      width: '440px',
      disableClose: true,
      data: {
        title: 'Save chain as recipe',
        message: 'This reconstructs a reusable recipe from exactly what ran in this chain.',
        label: 'Recipe label',
        value: chain.label,
        confirmText: 'Save'
      }
    });

    const label = await firstValueFrom(dialogRef.afterClosed());
    if (!label) return;

    this.savingRecipe.set(true);
    try {
      const recipe = await this.jobChainService.saveAsRecipe(this.chainId, label, `Saved from job chain ${chain.chainId}`);
      this.notification.showSuccess('Recipe saved.');
      await this.router.navigate(['/recipe', recipe.id]);
    } catch (error: any) {
      await this.wails.logToFile(`[JobChainDetail] Failed to save chain as recipe: ${error?.message || String(error)}`);
      this.notification.showError('Failed to save chain as a recipe');
    } finally {
      this.savingRecipe.set(false);
    }
  }

  getStatusColor(status: string): string {
    switch (status) {
      case 'completed': return 'primary';
      case 'running': return 'accent';
      case 'failed': return 'warn';
      case 'blocked': return 'warn';
      default: return '';
    }
  }

  getStatusIcon(status: string): string {
    switch (status) {
      case 'completed': return 'check_circle';
      case 'running': return 'sync';
      case 'failed': return 'error';
      case 'blocked': return 'block';
      default: return 'schedule';
    }
  }
}
