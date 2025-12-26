import { Component, OnInit, signal } from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSelectModule } from '@angular/material/select';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatDialog } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import { Wails, REnvironment, RenvEnvironment, Config, PluginEnvironmentBinding } from '../../../core/services/wails';
import { PackagesModal } from '../../../components/packages-modal/packages-modal';
import { DownloadPortableEnvDialogComponent } from '../../../components/download-portable-env-dialog/download-portable-env-dialog';
import { ConfirmDialogComponent } from '../../../components/confirm-dialog/confirm-dialog';
import { PromptDialogComponent } from '../../../components/prompt-dialog/prompt-dialog';
import { BoundPluginsDialogComponent, BoundPlugin } from '../../../components/bound-plugins-dialog/bound-plugins-dialog';

@Component({
  selector: 'app-settings-r',
  imports: [
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    MatSelectModule,
    MatChipsModule,
    MatDividerModule,
    MatListModule,
    MatTooltipModule,
    MatSlideToggleModule
  ],
  templateUrl: './settings-r.html',
  styleUrl: './settings-r.scss',
})
export class SettingsR implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected rVersion = signal('');
  protected rEnvironments = signal<REnvironment[]>([]);
  protected renvEnvironments = signal<RenvEnvironment[]>([]);
  protected detectingREnvs = signal(false);
  protected installingRPackages = signal(false);
  protected creatingRenvEnv = signal(false);
  protected selectedREnv = signal<string>('');
  protected rInstallProgress = signal<{message: string, percentage: number} | null>(null);
  protected renvCreationProgress = signal<{message: string, percentage: number} | null>(null);
  protected bindings = signal<PluginEnvironmentBinding[]>([]);
  protected plugins = signal<any[]>([]);
  protected useGlobalCache = signal(false);

  constructor(
    private wails: Wails,
    private dialog: MatDialog,
    private snackBar: MatSnackBar
  ) {}

  private showError(message: string): void {
    this.snackBar.open(message, 'Close', {
      duration: 5000,
      horizontalPosition: 'center',
      verticalPosition: 'top',
      panelClass: ['error-snackbar']
    });
  }

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    await this.loadVersion();
    this.detectAllREnvironments();
    await this.loadRenvEnvironments();
    await this.loadPlugins();
    this.setupProgressUpdates();
  }

  async loadPlugins(): Promise<void> {
    try {
      const plugins = await this.wails.getPluginsV2();
      this.plugins.set(plugins || []);
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to load plugins: ${error}`);
    }
  }

  setupProgressUpdates(): void {
    this.wails.progress$.subscribe(progress => {
      if (!progress) return;

      if (progress.type === 'install' || progress.type === 'download' || progress.type === 'extract') {
        const progressData = {
          message: progress.message,
          percentage: progress.percentage
        };

        const isCompleted = progress.status === 'completed' || progress.status === 'error';

        switch (progress.id) {
          case 'r-packages':
            this.rInstallProgress.set(isCompleted ? null : progressData);
            break;
          case 'renv-init':
          case 'renv-packages':
            this.renvCreationProgress.set(isCompleted ? null : progressData);
            if (isCompleted && progress.id === 'renv-packages') {
              this.creatingRenvEnv.set(false);
            }
            break;
          default:
            if (progress.id?.includes('r-portable')) {
              this.rInstallProgress.set(isCompleted ? null : progressData);
            }
            break;
        }
      }
    });
  }

  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to load settings: ${error}`);
    }
  }

  async loadVersion(): Promise<void> {
    try {
      const rVer = await this.wails.getRVersion();
      this.rVersion.set(rVer);
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to load version: ${error}`);
    }
  }

  async browseR(): Promise<void> {
    try {
      const path = await this.wails.openFile('Select R Executable');
      if (path) {
        this.config.update(c => ({ ...c, rPath: path }));
        await this.saveSetting('rPath', path);
        await this.loadVersion();
      }
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to browse for R: ${error}`);
    }
  }

  private async saveSetting(key: string, value: any): Promise<void> {
    try {
      await this.wails.setSetting(key, value);
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to save setting ${key}: ${error}`);
    }
  }

  async detectAllREnvironments(): Promise<void> {
    this.detectingREnvs.set(true);
    try {
      const envs = await this.wails.detectREnvironments();
      this.rEnvironments.set(envs || []);

      if (envs && envs.length > 0) {
        const activeEnv = await this.wails.getActiveREnvironment();
        if (activeEnv && activeEnv.path) {
          const foundEnv = envs.find(e => e.path === activeEnv.path);
          if (foundEnv) {
            this.selectedREnv.set(foundEnv.path);
          } else {
            this.selectedREnv.set('');
          }
        } else {
          this.selectedREnv.set('');
        }
      } else {
        this.selectedREnv.set('');
      }
    } catch (error) {
      this.rEnvironments.set([]);
      this.selectedREnv.set('');
    } finally {
      this.detectingREnvs.set(false);
    }
  }

  async selectREnvironment(envPath: string): Promise<void> {
    this.selectedREnv.set(envPath);
    this.config.update(c => ({ ...c, rPath: envPath }));
    await this.saveSetting('rPath', envPath);
    await this.wails.setActiveREnvironment(envPath);
    await this.loadVersion();
  }

  async installRPackages(): Promise<void> {
    const rPath = this.config().rPath;
    if (!rPath) {
      return;
    }

    try {
      this.installingRPackages.set(true);
      const requirementsPath = await this.wails.getBundledRequirementsPath('r');
      const packages = await this.wails.loadRPackagesFromFile(requirementsPath);
      await this.wails.installRPackages(rPath, packages);
    } catch (error) {
      this.showError('Failed to install R packages');
    } finally {
      this.installingRPackages.set(false);
    }
  }

  async viewRPackages(env: REnvironment): Promise<void> {
    const dialogRef = this.dialog.open(PackagesModal, {
      width: '600px',
      disableClose: true,
      data: {
        environmentName: env.name,
        packages: [],
        loading: true
      }
    });

    try {
      const packages = await this.wails.listRPackages(env.path);
      dialogRef.componentInstance.setPackages(packages);
      dialogRef.componentInstance.setLoading(false);
    } catch (error) {
      dialogRef.componentInstance.setLoading(false);
      this.snackBar.open('Failed to load R packages', 'Close', {
        duration: 3000,
        horizontalPosition: 'center',
        verticalPosition: 'top'
      });
    }
  }

  viewSelectedRPackages(): void {
    const selectedPath = this.selectedREnv();

    if (!selectedPath) {
      this.snackBar.open('No R environment selected', 'Close', {
        duration: 3000,
        horizontalPosition: 'center',
        verticalPosition: 'top'
      });
      return;
    }

    const env = this.rEnvironments().find(e => e.path === selectedPath);

    if (env) {
      this.viewRPackages(env);
    } else {
      this.snackBar.open('R environment not found', 'Close', {
        duration: 3000,
        horizontalPosition: 'center',
        verticalPosition: 'top'
      });
    }
  }

  downloadREnvironment(): void {
    const dialogRef = this.dialog.open(DownloadPortableEnvDialogComponent, {
      width: '600px',
      disableClose: true
    });

    const instance = dialogRef.componentInstance;
    instance.environment = 'r-portable';

    dialogRef.afterClosed().subscribe(() => {
      this.detectAllREnvironments();
      this.loadVersion();
    });
  }

  async loadRenvEnvironments(): Promise<void> {
    try {
      const [renvs, bindings] = await Promise.all([
        this.wails.getRenvEnvironments(),
        this.wails.getAllPluginEnvironmentBindings()
      ]);
      this.renvEnvironments.set(renvs || []);
      this.bindings.set(bindings || []);
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to load renv environments: ${error}`);
    }
  }

  async createRenvEnvironment(): Promise<void> {
    const nameDialog = this.dialog.open(PromptDialogComponent, {
      disableClose: true,
      data: {
        title: 'Create Renv Environment',
        label: 'Environment Name',
        placeholder: 'Enter name for renv environment',
        confirmText: 'Next'
      }
    });

    const name = await firstValueFrom(nameDialog.afterClosed());
    if (!name) return;

    const packagesDialog = this.dialog.open(PromptDialogComponent, {
      disableClose: true,
      data: {
        title: 'Install R Packages',
        label: 'Packages',
        placeholder: 'Enter R packages (comma-separated)',
        message: 'Enter R packages to install, or leave empty:',
        confirmText: 'Create'
      }
    });

    const packagesInput = await firstValueFrom(packagesDialog.afterClosed());
    if (packagesInput === null) return;

    const packages = packagesInput ? packagesInput.split(',').map((p: string) => p.trim()).filter((p: string) => p) : [];

    try {
      this.creatingRenvEnv.set(true);
      await this.wails.createRenvEnvironment(name, packages, '', this.useGlobalCache());
      await this.loadRenvEnvironments();
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to create renv environment: ${error}`);
      this.showError('Failed to create renv environment');
      this.creatingRenvEnv.set(false);
      this.renvCreationProgress.set(null);
    }
  }

  async deleteRenvEnvironment(id: number): Promise<void> {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      disableClose: true,
      data: {
        title: 'Delete Environment',
        message: 'Are you sure you want to delete this renv environment?',
        confirmText: 'Delete',
        cancelText: 'Cancel'
      }
    });

    const confirmed = await firstValueFrom(dialogRef.afterClosed());

    if (!confirmed) {
      return;
    }

    try {
      await this.wails.deleteRenvEnvironment(id);
      await this.loadRenvEnvironments();
    } catch (error) {
      await this.wails.logToFile(`[SettingsR] Failed to delete renv environment: ${error}`);
      this.showError('Failed to delete renv environment');
    }
  }

  getRenvName(env: RenvEnvironment): string {
    return env.Name || 'Unnamed';
  }

  formatDate(timestamp: number): string {
    return new Date(timestamp * 1000).toLocaleDateString();
  }

  getBoundPlugins(renvId: number): string[] {
    return this.bindings()
      .filter(b => b.EnvironmentID === renvId && b.EnvironmentType === 'r')
      .map(b => b.PluginID);
  }

  showBoundPlugins(renvId: number, envName: string): void {
    const pluginIds = this.getBoundPlugins(renvId);
    const boundPlugins: BoundPlugin[] = pluginIds.map(id => {
      const plugin = this.plugins().find(p => 
        p.id.toString() === id || 
        p.definition?.plugin?.id === id
      );
      
      if (plugin) {
        return {
          id: id,
          name: plugin.definition?.plugin?.name || 'Unknown Plugin',
          description: plugin.definition?.plugin?.description || 'No description available'
        };
      }
      
      return {
        id: id,
        name: `Plugin ID: ${id}`,
        description: 'Plugin details not found'
      };
    });

    this.dialog.open(BoundPluginsDialogComponent, {
      data: {
        envName: envName,
        plugins: boundPlugins
      },
      width: '500px',
      disableClose: true
    });
  }
}
