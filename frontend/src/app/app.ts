import { Component, OnInit, signal } from '@angular/core';
import { RouterOutlet, Router } from '@angular/router';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatDialog } from '@angular/material/dialog';
import { Sidenav } from './layout/sidenav/sidenav';
import { Breadcrumbs } from './layout/breadcrumbs/breadcrumbs';
import { ProtocolHandlerService } from './core/services/protocol-handler.service';
import { LoadingScreenComponent } from './components/loading-screen/loading-screen';
import { LoadingService } from './services/loading';
import { ThemeService } from './core/services/theme.service';
import { NotificationService } from './core/services/notification.service';
import { ConfirmPluginInstallDialog, PluginInstallConfirmData, PluginInstallConfirmResult } from './components/confirm-plugin-install-dialog/confirm-plugin-install-dialog';
import { ConfirmPluginInstallation, SaveGitAuthConfig } from '../wailsjs/go/main/App';

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
    private notification: NotificationService,
    private themeService: ThemeService
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
  }

  private setupPluginInstallationListeners(): void {
    if (!window.runtime) return;

    window.runtime.EventsOn('plugin:install:request', (data: PluginInstallConfirmData) => {
      const dialogRef = this.dialog.open(ConfirmPluginInstallDialog, {
        width: '600px',
        data: data,
        disableClose: true
      });

      dialogRef.afterClosed().subscribe(async (result: PluginInstallConfirmResult) => {
        if (result && result.confirmed) {
          try {
            if (result.sshKeyPath) {
              await SaveGitAuthConfig(data.repo, result.sshKeyPath, result.passphrase || '');
            }

            if (data.registry) {
              await window.go.main.App.ConfirmPluginInstallationWithRegistry(
                data.repo,
                data.ref || '',
                data.registry
              );
              this.notification.showInfo('Installing plugin...');
            } else {
              await ConfirmPluginInstallation(data.repo, data.ref || '');
              this.notification.showInfo('Installing plugin...');
            }
          } catch (err: any) {
            this.notification.showError(`Installation failed: ${err}`);
          }
        }
      });
    });

    window.runtime.EventsOn('plugin:install:start', (data: { repo: string }) => {
      this.notification.showInfo('Installing plugin...');
    });

    window.runtime.EventsOn('plugin:install:success', (data: { repo: string }) => {
      this.notification.showSuccess('Plugin installed successfully!');
    });

    window.runtime.EventsOn('plugin:install:error', (data: { repo: string; error: string }) => {
      this.notification.showError(`Installation failed: ${data.error}`);
    });
  }
}
