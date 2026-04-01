import { vi } from 'vitest';

export interface MockWailsApp {
  GetSettings: ReturnType<typeof vi.fn>;
  SetSetting: ReturnType<typeof vi.fn>;
  GetAllJobs: ReturnType<typeof vi.fn>;
  GetJob: ReturnType<typeof vi.fn>;
  DeleteJob: ReturnType<typeof vi.fn>;
  GetImportedFiles: ReturnType<typeof vi.fn>;
  DeleteImportedFile: ReturnType<typeof vi.fn>;
  ImportDataFile: ReturnType<typeof vi.fn>;
  GetPluginsV2: ReturnType<typeof vi.fn>;
  GetPluginV2: ReturnType<typeof vi.fn>;
  ExecutePluginV2: ReturnType<typeof vi.fn>;
  GetPythonVersion: ReturnType<typeof vi.fn>;
  GetRVersion: ReturnType<typeof vi.fn>;
  CheckDockerVersion: ReturnType<typeof vi.fn>;
  DetectPythonEnvironments: ReturnType<typeof vi.fn>;
  DetectREnvironments: ReturnType<typeof vi.fn>;
  GetActivePythonEnvironment: ReturnType<typeof vi.fn>;
  GetActiveREnvironment: ReturnType<typeof vi.fn>;
  SetActivePythonEnvironment: ReturnType<typeof vi.fn>;
  SetActiveREnvironment: ReturnType<typeof vi.fn>;
  OpenFile: ReturnType<typeof vi.fn>;
  OpenDirectory: ReturnType<typeof vi.fn>;
  SaveFile: ReturnType<typeof vi.fn>;
  ReadFile: ReturnType<typeof vi.fn>;
  ReadFilePreview: ReturnType<typeof vi.fn>;
  LogToFile: ReturnType<typeof vi.fn>;
  RerunJob: ReturnType<typeof vi.fn>;
  GetJobExecutionLog: ReturnType<typeof vi.fn>;
  ListJobOutputFiles: ReturnType<typeof vi.fn>;
  ReadJobOutputFile: ReturnType<typeof vi.fn>;
  WriteJobOutputFile: ReturnType<typeof vi.fn>;
  GetVirtualEnvironments: ReturnType<typeof vi.fn>;
  CreatePythonVirtualEnv: ReturnType<typeof vi.fn>;
  DeleteVirtualEnvironment: ReturnType<typeof vi.fn>;
  GetRenvEnvironments: ReturnType<typeof vi.fn>;
  CreateRenvEnvironment: ReturnType<typeof vi.fn>;
  DeleteRenvEnvironment: ReturnType<typeof vi.fn>;
  BindPluginToEnvironment: ReturnType<typeof vi.fn>;
  GetPluginEnvironmentBinding: ReturnType<typeof vi.fn>;
  DeletePluginEnvironmentBinding: ReturnType<typeof vi.fn>;
  GetAllPluginEnvironmentBindings: ReturnType<typeof vi.fn>;
  InstallPluginFromRepo: ReturnType<typeof vi.fn>;
  DeletePlugin: ReturnType<typeof vi.fn>;
  ReloadPlugins: ReturnType<typeof vi.fn>;
  GetPluginsDirectory: ReturnType<typeof vi.fn>;
  SavePluginYAML: ReturnType<typeof vi.fn>;
  ValidatePluginYAML: ReturnType<typeof vi.fn>;
  ConvertPluginToYAML: ReturnType<typeof vi.fn>;
  ParsePluginYAML: ReturnType<typeof vi.fn>;
  ListRegistryPlugins: ReturnType<typeof vi.fn>;
  GetRegistryPlugin: ReturnType<typeof vi.fn>;
  InstallPluginFromRegistry: ReturnType<typeof vi.fn>;
  PauseJobQueue: ReturnType<typeof vi.fn>;
  ResumeJobQueue: ReturnType<typeof vi.fn>;
  StopJobQueueImmediate: ReturnType<typeof vi.fn>;
  GetJobQueueStatus: ReturnType<typeof vi.fn>;
}

export function createMockWailsApp(): MockWailsApp {
  return {
    GetSettings: vi.fn().mockResolvedValue({
      pythonPath: '/usr/bin/python3',
      rPath: '/usr/bin/R',
      resultStoragePath: '/tmp/results',
      logLevel: 'info'
    }),
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
    InstallPluginFromRepo: vi.fn().mockResolvedValue({}),
    DeletePlugin: vi.fn().mockResolvedValue(undefined),
    ReloadPlugins: vi.fn().mockResolvedValue(undefined),
    GetPluginsDirectory: vi.fn().mockResolvedValue('/plugins'),
    SavePluginYAML: vi.fn().mockResolvedValue(undefined),
    ValidatePluginYAML: vi.fn().mockResolvedValue([true, []]),
    ConvertPluginToYAML: vi.fn().mockResolvedValue('plugin:\n  id: test'),
    ParsePluginYAML: vi.fn().mockResolvedValue({}),
    ListRegistryPlugins: vi.fn().mockResolvedValue({ plugins: [], total: 0 }),
    GetRegistryPlugin: vi.fn().mockResolvedValue(null),
    InstallPluginFromRegistry: vi.fn().mockResolvedValue(undefined),
    PauseJobQueue: vi.fn().mockResolvedValue(undefined),
    ResumeJobQueue: vi.fn().mockResolvedValue(undefined),
    StopJobQueueImmediate: vi.fn().mockResolvedValue(undefined),
    GetJobQueueStatus: vi.fn().mockResolvedValue({ paused: false, running: 0, pending: 0 })
  };
}

export function createMockJob(overrides: Partial<any> = {}): any {
  return {
    id: 'job-001',
    name: 'Test Job',
    type: 'test',
    status: 'completed',
    args: [],
    terminalOutput: [],
    createdAt: Date.now(),
    updatedAt: Date.now(),
    workingDirectory: '/tmp/job-001',
    ...overrides
  };
}

export function createMockPlugin(overrides: Partial<any> = {}): any {
  return {
    id: 1,
    definition: {
      plugin: {
        id: 'test-plugin',
        name: 'Test Plugin',
        description: 'A test plugin',
        version: '1.0.0',
        author: 'Test Author',
        category: 'analysis' as any
      },
      runtime: {
        environments: ['python'],
        entrypoint: 'main.py'
      },
      inputs: [],
      outputs: [],
      execution: {
        outputDir: '--output_folder'
      }
    },
    ...overrides
  };
}

export function createMockImportedFile(overrides: Partial<any> = {}): any {
  return {
    ID: 1,
    Name: 'test.csv',
    Path: '/tmp/test.csv',
    Size: 1024,
    ImportedAt: Date.now(),
    FileType: 'csv',
    Preview: 'col1,col2\nval1,val2',
    ...overrides
  };
}

export function createMockConfig(overrides: Partial<any> = {}): any {
  return {
    pythonPath: '/usr/bin/python3',
    rPath: '/usr/bin/R',
    resultStoragePath: '/tmp/results',
    logLevel: 'info',
    theme: 'system',
    ...overrides
  };
}
