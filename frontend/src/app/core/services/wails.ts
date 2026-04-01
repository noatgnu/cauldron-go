import { Injectable, signal, Signal } from '@angular/core';
import { Subscription } from 'rxjs';
import * as WailsApp from '../../../../bindings/github.com/noatgnu/cauldron-go/app';
import { Events } from '@wailsio/runtime';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import * as services from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/services/models';

declare global {
  interface Window {
    go: any;
    runtime: any;
  }
}

export type Config = models.Config;
export type Job = models.Job;
export type PythonEnvironment = services.PythonEnvironment;
export type REnvironment = services.REnvironment;
export type DataFilePreview = services.DataFilePreview;
export type VirtualEnvironment = services.VirtualEnvironment;
export type RenvEnvironment = services.RenvEnvironment;
export type PluginEnvironmentBinding = services.PluginEnvironmentBinding;
export type CustomEnvVar = services.CustomEnvVar;

export interface GitAuthConfig {
  id: number;
  repositoryURL: string;
  sshKeyPath: string;
  hasPassphrase: boolean;
  createdAt: number;
  updatedAt: number;
}

export interface ImportedFile {
  id: number;
  name: string;
  path: string;
  size: number;
  importedAt: number;
  fileType: string;
  preview: string;
}

export type ProgressType = 'download' | 'install' | 'script' | 'extract' | 'analysis' | 'generic';

export interface ProgressNotification {
  type: ProgressType;
  id: string;
  message: string;
  percentage: number;
  status: 'started' | 'in_progress' | 'completed' | 'error';
  data?: Record<string, any>;
}

@Injectable({
  providedIn: 'root'
})
export class Wails {
  isWails = typeof window !== 'undefined' && '_wails' in window;

  private _jobUpdate = signal<Job | null>(null);
  jobUpdate: Signal<Job | null> = this._jobUpdate.asReadonly();

  private _scriptOutput = signal<string>('');
  scriptOutput: Signal<string> = this._scriptOutput.asReadonly();

  private _downloadProgress = signal<{message: string, percentage: number} | null>(null);
  downloadProgress: Signal<{message: string, percentage: number} | null> = this._downloadProgress.asReadonly();

  private _progress = signal<ProgressNotification | null>(null);
  progress: Signal<ProgressNotification | null> = this._progress.asReadonly();

  private _queueStatus = signal<any | null>(null);
  queueStatus: Signal<any | null> = this._queueStatus.asReadonly();

  private _bindingsUpdated = signal<number>(0);
  bindingsUpdated: Signal<number> = this._bindingsUpdated.asReadonly();

  constructor() {
    this.setupEventListeners();
  }

  notifyBindingsUpdated(): void {
    this._bindingsUpdated.update(n => n + 1);
  }

  private setupEventListeners(): void {
    if (!this.isWails) return;

    try {
      Events.On('job:update', (ev: any) => {
        this._jobUpdate.set(ev.data as Job);
      });

      Events.On('script:output', (ev: any) => {
        this._scriptOutput.set(ev.data as string);
      });

      Events.On('download-progress', (ev: any) => {
        this._downloadProgress.set(ev.data as {message: string, percentage: number});
      });

      Events.On('progress', (ev: any) => {
        const data = ev.data as ProgressNotification;
        this._progress.set(data);
        if (data.type === 'download' || data.type === 'extract' || data.type === 'install') {
          this._downloadProgress.set({
            message: data.message,
            percentage: data.percentage
          });
        }
      });

      Events.On('queue:status', (ev: any) => {
        this._queueStatus.set(ev.data);
      });
    } catch (error) {
      console.error('Failed to setup event listeners:', error);
    }
  }

  async getSettings(): Promise<Config> {
    if (!this.isWails) throw new Error('Wails not available');
    const result = await WailsApp.GetSettings();
    if (!result) throw new Error('Settings not found');
    return result;
  }

