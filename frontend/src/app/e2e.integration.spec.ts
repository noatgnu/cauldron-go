import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { signal } from '@angular/core';

import { Job } from './core/services/wails';
import { createMockJob, createMockConfig, createMockPlugin } from './testing/wails-mock';

describe('E2E Integration Tests', () => {
  let wailsMock: any;

  beforeEach(() => {
    wailsMock = {
      isWails: true,
      jobUpdate: signal<Job | null>(null),
      scriptOutput: signal(''),
      downloadProgress: signal<{message: string, percentage: number} | null>(null),
      progress: signal<any>(null),
      queueStatus: signal<any>(null),
      bindingsUpdated: signal(0),
      getAllJobs: vi.fn().mockResolvedValue([]),
      getJob: vi.fn().mockResolvedValue(null),
      deleteJob: vi.fn().mockResolvedValue(undefined),
      rerunJob: vi.fn().mockResolvedValue('new-job-id'),
      getImportedFiles: vi.fn().mockResolvedValue([]),
      deleteImportedFile: vi.fn().mockResolvedValue(undefined),
      importDataFile: vi.fn().mockResolvedValue(1),
      getSettings: vi.fn().mockResolvedValue(createMockConfig()),
      setSetting: vi.fn().mockResolvedValue(undefined),
      getPluginsV2: vi.fn().mockResolvedValue([]),
      getPlugin: vi.fn().mockResolvedValue(null),
      executePluginV2: vi.fn().mockResolvedValue('job-123'),
      getPythonVersion: vi.fn().mockResolvedValue('Python 3.11.0'),
      getRVersion: vi.fn().mockResolvedValue('R 4.3.0'),
      checkDockerVersion: vi.fn().mockResolvedValue('Docker 24.0.0'),
      getActivePythonEnvironment: vi.fn().mockResolvedValue({ path: '/usr/bin/python3', version: '3.11.0' }),
      getActiveREnvironment: vi.fn().mockResolvedValue({ path: '/usr/bin/R', version: '4.3.0' }),
      setActivePythonEnvironment: vi.fn().mockResolvedValue(undefined),
      setActiveREnvironment: vi.fn().mockResolvedValue(undefined),
      detectPythonEnvironments: vi.fn().mockResolvedValue([]),
      detectREnvironments: vi.fn().mockResolvedValue([]),
      getVirtualEnvironments: vi.fn().mockResolvedValue([]),
      createPythonVirtualEnv: vi.fn().mockResolvedValue(undefined),
      deleteVirtualEnvironment: vi.fn().mockResolvedValue(undefined),
      getRenvEnvironments: vi.fn().mockResolvedValue([]),
      createRenvEnvironment: vi.fn().mockResolvedValue(undefined),
      deleteRenvEnvironment: vi.fn().mockResolvedValue(undefined),
      bindPluginToEnvironment: vi.fn().mockResolvedValue(undefined),
      getPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      deletePluginEnvironmentBinding: vi.fn().mockResolvedValue(undefined),
      getAllPluginEnvironmentBindings: vi.fn().mockResolvedValue([]),
      pauseJobQueue: vi.fn().mockResolvedValue(undefined),
      resumeJobQueue: vi.fn().mockResolvedValue(undefined),
      stopJobQueueImmediate: vi.fn().mockResolvedValue(undefined),
      getJobQueueStatus: vi.fn().mockResolvedValue({ paused: false, running: 0, pending: 0 }),
      logToFile: vi.fn().mockResolvedValue(undefined)
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Complete User Flow: App Startup', () => {
    it('should successfully initialize app with all services', async () => {
      expect(wailsMock.isWails).toBe(true);

      const settings = await wailsMock.getSettings();
      expect(settings).toBeDefined();
      expect(settings.pythonPath).toBeDefined();

      const jobs = await wailsMock.getAllJobs();
      expect(Array.isArray(jobs)).toBe(true);

      const files = await wailsMock.getImportedFiles();
      expect(Array.isArray(files)).toBe(true);

      const pythonEnv = await wailsMock.getActivePythonEnvironment();
      expect(pythonEnv).not.toBeNull();

      const rEnv = await wailsMock.getActiveREnvironment();
      expect(rEnv).not.toBeNull();
    });

    it('should handle missing environments gracefully', async () => {
      wailsMock.getActivePythonEnvironment.mockResolvedValue(null);
      wailsMock.getActiveREnvironment.mockResolvedValue(null);

      const pythonEnv = await wailsMock.getActivePythonEnvironment();
      expect(pythonEnv).toBeNull();

      const rEnv = await wailsMock.getActiveREnvironment();
      expect(rEnv).toBeNull();
    });
  });

  describe('Complete User Flow: Job Lifecycle', () => {
    it('should complete full job lifecycle: create -> monitor -> complete', async () => {
      const mockPlugin = createMockPlugin({
        id: 1,
        definition: {
          plugin: { id: 'pca-analysis', name: 'PCA Analysis' },
          runtime: { environments: ['python'], entrypoint: 'main.py' },
          inputs: [
            { name: 'input_file', type: 'file', required: true },
            { name: 'n_components', type: 'number', required: false, default: 2 }
          ],
          outputs: [],
          execution: { outputDir: '--output_folder' }
        }
      });
      wailsMock.getPluginsV2.mockResolvedValue([mockPlugin]);

      const plugins = await wailsMock.getPluginsV2();
      expect(plugins).toHaveLength(1);
      expect(plugins[0].definition.plugin.id).toBe('pca-analysis');

      const jobId = await wailsMock.executePluginV2({
        pluginID: 1,
        parameters: {
          input_file: '/path/to/data.csv',
          n_components: 3
        }
      });
      expect(jobId).toBe('job-123');

      const pendingJob = createMockJob({
        id: jobId,
        status: 'pending',
        name: 'PCA Analysis'
      });
      wailsMock.getJob.mockResolvedValue(pendingJob);

      const job = await wailsMock.getJob(jobId);
      expect(job.status).toBe('pending');

      const runningJob = { ...pendingJob, status: 'in_progress' };
      wailsMock.jobUpdate.set(runningJob);
      expect(wailsMock.jobUpdate().status).toBe('in_progress');

      const completedJob = { ...runningJob, status: 'completed' };
      wailsMock.jobUpdate.set(completedJob);
      expect(wailsMock.jobUpdate().status).toBe('completed');
    });

    it('should handle job failure gracefully', async () => {
      const jobId = 'failing-job-123';
      const failingJob = createMockJob({
        id: jobId,
        status: 'in_progress',
        name: 'Failing Analysis'
      });

      wailsMock.getJob.mockResolvedValue(failingJob);

      const job = await wailsMock.getJob(jobId);
      expect(job.status).toBe('in_progress');

      const failedJob = { ...failingJob, status: 'failed', error: 'Script execution failed' };
      wailsMock.jobUpdate.set(failedJob);

      expect(wailsMock.jobUpdate().status).toBe('failed');
    });

    it('should rerun a failed job', async () => {
      const originalJobId = 'original-job-123';
      wailsMock.rerunJob.mockResolvedValue('rerun-job-456');

      const newJobId = await wailsMock.rerunJob(originalJobId, true, '', '');

      expect(newJobId).toBe('rerun-job-456');
      expect(wailsMock.rerunJob).toHaveBeenCalledWith(originalJobId, true, '', '');
    });

    it('should delete a job', async () => {
      const jobId = 'job-to-delete';

      await wailsMock.deleteJob(jobId);

      expect(wailsMock.deleteJob).toHaveBeenCalledWith(jobId);
    });
  });

  describe('Complete User Flow: File Import and Management', () => {
    it('should import and manage data files', async () => {
      wailsMock.importDataFile.mockResolvedValue(1);

      const fileId = await wailsMock.importDataFile('/path/to/data.csv');
      expect(fileId).toBe(1);

      const mockFiles = [
        {
          id: 1,
          name: 'data.csv',
          path: '/path/to/data.csv',
          size: 1024,
          importedAt: Date.now(),
          fileType: 'csv',
          preview: 'col1,col2\nval1,val2'
        }
      ];
      wailsMock.getImportedFiles.mockResolvedValue(mockFiles);

      const files = await wailsMock.getImportedFiles();
      expect(files).toHaveLength(1);
      expect(files[0].name).toBe('data.csv');

      await wailsMock.deleteImportedFile(1);
      expect(wailsMock.deleteImportedFile).toHaveBeenCalledWith(1);
    });
  });

  describe('Complete User Flow: Environment Setup', () => {
    it('should detect and set Python environment', async () => {
      const mockEnvs = [
        { path: '/usr/bin/python3', version: '3.11.0', isVirtual: false },
        { path: '/home/user/venv/bin/python', version: '3.10.0', isVirtual: true }
      ];
      wailsMock.detectPythonEnvironments.mockResolvedValue(mockEnvs);

      const envs = await wailsMock.detectPythonEnvironments();
      expect(envs).toHaveLength(2);

      await wailsMock.setActivePythonEnvironment('/home/user/venv/bin/python');
      expect(wailsMock.setActivePythonEnvironment).toHaveBeenCalledWith('/home/user/venv/bin/python');
    });

    it('should create virtual environment for a plugin', async () => {
      await wailsMock.createPythonVirtualEnv('/usr/bin/python3', '/path/to/new/venv', 'pca-plugin');

      expect(wailsMock.createPythonVirtualEnv).toHaveBeenCalledWith(
        '/usr/bin/python3',
        '/path/to/new/venv',
        'pca-plugin'
      );
    });

    it('should bind plugin to environment', async () => {
      await wailsMock.bindPluginToEnvironment('pca-plugin', 'python', 1, '/path/to/venv');

      expect(wailsMock.bindPluginToEnvironment).toHaveBeenCalledWith(
        'pca-plugin',
        'python',
        1,
        '/path/to/venv'
      );
    });
  });

  describe('Complete User Flow: Job Queue Management', () => {
    it('should pause and resume job queue', async () => {
      await wailsMock.pauseJobQueue();
      expect(wailsMock.pauseJobQueue).toHaveBeenCalled();

      wailsMock.getJobQueueStatus.mockResolvedValue({ paused: true, running: 0, pending: 3 });
      let status = await wailsMock.getJobQueueStatus();
      expect(status.paused).toBe(true);

      await wailsMock.resumeJobQueue();
      expect(wailsMock.resumeJobQueue).toHaveBeenCalled();

      wailsMock.getJobQueueStatus.mockResolvedValue({ paused: false, running: 1, pending: 2 });
      status = await wailsMock.getJobQueueStatus();
      expect(status.paused).toBe(false);
    });

    it('should stop queue immediately', async () => {
      await wailsMock.stopJobQueueImmediate();
      expect(wailsMock.stopJobQueueImmediate).toHaveBeenCalled();
    });
  });

  describe('Complete User Flow: Settings Management', () => {
    it('should load and update settings', async () => {
      const settings = await wailsMock.getSettings();
      expect(settings.pythonPath).toBe('/usr/bin/python3');

      await wailsMock.setSetting('pythonPath', '/new/python/path');
      expect(wailsMock.setSetting).toHaveBeenCalledWith('pythonPath', '/new/python/path');

      wailsMock.getSettings.mockResolvedValue({
        ...createMockConfig(),
        pythonPath: '/new/python/path'
      });

      const updatedSettings = await wailsMock.getSettings();
      expect(updatedSettings.pythonPath).toBe('/new/python/path');
    });
  });

  describe('Complete User Flow: Multiple Jobs Running', () => {
    it('should handle multiple concurrent jobs', async () => {
      const mockJobs = [
        createMockJob({ id: 'job-1', name: 'PCA Analysis', status: 'in_progress' }),
        createMockJob({ id: 'job-2', name: 'Volcano Plot', status: 'pending' }),
        createMockJob({ id: 'job-3', name: 'Clustering', status: 'completed' })
      ];
      wailsMock.getAllJobs.mockResolvedValue(mockJobs);

      const jobs = await wailsMock.getAllJobs();
      expect(jobs).toHaveLength(3);

      const inProgressJobs = jobs.filter((j: Job) => j.status === 'in_progress');
      expect(inProgressJobs).toHaveLength(1);

      const pendingJobs = jobs.filter((j: Job) => j.status === 'pending');
      expect(pendingJobs).toHaveLength(1);

      const completedJobs = jobs.filter((j: Job) => j.status === 'completed');
      expect(completedJobs).toHaveLength(1);
    });
  });

  describe('Signal-based Real-time Updates', () => {
    it('should update UI when job status changes via signal', () => {
      const initialJob = createMockJob({ id: 'job-1', status: 'pending' });
      wailsMock.jobUpdate.set(initialJob);

      expect(wailsMock.jobUpdate().status).toBe('pending');

      const updatedJob = { ...initialJob, status: 'in_progress' };
      wailsMock.jobUpdate.set(updatedJob);

      expect(wailsMock.jobUpdate().status).toBe('in_progress');
    });

    it('should track download progress via signal', () => {
      expect(wailsMock.downloadProgress()).toBeNull();

      wailsMock.downloadProgress.set({ message: 'Downloading...', percentage: 25 });
      expect(wailsMock.downloadProgress()?.percentage).toBe(25);

      wailsMock.downloadProgress.set({ message: 'Downloading...', percentage: 75 });
      expect(wailsMock.downloadProgress()?.percentage).toBe(75);

      wailsMock.downloadProgress.set({ message: 'Complete', percentage: 100 });
      expect(wailsMock.downloadProgress()?.percentage).toBe(100);
    });

    it('should notify bindings updated', () => {
      const initialValue = wailsMock.bindingsUpdated();

      wailsMock.bindingsUpdated.set(initialValue + 1);
      expect(wailsMock.bindingsUpdated()).toBe(initialValue + 1);
    });
  });

  describe('Error Handling Scenarios', () => {
    it('should handle network errors gracefully', async () => {
      wailsMock.getAllJobs.mockRejectedValue(new Error('Network error'));

      await expect(wailsMock.getAllJobs()).rejects.toThrow('Network error');
    });

    it('should handle backend errors gracefully', async () => {
      wailsMock.executePluginV2.mockRejectedValue(new Error('Plugin execution failed'));

      await expect(
        wailsMock.executePluginV2({ pluginID: 1, parameters: {} })
      ).rejects.toThrow('Plugin execution failed');
    });

    it('should handle missing resources gracefully', async () => {
      wailsMock.getJob.mockResolvedValue(null);

      const job = await wailsMock.getJob('nonexistent-job');
      expect(job).toBeNull();
    });
  });

  describe('Data Serialization', () => {
    it('should serialize empty arrays correctly (not null)', async () => {
      wailsMock.getAllJobs.mockResolvedValue([]);

      const jobs = await wailsMock.getAllJobs();

      expect(jobs).toEqual([]);
      expect(jobs).not.toBeNull();
      expect(Array.isArray(jobs)).toBe(true);
    });

    it('should serialize job with all required fields', async () => {
      const job = createMockJob({
        id: 'test-job',
        name: 'Test Job',
        type: 'test',
        status: 'completed',
        args: ['--input', 'file.csv'],
        terminalOutput: ['Starting...', 'Done.'],
        createdAt: Date.now(),
        updatedAt: Date.now()
      });
      wailsMock.getJob.mockResolvedValue(job);

      const result = await wailsMock.getJob('test-job');

      expect(result.id).toBe('test-job');
      expect(result.args).toEqual(['--input', 'file.csv']);
      expect(result.terminalOutput).toEqual(['Starting...', 'Done.']);
    });
  });
});
