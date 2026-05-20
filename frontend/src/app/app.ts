import { Component, OnInit, signal, ChangeDetectionStrategy, HostListener, ChangeDetectorRef } from '@angular/core';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { filter, firstValueFrom } from 'rxjs';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatDialog } from '@angular/material/dialog';
import { Events, Window } from '@wailsio/runtime';
import { Sidenav } from './layout/sidenav/sidenav';
import { Breadcrumbs } from './layout/breadcrumbs/breadcrumbs';
import { ProtocolHandlerService } from './core/services/protocol-handler.service';
import { LoadingScreenComponent } from './components/loading-screen/loading-screen';
import { LoadingService } from './services/loading';
import { ThemeService } from './core/services/theme.service';
import { NotificationService } from './core/services/notification.service';
import { ConfirmPluginInstallDialog, PluginInstallConfirmData, PluginInstallConfirmResult } from './components/confirm-plugin-install-dialog/confirm-plugin-install-dialog';
import { ConfirmPluginInstallation, SaveGitAuthConfig, ConfirmPluginInstallationWithRegistry } from '../../bindings/github.com/noatgnu/cauldron-go/app';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, MatSidenavModule, Sidenav, Breadcrumbs, LoadingScreenComponent],
  templateUrl: './app.html',
  styleUrl: './app.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class App implements OnInit {
  protected readonly title = signal('cauldron-ui');

  constructor(
    private router: Router,
    private cdr: ChangeDetectorRef,
    private protocolHandler: ProtocolHandlerService,
    private loadingService: LoadingService,
    private dialog: MatDialog,
    private notification: NotificationService,
    private themeService: ThemeService
  ) {
    this.router.events.pipe(filter(e => e instanceof NavigationEnd)).subscribe(() => {
      setTimeout(() => this.cdr.detectChanges());
    });
  }

  @HostListener('document:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'F12') {
      Window.OpenDevTools().catch(() => {});
    }
  }

  ngOnInit(): void {
    this.setupMenuEventListeners();
    this.setupPluginInstallationListeners();
  }

  private setupMenuEventListeners(): void {
    Events.On('menu:view-home', () => {
      this.router.navigate(['/']);
    });

    Events.On('menu:view-jobs', () => {
      this.router.navigate(['/jobs']);
    });

    Events.On('menu:view-plugin-list', () => {
      this.router.navigate(['/plugin-list']);
    });

    Events.On('menu:settings', () => {
      this.router.navigate(['/settings']);
    });

    Events.On('menu:about', () => {
      this.router.navigate(['/about']);
    });
  }

  private setupPluginInstallationListeners(): void {
    Events.On('plugin:install:request', async (ev: any) => {
      const data = ev.data as PluginInstallConfirmData;
      const dialogRef = this.dialog.open(ConfirmPluginInstallDialog, {
        width: '600px',
        data: data,
        disableClose: true
      });

      const result = await firstValueFrom(dialogRef.afterClosed()) as PluginInstallConfirmResult;
      if (result && result.confirmed) {
        try {
          if (result.sshKeyPath) {
            await SaveGitAuthConfig(data.repo, result.sshKeyPath, result.passphrase || '');
          }

          if (data.registry) {
            await ConfirmPluginInstallationWithRegistry(
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

    Events.On('plugin:install:start', () => {
      this.notification.showInfo('Installing plugin...');
    });

    Events.On('plugin:install:success', () => {
      this.notification.showSuccess('Plugin installed successfully!');
    });

    Events.On('plugin:install:error', (ev: any) => {
      const data = ev.data as { repo: string; error: string };
      this.notification.showError(`Installation failed: ${data.error}`);
    });
  }
}
