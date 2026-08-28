import { Component, OnInit, signal, computed, ChangeDetectionStrategy, effect } from '@angular/core';
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
import { MatDialog } from '@angular/material/dialog';
import { Wails, PythonEnvironment, VirtualEnvironment, Config, PluginEnvironmentBinding, UvPythonVersion } from '../../../core/services/wails';
import { NotificationService } from '../../../core/services/notification.service';
import { PackagesModal } from '../../../components/packages-modal/packages-modal';
import { DownloadPortableEnvDialogComponent } from '../../../components/download-portable-env-dialog/download-portable-env-dialog';
import { UvInstallDialog } from '../../../components/uv-install-dialog/uv-install-dialog';
import { BoundPluginsDialogComponent, BoundPlugin } from '../../../components/bound-plugins-dialog/bound-plugins-dialog';

@Component({
  selector: 'app-settings-python',
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
    MatTooltipModule
  ],
  templateUrl: './settings-python.html',
  styleUrl: './settings-python.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SettingsPython implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected pythonVersion = signal('');
  protected pythonEnvironments = signal<PythonEnvironment[]>([]);
  protected basePythonEnvironments = computed(() =>
    this.pythonEnvironments().filter(env => !env.isVirtual)
  );
  protected virtualEnvironments = signal<VirtualEnvironment[]>([]);
  protected detectingPythonEnvs = signal(false);
  protected installingPythonPackages = signal(false);
  protected creatingVenvEnv = signal(false);
  protected selectedPythonEnv = signal<string>('');
  protected pythonInstallProgress = signal<{message: string, percentage: number} | null>(null);
  protected venvCreationProgress = signal<{message: string, percentage: number} | null>(null);
  protected bindings = signal<PluginEnvironmentBinding[]>([]);
  protected plugins = signal<any[]>([]);

  protected uvAvailable = signal(false);
  protected uvPath = signal('');
  protected uvManagedPythons = signal<UvPythonVersion[]>([]);
  protected loadingUvPythons = signal(false);
  protected creatingUvVenv = signal(false);
  protected uvVenvCreationProgress = signal<{message: string, percentage: number} | null>(null);

  constructor(
    private wails: Wails,
    private dialog: MatDialog,
    private notification: NotificationService
  ) {
    effect(() => {
      const progress = this.wails.progress();
      if (!progress) return;

      if (progress.type === 'install' || progress.type === 'download' || progress.type === 'extract') {
        const progressData = {
          message: progress.message,
          percentage: progress.percentage
        };

        const isCompleted = progress.status === 'completed' || progress.status === 'error';

        switch (progress.id) {
          case 'python-requirements':
            this.pythonInstallProgress.set(isCompleted ? null : progressData);
            break;
          case 'python-venv':
            this.venvCreationProgress.set(isCompleted ? null : progressData);
            if (isCompleted) {
              this.creatingVenvEnv.set(false);
            }
            break;
          case 'uv-venv':
            this.uvVenvCreationProgress.set(isCompleted ? null : progressData);
            if (isCompleted) {
              this.creatingUvVenv.set(false);
            }
            break;
          default:
            if (progress.id?.includes('python') && !progress.id?.startsWith('uv-')) {
              this.pythonInstallProgress.set(isCompleted ? null : progressData);
            }
            break;
        }
      }
    });
  }


  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    await this.loadVersion();
    this.detectAllPythonEnvironments();
    await this.loadVirtualEnvironments();
    await this.loadPlugins();
    await this.refreshUvStatus();
  }

  async loadPlugins(): Promise<void> {
    try {
      const plugins = await this.wails.getPluginsV2();
      this.plugins.set(plugins || []);
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to load plugins: ${error}`);
    }
  }


  async loadSettings(): Promise<void> {
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to load settings: ${error}`);
    }
  }

  async loadVersion(): Promise<void> {
    try {
      const pyVersion = await this.wails.getPythonVersion();
      this.pythonVersion.set(pyVersion);
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to load version: ${error}`);
    }
  }

  async browsePython(): Promise<void> {
    try {
      const path = await this.wails.openFile('Select Python Executable');
      if (path) {
        this.config.update(c => ({ ...c, pythonPath: path }));
        await this.saveSetting('pythonPath', path);
        await this.loadVersion();
      }
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to browse for Python: ${error}`);
    }
  }

  private async saveSetting(key: string, value: any): Promise<void> {
    try {
      await this.wails.setSetting(key, value);
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to save setting ${key}: ${error}`);
    }
  }

  async detectAllPythonEnvironments(): Promise<void> {
    this.detectingPythonEnvs.set(true);
    try {
      const envs = await this.wails.detectPythonEnvironments();
      await this.wails.logToFile(`[SettingsPython] Received ${envs?.length || 0} Python environments from backend`);
      if (envs && envs.length > 0) {
        for (let i = 0; i < envs.length; i++) {
          await this.wails.logToFile(`[SettingsPython] [${i}] Name=${envs[i].name}, Path=${envs[i].path}, Type=${envs[i].type}, IsVirtual=${envs[i].isVirtual}`);
        }
      }
      this.pythonEnvironments.set(envs || []);
      await this.wails.logToFile(`[SettingsPython] Signal pythonEnvironments now has ${this.pythonEnvironments().length} items`);

      if (envs && envs.length > 0) {
        const activeEnv = await this.wails.getActivePythonEnvironment();
        if (activeEnv && activeEnv.path) {
          const foundEnv = envs.find(e => e.path === activeEnv.path);
          if (foundEnv) {
            this.selectedPythonEnv.set(foundEnv.path);
          } else {
            this.selectedPythonEnv.set('');
          }
        } else {
          this.selectedPythonEnv.set('');
        }
      } else {
        this.selectedPythonEnv.set('');
      }
    } catch (error) {
      this.pythonEnvironments.set([]);
      this.selectedPythonEnv.set('');
    } finally {
      this.detectingPythonEnvs.set(false);
    }
  }

  async selectPythonEnvironment(envPath: string): Promise<void> {
    this.selectedPythonEnv.set(envPath);
    this.config.update(c => ({ ...c, pythonPath: envPath }));
    await this.saveSetting('pythonPath', envPath);
    await this.wails.setActivePythonEnvironment(envPath);
    await this.loadVersion();
  }

  async installPythonRequirements(): Promise<void> {
    const pythonPath = this.config().pythonPath;
    if (!pythonPath) {
      return;
    }

    try {
      this.installingPythonPackages.set(true);
      const requirementsPath = await this.wails.getBundledRequirementsPath('python');
      await this.wails.installPythonRequirements(pythonPath, requirementsPath);
    } catch (error) {
      this.notification.showError('Failed to install Python packages');
    } finally {
      this.installingPythonPackages.set(false);
    }
  }

  getEnvironmentTypeLabel(type: string): string {
    switch (type) {
      case 'system': return 'System';
      case 'conda': return 'Conda';
      case 'venv': return 'Virtual Env';
      case 'poetry': return 'Poetry';
      case 'portable': return 'Portable';
      default: return type;
    }
  }

  async viewPythonPackages(env: PythonEnvironment): Promise<void> {
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
      const packages = await this.wails.listPythonPackages(env.path);
      dialogRef.componentInstance.setPackages(packages);
      dialogRef.componentInstance.setLoading(false);
    } catch (error) {
      dialogRef.componentInstance.setLoading(false);
      this.notification.showError('Failed to load Python packages', 3000);
    }
  }

  viewSelectedPythonPackages(): void {
    const selectedPath = this.selectedPythonEnv();

    if (!selectedPath) {
      this.notification.showError('No Python environment selected', 3000);
      return;
    }

    const env = this.pythonEnvironments().find(e => e.path === selectedPath);

    if (env) {
      this.viewPythonPackages(env);
    } else {
      this.notification.showError('Python environment not found', 3000);
    }
  }

  isVirtualEnv(path: string): boolean {
    const env = this.pythonEnvironments().find(e => e.path === path);
    return env?.isVirtual ?? false;
  }

  isPortableEnv(path: string): boolean {
    if (!path) return false;
    const env = this.pythonEnvironments().find(e => e.path === path);
    return env?.type === 'portable';
  }

  getSelectedPythonEnv(): PythonEnvironment | undefined {
    return this.pythonEnvironments().find(e => e.path === this.selectedPythonEnv());
  }

  getEnvironmentTypeColor(type: string): string {
    switch (type) {
      case 'portable': return 'accent';
      case 'system': return 'primary';
      case 'venv': return 'warn';
      default: return 'primary';
    }
  }

  canCreateVirtualEnv(): boolean {
    const env = this.getSelectedPythonEnv();
    if (!env) return false;
    return !env.isVirtual;
  }

  getEnvironmentStatusMessage(): string {
    const env = this.getSelectedPythonEnv();
    if (!env) return 'No environment selected';

    switch (env.type) {
      case 'portable':
        return 'Using portable environment - fully self-contained, no additional setup needed';
      case 'system':
        return 'Using system Python - can create virtual environments for isolation';
      case 'venv':
        return 'Using virtual environment - isolated from system packages';
      default:
        return `Using ${env.type} environment`;
    }
  }

  async createVirtualEnv(): Promise<void> {
    const basePythonPath = this.selectedPythonEnv();
    if (!basePythonPath) return;

    try {
      const venvPath = await this.wails.openDirectoryDialog('Select location for virtual environment');
      if (!venvPath) return;

      this.creatingVenvEnv.set(true);
      await this.wails.createPythonVirtualEnv(basePythonPath, venvPath);

      await this.detectAllPythonEnvironments();
      await this.loadVirtualEnvironments();
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to create virtual environment: ${error}`);
      this.notification.showError('Failed to create virtual environment');
    } finally {
      this.creatingVenvEnv.set(false);
      this.venvCreationProgress.set(null);
    }
  }

  async loadVirtualEnvironments(): Promise<void> {
    try {
      const [venvs, bindings] = await Promise.all([
        this.wails.getVirtualEnvironments(),
        this.wails.getAllPluginEnvironmentBindings()
      ]);
      await this.wails.logToFile(`[SettingsPython] loadVirtualEnvironments received ${venvs?.length || 0} venvs`);
      if (venvs && venvs.length > 0) {
        for (let i = 0; i < venvs.length; i++) {
          await this.wails.logToFile(`[SettingsPython] Venv[${i}] ID=${venvs[i].ID}, Name=${venvs[i].Name}, Path=${venvs[i].Path}`);
        }
      }
      this.virtualEnvironments.set(venvs || []);
      this.bindings.set(bindings || []);
      await this.wails.logToFile(`[SettingsPython] Set virtualEnvironments signal with ${venvs?.length || 0} items`);
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to load virtual environments: ${error}`);
    }
  }

  async deleteVirtualEnv(id: number): Promise<void> {
    try {
      await this.wails.deleteVirtualEnvironment(id);
      await this.loadVirtualEnvironments();
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to delete virtual environment: ${error}`);
    }
  }

  getBasePythonName(basePath: string): string {
    const env = this.pythonEnvironments().find(e => e.path === basePath);
    return env ? env.name : basePath.split(/[\\/]/).pop() || basePath;
  }

  formatDate(timestamp: number): string {
    return new Date(timestamp * 1000).toLocaleDateString();
  }

  downloadPythonEnvironment(): void {
    const dialogRef = this.dialog.open(DownloadPortableEnvDialogComponent, {
      width: '600px',
      disableClose: true
    });

    const instance = dialogRef.componentInstance;
    instance.environment = 'python';

    dialogRef.afterClosed().subscribe(() => {
      this.detectAllPythonEnvironments();
      this.loadVersion();
    });
  }

  getBoundPlugins(venvId: number): string[] {
    return this.bindings()
      .filter(b => b.EnvironmentID === venvId && b.EnvironmentType === 'python')
      .map(b => b.PluginID);
  }

  showBoundPlugins(venvId: number, envName: string): void {
    const pluginIds = this.getBoundPlugins(venvId);
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

  async browseVenvStorage(): Promise<void> {
    try {
      const path = await this.wails.openDirectoryDialog('Select Python Virtual Environments Storage Directory');
      if (path) {
        this.config.update(c => ({ ...c, venvStoragePath: path }));
        await this.saveSetting('venvStoragePath', path);
        this.notification.showSuccess('Venv storage path updated');
      }
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to browse for venv storage: ${error}`);
    }
  }

  async clearVenvStorage(): Promise<void> {
    try {
      this.config.update(c => ({ ...c, venvStoragePath: '' }));
      await this.saveSetting('venvStoragePath', '');
      this.notification.showSuccess('Using default venv storage location');
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to clear venv storage: ${error}`);
    }
  }

  async refreshUvStatus(): Promise<void> {
    try {
      const available = await this.wails.isUvAvailable();
      this.uvAvailable.set(available);
      if (available) {
        this.uvPath.set(await this.wails.getUvPath());
        await this.loadUvManagedPythons();
      } else {
        this.uvPath.set('');
        this.uvManagedPythons.set([]);
      }
    } catch (error) {
      this.uvAvailable.set(false);
      await this.wails.logToFile(`[SettingsPython] Failed to check uv status: ${error}`);
    }
  }

  async loadUvManagedPythons(): Promise<void> {
    this.loadingUvPythons.set(true);
    try {
      const pythons = await this.wails.listUvManagedPythons();
      this.uvManagedPythons.set(pythons || []);
    } catch (error) {
      this.uvManagedPythons.set([]);
      await this.wails.logToFile(`[SettingsPython] Failed to list uv-managed Python versions: ${error}`);
    } finally {
      this.loadingUvPythons.set(false);
    }
  }

  openUvInstallDialog(): void {
    const dialogRef = this.dialog.open(UvInstallDialog, {
      width: '600px',
      disableClose: true
    });

    dialogRef.afterClosed().subscribe((changed) => {
      if (changed) {
        this.refreshUvStatus();
      }
    });
  }

  async createUvVirtualEnv(pythonVersion: string): Promise<void> {
    if (this.creatingUvVenv()) return;

    try {
      const venvPath = await this.wails.openDirectoryDialog('Select location for uv virtual environment');
      if (!venvPath) return;

      this.creatingUvVenv.set(true);
      await this.wails.createUvVirtualEnv(pythonVersion, venvPath);

      await this.loadVirtualEnvironments();
    } catch (error) {
      await this.wails.logToFile(`[SettingsPython] Failed to create uv virtual environment: ${error}`);
      this.notification.showError('Failed to create uv virtual environment');
    } finally {
      this.creatingUvVenv.set(false);
      this.uvVenvCreationProgress.set(null);
    }
  }
}
