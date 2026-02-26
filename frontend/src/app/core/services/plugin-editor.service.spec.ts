import { TestBed } from '@angular/core/testing';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { PluginEditorService } from './plugin-editor.service';
import { Wails } from './wails';
import { models } from '../../../wailsjs/go/models';

describe('PluginEditorService', () => {
  let service: PluginEditorService;
  let wailsMock: any;

  beforeEach(() => {
    wailsMock = {
      savePluginYAML: vi.fn(),
      validatePluginYAML: vi.fn(),
      deletePlugin: vi.fn(),
      convertPluginToYAML: vi.fn(),
      parsePluginYAML: vi.fn(),
      getPluginTemplates: vi.fn(),
      logToFile: vi.fn()
    };

    TestBed.configureTestingModule({
      providers: [
        PluginEditorService,
        { provide: Wails, useValue: wailsMock }
      ]
    });

    service = TestBed.inject(PluginEditorService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  describe('savePlugin', () => {
    it('should convert definition to YAML and save', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test-plugin',
          name: 'Test Plugin',
          description: 'Test description',
          version: '1.0.0',
          category: 'analysis'
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'script',
          script: 'test.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      const mockYaml = 'plugin:\n  id: test-plugin\n  name: Test Plugin';
      wailsMock.convertPluginToYAML.mockResolvedValue(mockYaml);
      wailsMock.savePluginYAML.mockResolvedValue(undefined);
      wailsMock.logToFile.mockResolvedValue(undefined);

      await service.savePlugin('test-plugin', mockDefinition);

      expect(wailsMock.convertPluginToYAML).toHaveBeenCalledWith(mockDefinition);
      expect(wailsMock.savePluginYAML).toHaveBeenCalledWith('test-plugin', mockYaml);
      expect(wailsMock.logToFile).toHaveBeenCalled();
    });
  });

  describe('validatePlugin', () => {
    it('should validate plugin definition successfully', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test-plugin',
          name: 'Test Plugin',
          description: 'Test description',
          version: '1.0.0',
          category: 'analysis'
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'script',
          script: 'test.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      const mockYaml = 'plugin:\n  id: test-plugin';
      wailsMock.convertPluginToYAML.mockResolvedValue(mockYaml);
      wailsMock.validatePluginYAML.mockResolvedValue({
        valid: true,
        errors: []
      });

      const result = await service.validatePlugin(mockDefinition);

      expect(result.valid).toBe(true);
      expect(result.errors).toEqual([]);
      expect(wailsMock.validatePluginYAML).toHaveBeenCalledWith(mockYaml);
    });

    it('should return validation errors', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: '',
          name: '',
          description: '',
          version: '',
          category: ''
        }),
        runtime: new models.PluginRuntimeV2({
          environments: [],
          entrypoint: '',
          script: ''
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: ''
        })
      });

      const mockYaml = 'plugin:\n  id: ""';
      wailsMock.convertPluginToYAML.mockResolvedValue(mockYaml);
      wailsMock.validatePluginYAML.mockResolvedValue({
        valid: false,
        errors: ['Plugin ID is required', 'Plugin name is required']
      });

      const result = await service.validatePlugin(mockDefinition);

      expect(result.valid).toBe(false);
      expect(result.errors.length).toBe(2);
      expect(result.errors).toContain('Plugin ID is required');
    });

    it('should handle validation errors', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test',
          name: 'Test',
          description: 'Test',
          version: '1.0.0',
          category: 'analysis'
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'script',
          script: 'test.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      wailsMock.convertPluginToYAML.mockRejectedValue('Conversion error');

      const result = await service.validatePlugin(mockDefinition);

      expect(result.valid).toBe(false);
      expect(result.errors[0]).toContain('Validation error');
    });
  });

  describe('deletePlugin', () => {
    it('should delete plugin and log action', async () => {
      wailsMock.deletePlugin.mockResolvedValue(undefined);
      wailsMock.logToFile.mockResolvedValue(undefined);

      await service.deletePlugin('test-plugin');

      expect(wailsMock.deletePlugin).toHaveBeenCalledWith('test-plugin');
      expect(wailsMock.logToFile).toHaveBeenCalled();
    });
  });

  describe('YAML conversion', () => {
    it('should convert definition to YAML', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test',
          name: 'Test',
          description: 'Test',
          version: '1.0.0',
          category: 'analysis'
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'script',
          script: 'test.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      const mockYaml = 'plugin:\n  id: test';
      wailsMock.convertPluginToYAML.mockResolvedValue(mockYaml);

      const result = await service.definitionToYAML(mockDefinition);

      expect(result).toBe(mockYaml);
      expect(wailsMock.convertPluginToYAML).toHaveBeenCalledWith(mockDefinition);
    });

    it('should parse YAML to definition', async () => {
      const mockYaml = 'plugin:\n  id: test';
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test',
          name: 'Test',
          description: 'Test',
          version: '1.0.0',
          category: 'analysis'
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'script',
          script: 'test.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      wailsMock.parsePluginYAML.mockResolvedValue(mockDefinition);

      const result = await service.yamlToDefinition(mockYaml);

      expect(result).toEqual(mockDefinition);
      expect(wailsMock.parsePluginYAML).toHaveBeenCalledWith(mockYaml);
    });
  });

  describe('getTemplates', () => {
    it('should return list of templates', async () => {
      const mockPlugins = [
        {
          definition: {
            plugin: { id: 'pca', name: 'PCA Analysis', category: 'analysis' }
          }
        },
        {
          definition: {
            plugin: { id: 'limma', name: 'Limma', category: 'analysis' }
          }
        }
      ];

      wailsMock.getPluginTemplates.mockResolvedValue(mockPlugins);

      const templates = await service.getTemplates();

      expect(templates.length).toBe(2);
      expect(templates[0].id).toBe('pca');
      expect(templates[0].name).toBe('PCA Analysis');
      expect(templates[1].id).toBe('limma');
    });
  });

  describe('createFromTemplate', () => {
    it('should create new plugin from template', async () => {
      const mockPlugins = [
        {
          definition: {
            plugin: { id: 'pca', name: 'PCA Analysis', category: 'analysis', version: '2.0.0' },
            runtime: { environments: ['python'], entrypoint: 'script', script: 'pca.py' },
            inputs: [{ name: 'input_file', label: 'Input File', type: 'file', required: true }],
            outputs: [],
            plots: [],
            execution: { argsMapping: {}, outputDir: '--output' }
          }
        }
      ];

      wailsMock.getPluginTemplates.mockResolvedValue(mockPlugins);

      const newDefinition = await service.createFromTemplate('pca', 'my-pca', 'My PCA');

      expect(newDefinition.plugin.id).toBe('my-pca');
      expect(newDefinition.plugin.name).toBe('My PCA');
      expect(newDefinition.plugin.version).toBe('1.0.0');
      expect(newDefinition.runtime.environments[0]).toBe('python');
      expect(newDefinition.inputs.length).toBe(1);
    });

    it('should throw error if template not found', async () => {
      wailsMock.getPluginTemplates.mockResolvedValue([]);

      await expect(
        service.createFromTemplate('nonexistent', 'new-id', 'New Name')
      ).rejects.toThrow('Template plugin not found: nonexistent');
    });
  });

  describe('generatePluginID', () => {
    it('should convert name to kebab-case', () => {
      expect(service.generatePluginID('My New Plugin')).toBe('my-new-plugin');
      expect(service.generatePluginID('PCA Analysis')).toBe('pca-analysis');
      expect(service.generatePluginID('test_plugin_name')).toBe('test-plugin-name');
      expect(service.generatePluginID('Test@#$Plugin')).toBe('test-plugin');
    });

    it('should handle special characters', () => {
      expect(service.generatePluginID('Hello!!! World??')).toBe('hello-world');
      expect(service.generatePluginID('---test---')).toBe('test');
    });
  });

  describe('validation methods', () => {
    it('should validate input names', () => {
      expect(service.validateInputName('input_file')).toBe(true);
      expect(service.validateInputName('inputFile')).toBe(true);
      expect(service.validateInputName('input1')).toBe(true);
      expect(service.validateInputName('_invalid')).toBe(false);
      expect(service.validateInputName('1invalid')).toBe(false);
      expect(service.validateInputName('input-file')).toBe(false);
    });

    it('should validate plugin IDs', () => {
      expect(service.validatePluginID('my-plugin')).toBe(true);
      expect(service.validatePluginID('test123')).toBe(true);
      expect(service.validatePluginID('my-plugin-123')).toBe(true);
      expect(service.validatePluginID('My-Plugin')).toBe(false);
      expect(service.validatePluginID('123-plugin')).toBe(false);
      expect(service.validatePluginID('plugin_name')).toBe(false);
    });

    it('should validate version numbers', () => {
      expect(service.validateVersion('1.0.0')).toBe(true);
      expect(service.validateVersion('2.10.3')).toBe(true);
      expect(service.validateVersion('1.0')).toBe(false);
      expect(service.validateVersion('1.0.0.0')).toBe(false);
      expect(service.validateVersion('v1.0.0')).toBe(false);
    });
  });

  describe('getters', () => {
    it('should return available input types', () => {
      const types = service.getAvailableInputTypes();
      expect(types.length).toBeGreaterThan(0);
      expect(types.find(t => t.value === 'file')).toBeDefined();
      expect(types.find(t => t.value === 'text')).toBeDefined();
      expect(types.find(t => t.value === 'number')).toBeDefined();
    });

    it('should return available categories', () => {
      const categories = service.getAvailableCategories();
      expect(categories).toContain('analysis');
      expect(categories).toContain('preprocessing');
      expect(categories).toContain('visualization');
      expect(categories).toContain('utilities');
    });

    it('should return available runtime types', () => {
      const types = service.getAvailableRuntimeTypes();
      expect(types.find(t => t.value === 'python')).toBeDefined();
      expect(types.find(t => t.value === 'r')).toBeDefined();
      expect(types.find(t => t.value === 'pythonWithR')).toBeDefined();
      expect(types.find(t => t.value === 'direct')).toBeDefined();
    });

    it('should return available output types', () => {
      const types = service.getAvailableOutputTypes();
      expect(types).toContain('data');
      expect(types).toContain('image');
      expect(types).toContain('plot');
      expect(types).toContain('log');
    });

    it('should return available output formats', () => {
      const formats = service.getAvailableOutputFormats();
      expect(formats).toContain('tsv');
      expect(formats).toContain('csv');
      expect(formats).toContain('json');
      expect(formats).toContain('svg');
      expect(formats).toContain('png');
    });

    it('should return available transforms', () => {
      const transforms = service.getAvailableTransforms();
      expect(transforms).toContain('comma-join');
      expect(transforms).toContain('space-join');
      expect(transforms).toContain('json-encode');
    });

    it('should return available when conditions', () => {
      const conditions = service.getAvailableWhenConditions();
      expect(conditions).toContain('true');
      expect(conditions).toContain('false');
      expect(conditions).toContain('not-empty');
      expect(conditions).toContain('empty');
    });
  });
});
