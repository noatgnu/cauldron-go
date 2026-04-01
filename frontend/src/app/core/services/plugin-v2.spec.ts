import { TestBed } from '@angular/core/testing';
import { PluginV2Service } from './plugin-v2';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';

describe('PluginV2Service', () => {
  let service: PluginV2Service;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(PluginV2Service);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('getPluginsByCategory', () => {
    it('should group plugins by category', () => {
      const plugins = [
        {
          id: 1,
          definition: {
            plugin: { id: 'p1', name: 'Plugin 1', category: 'analysis' },
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        },
        {
          id: 2,
          definition: {
            plugin: { id: 'p2', name: 'Plugin 2', category: 'visualization' },
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        },
        {
          id: 3,
          definition: {
            plugin: { id: 'p3', name: 'Plugin 3', category: 'analysis' },
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        }
      ] as unknown as models.PluginV2[];

      const categoryMap = service.getPluginsByCategory(plugins);

      expect(categoryMap.size).toBe(2);
      expect(categoryMap.get('analysis')?.length).toBe(2);
      expect(categoryMap.get('visualization')?.length).toBe(1);
    });
  });

  describe('filterPluginsByRuntime', () => {
    it('should filter plugins by runtime type', () => {
      const plugins = [
        {
          id: 1,
          definition: {
            plugin: {},
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        },
        {
          id: 2,
          definition: {
            plugin: {},
            runtime: { environments: ['r'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        }
      ] as unknown as models.PluginV2[];

      const pythonPlugins = service.filterPluginsByRuntime(plugins, 'python');

      expect(pythonPlugins.length).toBe(1);
      expect(pythonPlugins[0].definition.runtime.environments[0]).toBe('python');
    });
  });

  describe('searchPlugins', () => {
    it('should search plugins by name, description, and id', () => {
      const plugins = [
        {
          id: 1,
          definition: {
            plugin: { id: 'pca', name: 'PCA Analysis', description: 'Principal Component Analysis' },
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        },
        {
          id: 2,
          definition: {
            plugin: { id: 'phate', name: 'PHATE Analysis', description: 'Dimensionality reduction' },
            runtime: { environments: ['python'], entrypoint: 'script' },
            inputs: [],
            execution: {}
          },
          folderPath: '',
          scriptPath: '',
          installSource: 'builtin',
          commitHash: '',
          repository: '',
          enabled: true
        }
      ] as unknown as models.PluginV2[];

      const results = service.searchPlugins(plugins, 'pca');

      expect(results.length).toBe(1);
      expect(results[0].definition.plugin.id).toBe('pca');
    });
  });
});
