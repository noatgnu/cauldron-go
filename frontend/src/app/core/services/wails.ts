import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import * as WailsApp from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { models, services } from '../../../wailsjs/go/models';

declare global {
  interface Window {
    go: any;
    runtime: any;
  }
}

export type Config = models.Config;
export type Job = models.Job;
export type JobRequest = models.JobRequest;
export type PythonEnvironment = services.PythonEnvironment;
export type REnvironment = services.REnvironment;
export type DataFilePreview = services.DataFilePreview;
export type VirtualEnvironment = services.VirtualEnvironment;

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
  isWails = typeof window !== 'undefined' && !!window.go;

  private jobUpdateSubject = new BehaviorSubject<Job | null>(null);
  jobUpdate$: Observable<Job | null> = this.jobUpdateSubject.asObservable();

  private scriptOutputSubject = new BehaviorSubject<string>('');
  scriptOutput$: Observable<string> = this.scriptOutputSubject.asObservable();

  private downloadProgressSubject = new BehaviorSubject<{message: string, percentage: number} | null>(null);
  downloadProgress$: Observable<{message: string, percentage: number} | null> = this.downloadProgressSubject.asObservable();

  private progressSubject = new BehaviorSubject<ProgressNotification | null>(null);
  progress$: Observable<ProgressNotification | null> = this.progressSubject.asObservable();

  private queueStatusSubject = new BehaviorSubject<any | null>(null);
  queueStatus$: Observable<any | null> = this.queueStatusSubject.asObservable();

  constructor() {
    this.setupEventListeners();
  }

  private setupEventListeners(): void {
    if (!this.isWails) return;

    try {
      EventsOn('job:update', (data: Job) => {
        this.jobUpdateSubject.next(data);
      });

      EventsOn('script:output', (data: string) => {
        this.scriptOutputSubject.next(data);
      });

      EventsOn('download-progress', (data: {message: string, percentage: number}) => {
        this.downloadProgressSubject.next(data);
      });

      EventsOn('progress', (data: ProgressNotification) => {
        this.progressSubject.next(data);
        if (data.type === 'download' || data.type === 'extract' || data.type === 'install') {
          this.downloadProgressSubject.next({
            message: data.message,
            percentage: data.percentage
          });
        }
      });

      EventsOn('queue:status', (data: any) => {
        this.queueStatusSubject.next(data);
      });
    } catch (error) {
      console.error('Failed to setup event listeners:', error);
    }
  }

  async getSettings(): Promise<Config> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetSettings();
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

  async readFilePreview(path: string, limit: number = 10): Promise<string[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReadFilePreview(path, limit);
  }

  async createJob(jobRequest: JobRequest): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.CreateJob(jobRequest);
  }

  async getJob(id: string): Promise<Job> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetJob(id);
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
      return result;
    } catch (error) {
      console.error('[Wails Service] GetAllJobs() failed:', error);
      throw error;
    }
  }

  async deleteJob(id: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteJob(id);
  }

  async reExecuteJob(id: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReExecuteJob(id);
  }

  async rerunJob(jobID: string, useSameEnvironment: boolean, pythonEnvPath: string, rEnvPath: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.RerunJob(jobID, useSameEnvironment, pythonEnvPath, rEnvPath);
  }

  async executePythonScript(scriptName: string, args: string[] = []): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ExecutePythonScript(scriptName, args);
  }

  async executeRScript(scriptName: string, args: string[] = []): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ExecuteRScript(scriptName, args);
  }

  async getPythonVersion(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPythonVersion();
  }

  async getRVersion(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetRVersion();
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

  async openDirectoryInExplorer(path: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenDirectoryInExplorer(path);
  }

  async readJobOutputFile(jobID: string, filename: string): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ReadJobOutputFile(jobID, filename);
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

  async openDataFileDialog(): Promise<string> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.OpenDataFileDialog();
  }

  async parseDataFile(path: string, previewRows: number = 10): Promise<DataFilePreview> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.ParseDataFile(path, previewRows);
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

  async createPythonVirtualEnv(basePythonPath: string, venvPath: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.CreatePythonVirtualEnv(basePythonPath, venvPath);
  }

  async getVirtualEnvironments(): Promise<services.VirtualEnvironment[]> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetVirtualEnvironments();
  }

  async deleteVirtualEnvironment(id: number): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.DeleteVirtualEnvironment(id);
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

  async getPlugin(id: string): Promise<any> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.GetPlugin(id);
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

  async logToFile(message: string): Promise<void> {
    if (!this.isWails) throw new Error('Wails not available');
    return WailsApp.LogToFile(message);
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
}