  async setSetting(key: string, value: any): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SetSetting(key, value);
  }

  async detectPythonPath(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DetectPythonPath();
  }

  async detectRPath(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DetectRPath();
  }

  async openFile(title: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenFile(title);
  }

  async openFileDialog(title: string): Promise<string> {
    return this.openFile(title);
  }

  async openDirectoryDialog(title: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenDirectory(title);
  }

  async saveFileDialog(title: string, defaultName: string = ''): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SaveFile(title, defaultName);
  }

  async readFile(path: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    const content = await WailsApp.ReadFile(path);
    if (typeof content === 'string') {
      try {
        const binaryString = atob(content);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          bytes[i] = binaryString.charCodeAt(i);
        }
        return new TextDecoder().decode(bytes);
      } catch (e) {
        return content;
      }
    }
    return new TextDecoder().decode(new Uint8Array(content));
  }

  async readFileAsUint8Array(path: string): Promise<Uint8Array> {
    if (!this.isWails) throw new Error('Wails not available');
    const content = await WailsApp.ReadFile(path);
    if (typeof content === 'string') {
      try {
        const binaryString = atob(content);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
      } catch (e) {
        return new TextEncoder().encode(content);
      }
    }
    return new Uint8Array(content);
  }

  async readFilePreview(path: string, limit: number = 10): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReadFilePreview(path, limit);
  }

  async getJob(id: string): Promise<Job> {
    if (!this.isWails) throw new Error('Wails not available');
    const result = await WailsApp.GetJob(id);
    if (!result) throw new Error(`Job not found: ${id}`);
    return result;
  }

  async getJobExecutionLog(id: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetJobExecutionLog(id);
  }

  async getAllJobs(): Promise<Job[]> {
    console.log('[Wails Service] getAllJobs() called, isWails:', this.isWails);
    if (!this.isWails) {
      console.error('[Wails Service] Wails not available!');
      throw new Error('Wails not available');
    }

    try {
      console.log('[Wails Service] Calling WailsApp.GetAllJobs()...');
      const result = await WailsApp.GetAllJobs();
      console.log('[Wails Service] GetAllJobs() returned:', result);
      return (result || []).filter((job): job is Job => job !== null);
    } catch (error) {
      console.error('[Wails Service] GetAllJobs() failed:', error);
      throw error;
    }
  }

  async deleteJob(id: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteJob(id);
  }

  async rerunJob(jobID: string, useSameEnvironment: boolean, pythonEnvPath: string, rEnvPath: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.RerunJob(jobID, useSameEnvironment, pythonEnvPath, rEnvPath);
  }

  async getPythonVersion(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPythonVersion();
  }

  async getRVersion(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetRVersion();
  }

  async checkDockerVersion(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.CheckDockerVersion();
  }

  async greet(name: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.Greet(name);
  }

  async detectPythonEnvironments(): Promise<PythonEnvironment[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DetectPythonEnvironments();
  }

  async detectREnvironments(): Promise<REnvironment[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DetectREnvironments();
  }

  async getActivePythonEnvironment(): Promise<PythonEnvironment | null> {
    if (!this.isWails) throw new Error('Wails not available');
    try {
      return await WailsApp.GetActivePythonEnvironment();
    } catch (error) {
      return null;
    }
  }

  async getActiveREnvironment(): Promise<REnvironment | null> {
    if (!this.isWails) throw new Error('Wails not available');
    try {
      return await WailsApp.GetActiveREnvironment();
    } catch (error) {
      return null;
    }
  }

  async setActivePythonEnvironment(path: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SetActivePythonEnvironment(path);
  }

  async setActiveREnvironment(path: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SetActiveREnvironment(path);
  }

  async getLicenseInfo(): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetLicenseInfo();
  }

  async openDirectoryInExplorer(path: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenDirectoryInExplorer(path);
  }

  async readJobOutputFile(jobID: string, filename: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReadJobOutputFile(jobID, filename);
  }

  async listJobOutputFiles(jobID: string): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ListJobOutputFiles(jobID);
  }

  async writeJobOutputFile(jobID: string, filename: string, content: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.WriteJobOutputFile(jobID, filename, content);
  }

  async installPythonPackages(pythonPath: string, packages: string[]): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallPythonPackages(pythonPath, packages);
  }

  async installPythonRequirements(pythonPath: string, requirementsPath: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallPythonRequirements(pythonPath, requirementsPath);
  }

  async installRPackages(rPath: string, packages: string[]): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallRPackages(rPath, packages);
  }

  async listPythonPackages(pythonPath: string): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ListPythonPackages(pythonPath);
  }

  async listRPackages(rPath: string): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ListRPackages(rPath);
  }

  async getBundledRequirementsPath(requirementType: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetBundledRequirementsPath(requirementType);
  }

  async loadRPackagesFromFile(filePath: string): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.LoadRPackagesFromFile(filePath);
  }

  async getExampleFilePath(exampleType: string, fileName: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetExampleFilePath(exampleType, fileName);
  }

  async getPluginExampleFilePath(pluginID: string, filePath: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginExampleFilePath(pluginID, filePath);
  }

  async openDataFileDialog(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenDataFileDialog();
  }

  async parseDataFile(path: string, previewRows: number = 10): Promise<DataFilePreview> {
    if (!this.isWails) throw new Error('Wails not available');
    const result = await WailsApp.ParseDataFile(path, previewRows);
    if (!result) throw new Error('Failed to parse data file');
    return result;
  }

  async importDataFile(path: string): Promise<number> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ImportDataFile(path);
  }

  async getImportedFiles(): Promise<ImportedFile[]> {
    if (!this.isWails) throw new Error('Wails not available');
    const files = await WailsApp.GetImportedFiles();
    return files.map((f: services.ImportedFile) => ({
      id: f.ID,
      name: f.Name,
      path: f.Path,
      size: f.Size,
      importedAt: f.ImportedAt,
      fileType: f.FileType,
      preview: f.Preview
    }));
  }

  async deleteImportedFile(id: number): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteImportedFile(id);
  }

  async createPythonVirtualEnv(basePythonPath: string, venvPath: string, pluginID: string = ''): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.CreatePythonVirtualEnv(basePythonPath, venvPath, pluginID);
    this.notifyBindingsUpdated();
  }

  async getDefaultVenvPath(pluginID: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetDefaultVenvPath(pluginID);
  }

  async getVirtualEnvironments(): Promise<services.VirtualEnvironment[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetVirtualEnvironments();
  }

  async deleteVirtualEnvironment(id: number): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.DeleteVirtualEnvironment(id);
    this.notifyBindingsUpdated();
  }

  async createRenvEnvironment(name: string, packages: string[], pluginID: string, useCache: boolean = false) {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.CreateRenvEnvironment(name, packages, pluginID, useCache);
    this.notifyBindingsUpdated();
  }

  async getRenvEnvironments(): Promise<services.RenvEnvironment[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetRenvEnvironments();
  }

  async deleteRenvEnvironment(id: number): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.DeleteRenvEnvironment(id);
    this.notifyBindingsUpdated();
  }

  async bindPluginToEnvironment(pluginID: string, envType: string, envID: number, envPath: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.BindPluginToEnvironment(pluginID, envType, envID, envPath);
    this.notifyBindingsUpdated();
  }

  async getPluginEnvironmentBinding(pluginID: string, envType: string): Promise<services.PluginEnvironmentBinding | null> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginEnvironmentBinding(pluginID, envType);
  }

  async deletePluginEnvironmentBinding(pluginID: string, envType: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    await WailsApp.DeletePluginEnvironmentBinding(pluginID, envType);
    this.notifyBindingsUpdated();
  }

  async getAllPluginEnvironmentBindings(): Promise<services.PluginEnvironmentBinding[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetAllPluginEnvironmentBindings();
  }

  async getPortableEnvironmentURL(platform: string, arch: string, version: string, environment: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPortableEnvironmentURL(platform, arch, version, environment);
  }

  async downloadPortableEnvironment(url: string, environment: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DownloadPortableEnvironment(url, environment);
  }

  async getPortableEnvironmentPath(environment: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPortableEnvironmentPath(environment);
  }

  async getPlugins(): Promise<any[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPlugins();
  }

  async getPluginsV2(): Promise<any[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginsV2();
  }

  async getPlugin(id: number): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginV2(id);
  }

  async getCustomEnvVars(pluginId: number): Promise<CustomEnvVar[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetCustomEnvVars(pluginId);
  }

  async getGlobalCustomEnvVars(): Promise<CustomEnvVar[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetGlobalCustomEnvVars();
  }

  async saveCustomEnvVar(envVar: CustomEnvVar): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SaveCustomEnvVar(envVar);
  }

  async deleteCustomEnvVar(id: number): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteCustomEnvVar(id);
  }

  async deleteCustomEnvVarByKey(pluginId: number, key: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteCustomEnvVarByKey(pluginId, key);
  }

  async saveGitAuthConfig(repoURL: string, sshKeyPath: string, passphrase: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SaveGitAuthConfig(repoURL, sshKeyPath, passphrase);
  }

  async getGitAuthConfig(repoURL: string): Promise<GitAuthConfig | null> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetGitAuthConfig(repoURL);
  }

  async getAllGitAuthConfigs(): Promise<GitAuthConfig[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetAllGitAuthConfigs();
  }

  async deleteGitAuthConfig(repoURL: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteGitAuthConfig(repoURL);
  }

  async validateSSHKey(keyPath: string, passphrase: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ValidateSSHKey(keyPath, passphrase);
  }

  async reloadPlugins(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReloadPlugins();
  }

  async getPluginsDirectory(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginsDirectory();
  }

  async createSamplePlugin(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.CreateSamplePlugin();
  }

  async executePlugin(request: any): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ExecutePlugin(request);
  }

  async executePluginV2(request: any): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ExecutePluginV2(request);
  }

  async logToFile(message: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.LogToFile(message);
  }

  async saveTempFile(filename: string, content: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SaveTempFile(filename, content);
  }

  async savePluginYAML(pluginID: string, yamlContent: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SavePluginYAML(pluginID, yamlContent);
  }

  async validatePluginYAML(yamlContent: string): Promise<{valid: boolean, errors: string[]}> {
    if (!this.isWails) throw new Error('Wails not available');
    const result: any = await WailsApp.ValidatePluginYAML(yamlContent);
    return { valid: result[0] as boolean, errors: (result[1] as string[]) || [] };
  }

  async convertPluginToYAML(definition: any): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ConvertPluginToYAML(definition);
  }

  async parsePluginYAML(yamlContent: string): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ParsePluginYAML(yamlContent);
  }

  async getPluginTemplates(): Promise<any[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginTemplates();
  }

  async deletePlugin(pluginID: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeletePlugin(pluginID);
  }

  async installPluginFromRepo(repoURL: string, commitHash: string = ''): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallPluginFromRepo(repoURL, commitHash);
  }

  async updatePluginFromRepo(repoURL: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UpdatePluginFromRepo(repoURL);
  }

  async updatePluginToCommit(repoURL: string, commitHash: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UpdatePluginToCommit(repoURL, commitHash);
  }

  async updatePluginToCommitForce(repoURL: string, commitHash: string, force: boolean): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UpdatePluginToCommitForce(repoURL, commitHash, force);
  }

  async updatePluginFromRepoForce(repoURL: string, force: boolean): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UpdatePluginFromRepoForce(repoURL, force);
  }

  async reinstallPlugin(repoURL: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReinstallPlugin(repoURL);
  }

  async updateAllRemotePlugins(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UpdateAllRemotePlugins();
  }

  async forceUpdateAllRemotePlugins(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ForceUpdateAllRemotePlugins();
  }

  async getRemotePlugins(): Promise<models.PluginRegistry[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetRemotePlugins();
  }

  async forceUpdateRemotePlugin(pluginID: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ForceUpdateRemotePlugin(pluginID);
  }

  async uninstallPluginFromRepo(repoURL: string, removeGitAuth: boolean, deleteJobHistory: boolean, deleteEnvironments: boolean): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UninstallPluginFromRepo(repoURL, removeGitAuth, deleteJobHistory, deleteEnvironments);
  }

  async getPluginJobCount(pluginID: string): Promise<number> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginJobCount(pluginID);
  }

  async getPluginEnvironmentCount(pluginID: string): Promise<number> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginEnvironmentCount(pluginID);
  }

  async isPluginInstalled(repoURL: string): Promise<boolean> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.IsPluginInstalled(repoURL);
  }

  async getPluginVersion(repoURL: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginVersion(repoURL);
  }

  async decodePluginRepoURL(encoded: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DecodePluginRepoURL(encoded);
  }

  async confirmPluginInstallation(repoURL: string, commitHash: string = ''): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ConfirmPluginInstallation(repoURL, commitHash);
  }

  listen(eventName: string, callback: (data: any) => void): Subscription {
    if (this.isWails) {
      Events.On(eventName, callback);
    }
    return new Subscription();
  }

  async pauseJobQueue(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.PauseJobQueue();
  }

  async stopJobQueueImmediate(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.StopJobQueueImmediate();
  }

  async resumeJobQueue(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ResumeJobQueue();
  }

  async getJobQueueStatus(): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetJobQueueStatus();
  }

  async processPendingJobs(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ProcessPendingJobs();
  }

  async openLogFile(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenLogFile();
  }

  async openLogDirectory(): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenLogDirectory();
  }

  async getLogFilePath(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetLogFilePath();
  }

  async listRegistryPlugins(searchQuery: string, categoryName: string, authorName: string, limit: number, offset: number): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ListRegistryPlugins(searchQuery, categoryName, authorName, limit, offset);
  }

  async getRegistryPlugin(id: string): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetRegistryPlugin(id);
  }

  async listRegistryCategories(): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ListRegistryCategories();
  }

  async installPluginFromRegistry(pluginID: string, commitHash: string = ''): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallPluginFromRegistry(pluginID, commitHash);
  }

  async checkPluginUpdate(repoURL: string, currentCommit: string, registrySource: string | null): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.CheckPluginUpdate(repoURL, currentCommit, registrySource);
  }

  async setPluginUpdatePolicy(repoURL: string, policy: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.SetPluginUpdatePolicy(repoURL, policy);
  }

  async pinPluginVersion(repoURL: string, version: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.PinPluginVersion(repoURL, version);
  }

  async unpinPluginVersion(repoURL: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.UnpinPluginVersion(repoURL);
  }

  async getPluginRequirements(pluginId: string): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPluginRequirements(pluginId);
  }

  async installPluginRequirements(pluginId: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.InstallPluginRequirements(pluginId);
  }

  async fetchPluginDependencies(repoURL: string): Promise<Record<string, any>> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.FetchPluginDependencies(repoURL);
  }
}
