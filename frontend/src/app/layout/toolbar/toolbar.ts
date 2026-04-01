import { Component, output, signal, OnInit, ChangeDetectionStrategy, effect } from '@angular/core';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatBadgeModule } from '@angular/material/badge';
import { Router } from '@angular/router';
import { Wails } from '../../core/services/wails';

@Component({
  selector: 'app-toolbar',
  imports: [MatToolbarModule, MatButtonModule, MatIconModule, MatBadgeModule],
  templateUrl: './toolbar.html',
  styleUrl: './toolbar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Toolbar implements OnInit {
  menuToggle = output<void>();
  protected activeJobsCount = signal(0);

  constructor(private router: Router, private wails: Wails) {
    effect(() => {
      const job = this.wails.jobUpdate();
      if (job) {
        this.updateActiveJobsCount();
      }
    });
  }

  async ngOnInit() {
    await this.updateActiveJobsCount();
  }

  async updateActiveJobsCount() {
    try {
      const jobs = await this.wails.getAllJobs();
      const activeCount = jobs.filter(job =>
        job.status === 'in_progress' || job.status === 'pending'
      ).length;
      this.activeJobsCount.set(activeCount);
    } catch (err) {
      this.activeJobsCount.set(0);
    }
  }

  onMenuToggle(): void {
    this.menuToggle.emit();
  }

  navigateToJobs(): void {
    this.router.navigate(['/jobs']);
  }

  navigateToSettings(): void {
    this.router.navigate(['/settings']);
  }
}
