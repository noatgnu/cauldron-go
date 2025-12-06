import { Component, output, signal, OnInit, OnDestroy } from '@angular/core';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatBadgeModule } from '@angular/material/badge';
import { Router } from '@angular/router';
import { Wails, Job } from '../../core/services/wails';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-toolbar',
  imports: [MatToolbarModule, MatButtonModule, MatIconModule, MatBadgeModule],
  templateUrl: './toolbar.html',
  styleUrl: './toolbar.scss',
})
export class Toolbar implements OnInit, OnDestroy {
  menuToggle = output<void>();
  protected activeJobsCount = signal(0);
  private jobUpdateSubscription?: Subscription;

  constructor(private router: Router, private wails: Wails) {}

  async ngOnInit() {
    await this.updateActiveJobsCount();

    this.jobUpdateSubscription = this.wails.jobUpdate$.subscribe((job: Job | null) => {
      if (job) {
        this.updateActiveJobsCount();
      }
    });
  }

  ngOnDestroy() {
    if (this.jobUpdateSubscription) {
      this.jobUpdateSubscription.unsubscribe();
    }
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
