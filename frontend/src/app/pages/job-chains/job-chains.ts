import { Component, OnInit, signal, ChangeDetectionStrategy, effect, untracked } from '@angular/core';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Wails, ChainStatus } from '../../core/services/wails';
import { JobChainService } from '../../core/services/job-chain';

@Component({
  selector: 'app-job-chains',
  imports: [
    CommonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatButtonModule,
    MatTableModule,
    MatTooltipModule
  ],
  templateUrl: './job-chains.html',
  styleUrl: './job-chains.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobChains implements OnInit {
  protected chains = signal<ChainStatus[]>([]);
  protected loading = signal(false);
  protected displayedColumns: string[] = ['label', 'status', 'createdAt', 'actions'];

  constructor(
    private wails: Wails,
    private jobChainService: JobChainService,
    private router: Router
  ) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (!job || !job.chainId) return;

      const chains = untracked(() => this.chains());
      if (!chains.some(c => c.chainId === job.chainId)) return;

      this.refreshChain(job.chainId);
    });
  }

  async ngOnInit(): Promise<void> {
    await this.loadChains();
  }

  async loadChains(): Promise<void> {
    this.loading.set(true);
    try {
      const list = await this.jobChainService.getAllChains(100, 0);
      const statuses = await Promise.all(list.map(c => this.jobChainService.getChainStatus(c.id)));
      this.chains.set(statuses);
    } catch (error: any) {
      await this.wails.logToFile(`[JobChains] Failed to load chains: ${error?.message || String(error)}`);
    } finally {
      this.loading.set(false);
    }
  }

  private async refreshChain(chainId: string): Promise<void> {
    try {
      const status = await this.jobChainService.getChainStatus(chainId);
      this.chains.update(chains => chains.map(c => (c.chainId === chainId ? status : c)));
    } catch (error: any) {
      await this.wails.logToFile(`[JobChains] Failed to refresh chain ${chainId}: ${error?.message || String(error)}`);
    }
  }

  viewChainDetail(id: string): void {
    this.router.navigate(['/job-chain', id]);
  }

  async deleteChain(event: Event, id: string): Promise<void> {
    event.stopPropagation();
    try {
      await this.jobChainService.deleteChain(id);
      this.chains.update(chains => chains.filter(c => c.chainId !== id));
    } catch (error: any) {
      await this.wails.logToFile(`[JobChains] Failed to delete chain: ${error?.message || String(error)}`);
    }
  }

  statusColor(status: string): string {
    switch (status) {
      case 'completed': return 'primary';
      case 'running': return 'accent';
      case 'failed': return 'warn';
      default: return '';
    }
  }
}
