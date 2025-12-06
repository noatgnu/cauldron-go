import { Component, OnInit, signal } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { Toolbar } from './layout/toolbar/toolbar';
import { Sidenav } from './layout/sidenav/sidenav';
import { Breadcrumbs } from './layout/breadcrumbs/breadcrumbs';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, MatSidenavModule, Toolbar, Sidenav, Breadcrumbs],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  protected readonly title = signal('cauldron-ui');
  protected readonly sidenavOpened = signal(true);

  constructor(private router: Router) {}

  ngOnInit(): void {
    this.setupMenuEventListeners();
  }

  toggleSidenav(): void {
    this.sidenavOpened.update(v => !v);
  }

  closeSidenav(): void {
  }

  private setupMenuEventListeners(): void {
    if (!window.runtime) return;

    window.runtime.EventsOn('menu:view-home', () => {
      this.router.navigate(['/']);
    });

    window.runtime.EventsOn('menu:view-jobs', () => {
      this.router.navigate(['/jobs']);
    });

    window.runtime.EventsOn('menu:view-plugins', () => {
      this.router.navigate(['/plugins']);
    });

    window.runtime.EventsOn('menu:view-settings', () => {
      this.router.navigate(['/settings']);
    });

    window.runtime.EventsOn('menu:settings', () => {
      this.router.navigate(['/settings']);
    });

    window.runtime.EventsOn('menu:pca', () => {
      this.router.navigate(['/analysis/pca']);
    });

    window.runtime.EventsOn('menu:imputation', () => {
      this.router.navigate(['/analysis/imputation']);
    });

    window.runtime.EventsOn('menu:normalization', () => {
      this.router.navigate(['/analysis/normalization']);
    });

    window.runtime.EventsOn('menu:limma', () => {
      this.router.navigate(['/analysis/limma']);
    });
  }
}
