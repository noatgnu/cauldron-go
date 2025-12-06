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
import { MatDialog } from '@angular/material/dialog';
import { Wails, PythonEnvironment, REnvironment, VirtualEnvironment, Config } from '../../core/services/wails';
import { PackagesModal } from '../../components/packages-modal/packages-modal';
import { DownloadPortableEnvDialogComponent } from '../../components/download-portable-env-dialog/download-portable-env-dialog';

@Component({
  selector: 'app-settings',
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
  templateUrl: './settings.html',
  styleUrl: './settings.scss',
})
export class Settings implements OnInit {
  protected config = signal<Partial<Config>>({});
  protected loading = signal(false);
  protected pythonVersion = signal('');
  protected rVersion = signal('');
  protected pythonEnvironments = signal<PythonEnvironment[]>([]);
  protected rEnvironments = signal<REnvironment[]>([]);
  protected virtualEnvironments = signal<VirtualEnvironment[]>([]);
  protected detectingPythonEnvs = signal(false);
  protected detectingREnvs = signal(false);
  protected installingPackages = signal(false);
  protected selectedPythonEnv = signal<string>('');
  protected selectedREnv = signal<string>('');
  protected pythonInstallProgress = signal<{message: string, percentage: number} | null>(null);
  protected rInstallProgress = signal<{message: string, percentage: number} | null>(null);

  constructor(
    private wails: Wails,
    private dialog: MatDialog
  ) {}

  async ngOnInit(): Promise<void> {
    await this.loadSettings();
    await this.loadVersions();
    this.detectAllPythonEnvironments();
    this.detectAllREnvironments();
    await this.loadVirtualEnvironments();
    this.setupProgressUpdates();
  }

  setupProgressUpdates(): void {
    this.wails.progress$.subscribe(progress => {
      if (!progress) return;

      if (progress.type === 'install' || progress.type === 'download' || progress.type === 'extract') {
        const isPython = progress.id?.includes('python');
        const isR = progress.id?.includes('r-portable');

        if (progress.status === 'completed' || progress.status === 'error') {
          if (isPython) {
            this.pythonInstallProgress.set(null);
          } else if (isR) {
            this.rInstallProgress.set(null);
          }
        } else {
          const progressData = {
            message: progress.message,
            percentage: progress.percentage
          };

          if (isPython) {
            this.pythonInstallProgress.set(progressData);
          } else if (isR) {
            this.rInstallProgress.set(progressData);
          }
        }
      }
    });
  }

  async loadSettings(): Promise<void> {
    this.loading.set(true);
    try {
      const config = await this.wails.getSettings();
      this.config.set(config);
    } catch (error) {
      console.error('Failed to load settings:', error);
    } finally {
      this.loading.set(false);
    }
  }

  async loadVersions(): Promise<void> {
    try {
      const [pyVersion, rVer] = await Promise.all([
        this.wails.getPythonVersion(),
        this.wails.getRVersion()
      ]);
      this.pythonVersion.set(pyVersion);
      this.rVersion.set(rVer);
    } catch (error) {
      console.error('Failed to load versions:', error);
    }
  }

  async detectPython(): Promise<void> {
    try {
      const path = await this.wails.detectPythonPath();
      this.config.update(c => ({ ...c, pythonPath: path }));
      await this.saveSetting('pythonPath', path);
      await this.loadVersions();
    } catch (error) {
      console.error('Failed to detect Python:', error);
    }
  }

  async detectR(): Promise<void> {
    try {
      const path = await this.wails.detectRPath();
      this.config.update(c => ({ ...c, rPath: path }));
      await this.saveSetting('rPath', path);
      await this.loadVersions();
    } catch (error) {
      console.error('Failed to detect R:', error);
    }
  }

  async browsePython(): Promise<void> {
    try {
      const path = await this.wails.openFile('Select Python Executable');
      if (path) {
        this.config.update(c => ({ ...c, pythonPath: path }));
        await this.saveSetting('pythonPath', path);
        await this.loadVersions();
      }
    } catch (error) {
      console.error('Failed to browse for Python:', error);
    }
  }

  async browseR(): Promise<void> {
    try {
      const path = await this.wails.openFile('Select R Executable');
      if (path) {
        this.config.update(c => ({ ...c, rPath: path }));
        await this.saveSetting('rPath', path);
        await this.loadVersions();
      }
    } catch (error) {
      console.error('Failed to browse for R:', error);
    }
  }

