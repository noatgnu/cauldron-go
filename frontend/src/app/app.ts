import { Component, OnInit, signal } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Sidenav } from './layout/sidenav/sidenav';
import { Breadcrumbs } from './layout/breadcrumbs/breadcrumbs';
import { ProtocolHandlerService } from './core/services/protocol-handler.service';
import { LoadingScreenComponent } from './components/loading-screen/loading-screen';
import { LoadingService } from './services/loading';
import { ConfirmPluginInstallDialog, PluginInstallConfirmData } from './components/confirm-plugin-install-dialog/confirm-plugin-install-dialog';
import { ConfirmPluginInstallation } from '../wailsjs/go/main/App';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, MatSidenavModule, Sidenav, Breadcrumbs, LoadingScreenComponent],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  protected readonly title = signal('cauldron-ui');

  constructor(
    private router: Router,
    private protocolHandler: ProtocolHandlerService,
    private loadingService: LoadingService,
    private dialog: MatDialog,
    private snackBar: MatSnackBar
  ) {}

  ngOnInit(): void {
    this.setupMenuEventListeners();
    this.setupPluginInstallationListeners();
  }

  private setupMenuEventListeners(): void {
    if (!window.runtime) return;

    window.runtime.EventsOn('menu:view-home', () => {
      this.router.navigate(['/']);
    });

    window.runtime.EventsOn('menu:view-jobs', () => {
      this.router.navigate(['/jobs']);
    });

    window.runtime.EventsOn('menu:view-plugin-list', () => {
      this.router.navigate(['/plugin-list']);
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

  private setupPluginInstallationListeners(): void {
    if (!window.runtime) return;

    window.runtime.EventsOn('plugin:install:request', (data: PluginInstallConfirmData) => {
      const dialogRef = this.dialog.open(ConfirmPluginInstallDialog, {
        width: '600px',
        data: data,
        disableClose: true
      });

      dialogRef.afterClosed().subscribe(confirmed => {
        if (confirmed) {
          ConfirmPluginInstallation(data.repo).then(() => {
            this.snackBar.open('Installing plugin...', 'Close', { duration: 3000 });
          }).catch(err => {
            this.snackBar.open(`Installation failed: ${err}`, 'Close', { duration: 5000 });
          });
        }
      });
    });

    window.runtime.EventsOn('plugin:install:start', (data: { repo: string }) => {
      this.snackBar.open('Installing plugin...', 'Close', { duration: 3000 });
    });

    window.runtime.EventsOn('plugin:install:success', (data: { repo: string }) => {
      this.snackBar.open('Plugin installed successfully!', 'Close', { duration: 5000 });
    });

    window.runtime.EventsOn('plugin:install:error', (data: { repo: string; error: string }) => {
      this.snackBar.open(`Installation failed: ${data.error}`, 'Close', { duration: 5000 });
    });
  }
}
