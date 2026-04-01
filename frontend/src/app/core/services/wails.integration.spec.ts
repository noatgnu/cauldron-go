import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createMockJob, createMockPlugin, createMockImportedFile, createMockConfig } from '../../testing/wails-mock';

describe('Wails Service Integration Tests', () => {
  let mockWailsApp: any;

  beforeEach(() => {
    mockWailsApp = {
      GetSettings: vi.fn().mockResolvedValue(createMockConfig()),
      SetSetting: vi.fn().mockResolvedValue(undefined),
      GetAllJobs: vi.fn().mockResolvedValue([]),
      GetJob: vi.fn().mockResolvedValue(null),
      DeleteJob: vi.fn().mockResolvedValue(undefined),
      GetImportedFiles: vi.fn().mockResolvedValue([]),
      DeleteImportedFile: vi.fn().mockResolvedValue(undefined),
      ImportDataFile: vi.fn().mockResolvedValue(1),
      GetPluginsV2: vi.fn().mockResolvedValue([]),
      GetPluginV2: vi.fn().mockResolvedValue(null),
      ExecutePluginV2: vi.fn().mockResolvedValue('job-123'),
      GetPythonVersion: vi.fn().mockResolvedValue('3.11.0'),
      GetRVersion: vi.fn().mockResolvedValue('4.3.0'),
      CheckDockerVersion: vi.fn().mockResolvedValue('24.0.0'),
      DetectPythonEnvironments: vi.fn().mockResolvedValue([]),
      DetectREnvironments: vi.fn().mockResolvedValue([]),
      GetActivePythonEnvironment: vi.fn().mockResolvedValue(null),
      GetActiveREnvironment: vi.fn().mockResolvedValue(null),
      SetActivePythonEnvironment: vi.fn().mockResolvedValue(undefined),
      SetActiveREnvironment: vi.fn().mockResolvedValue(undefined),
      OpenFile: vi.fn().mockResolvedValue('/path/to/file.txt'),
      OpenDirectory: vi.fn().mockResolvedValue('/path/to/directory'),
      SaveFile: vi.fn().mockResolvedValue('/path/to/saved.txt'),
      ReadFile: vi.fn().mockResolvedValue('file content'),
      ReadFilePreview: vi.fn().mockResolvedValue(['line1', 'line2']),
      LogToFile: vi.fn().mockResolvedValue(undefined),
      RerunJob: vi.fn().mockResolvedValue('new-job-123'),
      GetJobExecutionLog: vi.fn().mockResolvedValue(''),
      ListJobOutputFiles: vi.fn().mockResolvedValue([]),
      ReadJobOutputFile: vi.fn().mockResolvedValue(''),
      WriteJobOutputFile: vi.fn().mockResolvedValue(undefined),
      GetVirtualEnvironments: vi.fn().mockResolvedValue([]),
      CreatePythonVirtualEnv: vi.fn().mockResolvedValue(undefined),
      DeleteVirtualEnvironment: vi.fn().mockResolvedValue(undefined),
      GetRenvEnvironments: vi.fn().mockResolvedValue([]),
      CreateRenvEnvironment: vi.fn().mockResolvedValue(undefined),
      DeleteRenvEnvironment: vi.fn().mockResolvedValue(undefined),
      BindPluginToEnvironment: vi.fn().mockResolvedValue(undefined),
      GetPluginEnvironmentBinding: vi.fn().mockResolvedValue(null),
      DeletePluginEnvironmentBinding: vi.fn().mockResolvedValue(undefined),
      GetAllPluginEnvironmentBindings: vi.fn().mockResolvedValue([]),
      PauseJobQueue: vi.fn().mockResolvedValue(undefined),
      ResumeJobQueue: vi.fn().mockResolvedValue(undefined),
      StopJobQueueImmediate: vi.fn().mockResolvedValue(undefined),
      GetJobQueueStatus: vi.fn().mockResolvedValue({ paused: false, running: 0, pending: 0 }),
      SavePluginYAML: vi.fn().mockResolvedValue(undefined),
      ValidatePluginYAML: vi.fn().mockResolvedValue([true, []]),
      ConvertPluginToYAML: vi.fn().mockResolvedValue('plugin:\n  id: test'),
      ParsePluginYAML: vi.fn().mockResolvedValue({}),
      ListRegistryPlugins: vi.fn().mockResolvedValue({ plugins: [], total: 0 }),
      GetRegistryPlugin: vi.fn().mockResolvedValue(null),
      InstallPluginFromRegistry: vi.fn().mockResolvedValue(undefined)
    };
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('Job Management API', () => {
    it('should get all jobs successfully', async () => {
      const mockJobs = [
        createMockJob({ id: 'job-1', name: 'Job 1', status: 'completed' }),
        createMockJob({ id: 'job-2', name: 'Job 2', status: 'in_progress' })
      ];
      mockWailsApp.GetAllJobs.mockResolvedValue(mockJobs);

      const jobs = await mockWailsApp.GetAllJobs();

      expect(jobs).toHaveLength(2);
      expect(jobs[0].id).toBe('job-1');
      expect(jobs[1].id).toBe('job-2');
    });

    it('should handle empty jobs list', async () => {
      mockWailsApp.GetAllJobs.mockResolvedValue([]);

      const jobs = await mockWailsApp.GetAllJobs();

      expect(jobs).toEqual([]);
    });

    it('should get a specific job by ID', async () => {
      const mockJob = createMockJob({ id: 'job-123', name: 'Specific Job' });
      mockWailsApp.GetJob.mockResolvedValue(mockJob);

      const job = await mockWailsApp.GetJob('job-123');

      expect(job.id).toBe('job-123');
      expect(job.name).toBe('Specific Job');
    });

    it('should return null when job not found', async () => {
      mockWailsApp.GetJob.mockResolvedValue(null);

      const job = await mockWailsApp.GetJob('nonexistent');

      expect(job).toBeNull();
    });

    it('should delete a job successfully', async () => {
      mockWailsApp.DeleteJob.mockResolvedValue(undefined);

      await expect(mockWailsApp.DeleteJob('job-123')).resolves.toBeUndefined();
    });

    it('should rerun a job and return new job ID', async () => {
      mockWailsApp.RerunJob.mockResolvedValue('new-job-456');

      const newJobId = await mockWailsApp.RerunJob('job-123', true, '', '');

      expect(newJobId).toBe('new-job-456');
    });
  });

  describe('Settings API', () => {
    it('should get settings successfully', async () => {
      const mockConfig = createMockConfig();
      mockWailsApp.GetSettings.mockResolvedValue(mockConfig);

      const config = await mockWailsApp.GetSettings();

      expect(config.pythonPath).toBe('/usr/bin/python3');
      expect(config.resultStoragePath).toBe('/tmp/results');
    });

    it('should set a setting value', async () => {
      mockWailsApp.SetSetting.mockResolvedValue(undefined);

      await expect(mockWailsApp.SetSetting('pythonPath', '/new/python')).resolves.toBeUndefined();
    });
  });

  describe('File Operations API', () => {
    it('should get imported files', async () => {
      const mockFiles = [
        createMockImportedFile({ ID: 1, Name: 'file1.csv' }),
        createMockImportedFile({ ID: 2, Name: 'file2.tsv' })
      ];
      mockWailsApp.GetImportedFiles.mockResolvedValue(mockFiles);

      const files = await mockWailsApp.GetImportedFiles();

      expect(files).toHaveLength(2);
      expect(files[0].Name).toBe('file1.csv');
      expect(files[1].Name).toBe('file2.tsv');
    });

    it('should delete an imported file', async () => {
      mockWailsApp.DeleteImportedFile.mockResolvedValue(undefined);

      await expect(mockWailsApp.DeleteImportedFile(1)).resolves.toBeUndefined();
    });

    it('should import a data file and return ID', async () => {
      mockWailsApp.ImportDataFile.mockResolvedValue(42);

      const fileId = await mockWailsApp.ImportDataFile('/path/to/file.csv');

      expect(fileId).toBe(42);
    });

    it('should read file preview', async () => {
      mockWailsApp.ReadFilePreview.mockResolvedValue(['line1', 'line2', 'line3']);

      const preview = await mockWailsApp.ReadFilePreview('/path/to/file.csv', 3);

      expect(preview).toEqual(['line1', 'line2', 'line3']);
    });
  });

  describe('Plugin Management API', () => {
    it('should get all plugins V2', async () => {
      const mockPlugins = [
        createMockPlugin({ id: 1 }),
        createMockPlugin({ id: 2 })
      ];
      mockWailsApp.GetPluginsV2.mockResolvedValue(mockPlugins);

      const plugins = await mockWailsApp.GetPluginsV2();

      expect(plugins).toHaveLength(2);
    });

    it('should get a specific plugin', async () => {
      const mockPlugin = createMockPlugin({ id: 5 });
      mockWailsApp.GetPluginV2.mockResolvedValue(mockPlugin);

      const plugin = await mockWailsApp.GetPluginV2(5);

      expect(plugin.id).toBe(5);
    });

    it('should execute a plugin V2', async () => {
      mockWailsApp.ExecutePluginV2.mockResolvedValue('job-xyz-123');

      const jobId = await mockWailsApp.ExecutePluginV2({ pluginID: 1, parameters: {} });

      expect(jobId).toBe('job-xyz-123');
    });
  });

  describe('Environment Management API', () => {
    it('should get Python version', async () => {
      mockWailsApp.GetPythonVersion.mockResolvedValue('Python 3.11.4');

      const version = await mockWailsApp.GetPythonVersion();

      expect(version).toBe('Python 3.11.4');
    });

    it('should get R version', async () => {
      mockWailsApp.GetRVersion.mockResolvedValue('R 4.3.1');

      const version = await mockWailsApp.GetRVersion();

      expect(version).toBe('R 4.3.1');
    });

    it('should get Docker version', async () => {
      mockWailsApp.CheckDockerVersion.mockResolvedValue('Docker 24.0.5');

      const version = await mockWailsApp.CheckDockerVersion();

      expect(version).toBe('Docker 24.0.5');
    });

    it('should detect Python environments', async () => {
      const mockEnvs = [
        { path: '/usr/bin/python3', version: '3.11.0', isVirtual: false },
        { path: '/home/user/venv/bin/python', version: '3.10.0', isVirtual: true }
      ];
      mockWailsApp.DetectPythonEnvironments.mockResolvedValue(mockEnvs);

      const envs = await mockWailsApp.DetectPythonEnvironments();

      expect(envs).toHaveLength(2);
    });

    it('should set active Python environment', async () => {
      mockWailsApp.SetActivePythonEnvironment.mockResolvedValue(undefined);

      await expect(mockWailsApp.SetActivePythonEnvironment('/new/python')).resolves.toBeUndefined();
    });

    it('should create Python virtual environment', async () => {
      mockWailsApp.CreatePythonVirtualEnv.mockResolvedValue(undefined);

      await mockWailsApp.CreatePythonVirtualEnv('/usr/bin/python3', '/path/to/venv', 'plugin-1');

      expect(mockWailsApp.CreatePythonVirtualEnv).toHaveBeenCalled();
    });

    it('should get virtual environments', async () => {
      const mockVenvs = [
        { id: 1, path: '/path/venv1', pythonVersion: '3.11.0' },
        { id: 2, path: '/path/venv2', pythonVersion: '3.10.0' }
      ];
      mockWailsApp.GetVirtualEnvironments.mockResolvedValue(mockVenvs);

      const venvs = await mockWailsApp.GetVirtualEnvironments();

      expect(venvs).toHaveLength(2);
    });
  });

  describe('Job Queue Management API', () => {
    it('should pause job queue', async () => {
      mockWailsApp.PauseJobQueue.mockResolvedValue(undefined);

      await expect(mockWailsApp.PauseJobQueue()).resolves.toBeUndefined();
    });

    it('should resume job queue', async () => {
      mockWailsApp.ResumeJobQueue.mockResolvedValue(undefined);

      await expect(mockWailsApp.ResumeJobQueue()).resolves.toBeUndefined();
    });

    it('should stop job queue immediately', async () => {
      mockWailsApp.StopJobQueueImmediate.mockResolvedValue(undefined);

      await expect(mockWailsApp.StopJobQueueImmediate()).resolves.toBeUndefined();
    });

    it('should get job queue status', async () => {
      const mockStatus = { paused: false, running: 2, pending: 5 };
      mockWailsApp.GetJobQueueStatus.mockResolvedValue(mockStatus);

      const status = await mockWailsApp.GetJobQueueStatus();

      expect(status.paused).toBe(false);
      expect(status.running).toBe(2);
      expect(status.pending).toBe(5);
    });
  });

  describe('Dialog Operations API', () => {
    it('should open file dialog', async () => {
      mockWailsApp.OpenFile.mockResolvedValue('/selected/file.txt');

      const filePath = await mockWailsApp.OpenFile('Select a file');

      expect(filePath).toBe('/selected/file.txt');
    });

    it('should open directory dialog', async () => {
      mockWailsApp.OpenDirectory.mockResolvedValue('/selected/directory');

      const dirPath = await mockWailsApp.OpenDirectory('Select a directory');

      expect(dirPath).toBe('/selected/directory');
    });

    it('should open save file dialog', async () => {
      mockWailsApp.SaveFile.mockResolvedValue('/saved/file.txt');

      const filePath = await mockWailsApp.SaveFile('Save file', 'output.txt');

      expect(filePath).toBe('/saved/file.txt');
    });
  });

  describe('Plugin YAML Operations', () => {
    it('should save plugin YAML', async () => {
      mockWailsApp.SavePluginYAML.mockResolvedValue(undefined);

      await expect(mockWailsApp.SavePluginYAML('test-plugin', 'plugin:\n  id: test')).resolves.toBeUndefined();
    });

    it('should validate plugin YAML', async () => {
      mockWailsApp.ValidatePluginYAML.mockResolvedValue([true, []]);

      const result = await mockWailsApp.ValidatePluginYAML('plugin:\n  id: test');

      expect(result[0]).toBe(true);
      expect(result[1]).toEqual([]);
    });

    it('should return validation errors', async () => {
      mockWailsApp.ValidatePluginYAML.mockResolvedValue([false, ['Missing required field: name']]);

      const result = await mockWailsApp.ValidatePluginYAML('plugin:\n  id: test');

      expect(result[0]).toBe(false);
      expect(result[1]).toContain('Missing required field: name');
    });

    it('should convert plugin definition to YAML', async () => {
      mockWailsApp.ConvertPluginToYAML.mockResolvedValue('plugin:\n  id: test\n  name: Test Plugin');

      const yaml = await mockWailsApp.ConvertPluginToYAML({ plugin: { id: 'test', name: 'Test Plugin' } });

      expect(yaml).toContain('plugin:');
      expect(yaml).toContain('id: test');
    });
  });

  describe('Registry Operations', () => {
    it('should list registry plugins', async () => {
      const mockResult = {
        plugins: [
          { id: 'plugin-1', name: 'Plugin 1' },
          { id: 'plugin-2', name: 'Plugin 2' }
        ],
        total: 2
      };
      mockWailsApp.ListRegistryPlugins.mockResolvedValue(mockResult);

      const result = await mockWailsApp.ListRegistryPlugins('', '', '', 10, 0);

      expect(result.plugins).toHaveLength(2);
      expect(result.total).toBe(2);
    });

    it('should get registry plugin details', async () => {
      const mockPlugin = { id: 'plugin-1', name: 'Plugin 1', description: 'A plugin' };
      mockWailsApp.GetRegistryPlugin.mockResolvedValue(mockPlugin);

      const plugin = await mockWailsApp.GetRegistryPlugin('plugin-1');

      expect(plugin.name).toBe('Plugin 1');
    });

    it('should install plugin from registry', async () => {
      mockWailsApp.InstallPluginFromRegistry.mockResolvedValue(undefined);

      await expect(mockWailsApp.InstallPluginFromRegistry('plugin-1')).resolves.toBeUndefined();
    });
  });

  describe('Error Handling', () => {
    it('should handle API errors gracefully', async () => {
      mockWailsApp.GetAllJobs.mockRejectedValue(new Error('Backend error'));

      await expect(mockWailsApp.GetAllJobs()).rejects.toThrow('Backend error');
    });
  });
});