  async browseOutputDirectory(): Promise<void> {
    try {
      const path = await this.wails.openDirectoryDialog('Select Output Directory');
      if (path) {
        this.config.update(c => ({ ...c, outputDirectory: path }));
        await this.saveSetting('outputDirectory', path);
      }
    } catch (error) {
      console.error('Failed to browse for output directory:', error);
    }
  }

  private async saveSetting(key: string, value: any): Promise<void> {
    try {
      await this.wails.setSetting(key, value);
    } catch (error) {
      console.error('Failed to save setting:', error);
    }
  }

  async detectAllPythonEnvironments(): Promise<void> {
    this.detectingPythonEnvs.set(true);
    try {
      const envs = await this.wails.detectPythonEnvironments();
      this.pythonEnvironments.set(envs || []);

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

  async selectPythonEnvironment(envPath: string): Promise<void> {
    this.selectedPythonEnv.set(envPath);
    this.config.update(c => ({ ...c, pythonPath: envPath }));
    await this.saveSetting('pythonPath', envPath);
    await this.wails.setActivePythonEnvironment(envPath);
    await this.loadVersions();
  }

  async selectREnvironment(envPath: string): Promise<void> {
    this.selectedREnv.set(envPath);
    this.config.update(c => ({ ...c, rPath: envPath }));
    await this.saveSetting('rPath', envPath);
    await this.wails.setActiveREnvironment(envPath);
    await this.loadVersions();
  }

  async installPythonRequirements(): Promise<void> {
    const pythonPath = this.config().pythonPath;
    if (!pythonPath) {
      console.error('No Python environment selected');
      return;
    }

    try {
      this.installingPackages.set(true);
      const requirementsPath = await this.wails.getBundledRequirementsPath('python');
      await this.wails.installPythonRequirements(pythonPath, requirementsPath);
      console.log('Requirements installed successfully');
    } catch (error) {
      console.error('Failed to install requirements:', error);
    } finally {
      this.installingPackages.set(false);
    }
  }

  async installRPackages(): Promise<void> {
    const rPath = this.config().rPath;
    if (!rPath) {
      console.error('No R environment selected');
      return;
    }

    try {
      this.installingPackages.set(true);
      const requirementsPath = await this.wails.getBundledRequirementsPath('r');
      const packages = await this.wails.loadRPackagesFromFile(requirementsPath);
      await this.wails.installRPackages(rPath, packages);
      console.log('R packages installed successfully via BiocManager');
    } catch (error) {
      console.error('Failed to install R packages:', error);
    } finally {
      this.installingPackages.set(false);
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
      console.error('Failed to load Python packages:', error);
      dialogRef.componentInstance.setLoading(false);
    }
  }

  async viewRPackages(env: REnvironment): Promise<void> {
    const dialogRef = this.dialog.open(PackagesModal, {
      width: '600px',
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
      console.error('Failed to load R packages:', error);
      dialogRef.componentInstance.setLoading(false);
    }
  }

  viewSelectedPythonPackages(): void {
    const env = this.pythonEnvironments().find(e => e.path === this.selectedPythonEnv());
    if (env) {
      this.viewPythonPackages(env);
    }
  }

  viewSelectedRPackages(): void {
    const env = this.rEnvironments().find(e => e.path === this.selectedREnv());
    if (env) {
      this.viewRPackages(env);
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
    return !env.isVirtual && env.type !== 'portable';
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

      console.log('Creating virtual environment at:', venvPath);
      await this.wails.createPythonVirtualEnv(basePythonPath, venvPath);

      await this.detectAllPythonEnvironments();
      await this.loadVirtualEnvironments();
    } catch (error) {
      console.error('Failed to create virtual environment:', error);
    }
  }

  async loadVirtualEnvironments(): Promise<void> {
    try {
      const venvs = await this.wails.getVirtualEnvironments();
      this.virtualEnvironments.set(venvs || []);
    } catch (error) {
      console.error('Failed to load virtual environments:', error);
    }
  }

  async deleteVirtualEnv(id: number): Promise<void> {
    try {
      await this.wails.deleteVirtualEnvironment(id);
      await this.loadVirtualEnvironments();
    } catch (error) {
      console.error('Failed to delete virtual environment:', error);
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
      this.loadVersions();
    });
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
      this.loadVersions();
    });
  }
}
