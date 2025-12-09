import { TestBed } from '@angular/core/testing';
import { PluginV2Service } from './plugin-v2';
import { models } from '../../../wailsjs/go/models';

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
      const plugins: models.PluginV2[] = [
        {
          definition: {
            plugin: { id: 'p1', name: 'Plugin 1', category: 'analysis' } as any,
            runtime: {} as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2,
        {
          definition: {
            plugin: { id: 'p2', name: 'Plugin 2', category: 'visualization' } as any,
            runtime: {} as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2,
        {
          definition: {
            plugin: { id: 'p3', name: 'Plugin 3', category: 'analysis' } as any,
            runtime: {} as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2
      ];

      const categoryMap = service.getPluginsByCategory(plugins);

      expect(categoryMap.size).toBe(2);
      expect(categoryMap.get('analysis')?.length).toBe(2);
      expect(categoryMap.get('visualization')?.length).toBe(1);
    });
  });

  describe('filterPluginsByRuntime', () => {
    it('should filter plugins by runtime type', () => {
      const plugins: models.PluginV2[] = [
        {
          definition: {
            plugin: {} as any,
            runtime: { type: 'python' } as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2,
        {
          definition: {
            plugin: {} as any,
            runtime: { type: 'r' } as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2
      ];

      const pythonPlugins = service.filterPluginsByRuntime(plugins, 'python');

      expect(pythonPlugins.length).toBe(1);
      expect(pythonPlugins[0].definition.runtime.type).toBe('python');
    });
  });

  describe('searchPlugins', () => {
    it('should search plugins by name, description, and id', () => {
      const plugins: models.PluginV2[] = [
        {
          definition: {
            plugin: { id: 'pca', name: 'PCA Analysis', description: 'Principal Component Analysis' } as any,
            runtime: {} as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2,
        {
          definition: {
            plugin: { id: 'phate', name: 'PHATE Analysis', description: 'Dimensionality reduction' } as any,
            runtime: {} as any,
            inputs: [],
            execution: {} as any
          },
          folderPath: '',
          scriptPath: ''
        } as models.PluginV2
      ];

      const results = service.searchPlugins(plugins, 'pca');

      expect(results.length).toBe(1);
      expect(results[0].definition.plugin.id).toBe('pca');
    });
  });
});
