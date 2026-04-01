import { ComponentFixture, TestBed } from '@angular/core/testing';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { PluginEditor } from './plugin-editor';
import { PluginEditorService } from '../../core/services/plugin-editor.service';
import { Wails } from '../../core/services/wails';
import { Router } from '@angular/router';
import { ActivatedRoute } from '@angular/router';
import { ReactiveFormsModule } from '@angular/forms';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import * as models from '../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import { of } from 'rxjs';

describe('PluginEditor', () => {
  let component: PluginEditor;
  let fixture: ComponentFixture<PluginEditor>;
  let pluginEditorServiceMock: any;
  let wailsMock: any;
  let routerMock: any;
  let activatedRouteMock: any;

  beforeEach(async () => {
    pluginEditorServiceMock = {
      savePlugin: vi.fn(),
      validatePlugin: vi.fn(),
      deletePlugin: vi.fn(),
      definitionToYAML: vi.fn(),
      yamlToDefinition: vi.fn(),
      getTemplates: vi.fn(),
      createFromTemplate: vi.fn(),
      generatePluginID: vi.fn(),
      getAvailableCategories: vi.fn().mockReturnValue(['analysis', 'preprocessing', 'visualization', 'utilities']),
      getAvailableRuntimeTypes: vi.fn().mockReturnValue([
        { value: 'python', label: 'Python' },
        { value: 'r', label: 'R' },
        { value: 'pythonWithR', label: 'Python with R' },
        { value: 'direct', label: 'Direct' }
      ]),
      getAvailableInputTypes: vi.fn().mockReturnValue([
        { value: 'file', label: 'File', description: 'File input' },
        { value: 'text', label: 'Text', description: 'Text input' },
        { value: 'number', label: 'Number', description: 'Number input' }
      ])
    };

    wailsMock = {
      getPlugin: vi.fn(),
      getPluginsV2: vi.fn(),
      logToFile: vi.fn()
    };

    routerMock = {
      navigate: vi.fn()
    };

    activatedRouteMock = {
      snapshot: {
        paramMap: {
          get: vi.fn().mockReturnValue(null)
        },
        queryParamMap: {
          get: vi.fn().mockReturnValue(null)
        }
      }
    };

    await TestBed.configureTestingModule({
      imports: [PluginEditor, ReactiveFormsModule, NoopAnimationsModule],
      providers: [
        { provide: PluginEditorService, useValue: pluginEditorServiceMock },
        { provide: Wails, useValue: wailsMock },
        { provide: Router, useValue: routerMock },
        { provide: ActivatedRoute, useValue: activatedRouteMock }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(PluginEditor);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  describe('initialization', () => {
    it('should initialize in create mode by default', async () => {
      fixture.detectChanges();
      await fixture.whenStable();

      expect(component.mode()).toBe('create');
    });

    it('should load plugin in edit mode', async () => {
      const mockPlugin = {
        id: 1,
        definition: {
          plugin: {
            id: 'test-plugin',
            name: 'Test Plugin',
            description: 'Test description',
            version: '1.0.0',
            category: models.PluginCategory.PluginCategoryAnalysis
          },
          runtime: {
            environments: ['python'],
            entrypoint: 'test.py'
          },
          inputs: [],
          outputs: [],
          plots: [],
          execution: {
            argsMapping: {},
            outputDir: '--output'
          }
        }
      };

      activatedRouteMock.snapshot.paramMap.get = vi.fn().mockReturnValue('1');
      wailsMock.getPluginsV2.mockResolvedValue([mockPlugin]);
      wailsMock.logToFile.mockResolvedValue(undefined);

      fixture.detectChanges();
      await fixture.whenStable();

      expect(component.mode()).toBe('edit');
      expect(wailsMock.getPluginsV2).toHaveBeenCalled();
    });

    it('should clone from template', async () => {
      const mockDefinition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: '',
          name: 'Copy of pca',
          description: 'Test',
          version: '1.0.0',
          category: models.PluginCategory.PluginCategoryAnalysis
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['python'],
          entrypoint: 'pca.py'
        }),
        inputs: [],
        outputs: [],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output'
        })
      });

      activatedRouteMock.snapshot.paramMap.get = vi.fn().mockReturnValue('new');
      activatedRouteMock.snapshot.queryParamMap.get = vi.fn().mockReturnValue('pca');
      pluginEditorServiceMock.createFromTemplate.mockResolvedValue(mockDefinition);
      wailsMock.logToFile.mockResolvedValue(undefined);

      fixture.detectChanges();
      await fixture.whenStable();

      expect(component.mode()).toBe('create');
      expect(pluginEditorServiceMock.createFromTemplate).toHaveBeenCalledWith('pca', '', 'Copy of pca');
    });
  });

  describe('form getters', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should return inputs FormArray', () => {
      expect(component.inputs).toBeDefined();
      expect(component.inputs.length).toBe(0);
    });

    it('should return outputs FormArray', () => {
      expect(component.outputs).toBeDefined();
      expect(component.outputs.length).toBe(0);
    });

    it('should return plots FormArray', () => {
      expect(component.plots).toBeDefined();
      expect(component.plots.length).toBe(0);
    });

    it('should return packages FormArray', () => {
      expect(component.packages).toBeDefined();
      expect(component.packages.length).toBe(0);
    });
  });

  describe('input management', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should add input', () => {
      const initialLength = component.inputs.length;
      component.addInput();
      expect(component.inputs.length).toBe(initialLength + 1);
    });

    it('should remove input', () => {
      component.addInput();
      component.addInput();
      const length = component.inputs.length;
      component.removeInput(0);
      expect(component.inputs.length).toBe(length - 1);
    });
  });

  describe('output management', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should add output', () => {
      const initialLength = component.outputs.length;
      component.addOutput();
      expect(component.outputs.length).toBe(initialLength + 1);
    });

    it('should remove output', () => {
      component.addOutput();
      component.addOutput();
      const length = component.outputs.length;
      component.removeOutput(0);
      expect(component.outputs.length).toBe(length - 1);
    });
  });

  describe('generatePreview', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should generate preview successfully', async () => {
      const mockYaml = 'plugin:\n  id: test';
      pluginEditorServiceMock.definitionToYAML.mockResolvedValue(mockYaml);
      pluginEditorServiceMock.validatePlugin.mockResolvedValue({
        valid: true,
        errors: []
      });
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.generatePreview();

      expect(component.yamlPreview()).toBe(mockYaml);
      expect(component.validationResult()?.valid).toBe(true);
    });

    it('should handle validation errors', async () => {
      const mockYaml = 'plugin:\n  id: ""';
      pluginEditorServiceMock.definitionToYAML.mockResolvedValue(mockYaml);
      pluginEditorServiceMock.validatePlugin.mockResolvedValue({
        valid: false,
        errors: ['Plugin ID is required']
      });
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.generatePreview();

      expect(component.validationResult()?.valid).toBe(false);
      expect(component.validationResult()?.errors).toContain('Plugin ID is required');
    });

    it('should handle preview generation errors', async () => {
      pluginEditorServiceMock.definitionToYAML.mockRejectedValue('Preview error');
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.generatePreview();

      expect(component.validationResult()?.valid).toBe(false);
      expect(component.validationResult()?.errors[0]).toContain('Preview generation error');
    });
  });

  describe('savePlugin', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should save valid plugin', async () => {
      component.pluginForm.patchValue({
        plugin: {
          id: 'test-plugin',
          name: 'Test Plugin',
          description: 'Test description long enough',
          version: '1.0.0',
          category: models.PluginCategory.PluginCategoryAnalysis,
          author: 'Test',
          icon: ''
        },
        runtime: {
          entrypoint: 'test.py'
        }
      });
      // environments is a FormArray, needs special handling if not using patchValue for it
      component.environments.clear();
      component.environments.push(component['fb'].control('python'));

      pluginEditorServiceMock.savePlugin.mockResolvedValue(undefined);
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.savePlugin();

      expect(pluginEditorServiceMock.savePlugin).toHaveBeenCalled();
      expect(routerMock.navigate).toHaveBeenCalledWith(['/plugin-list']);
    });

    it('should not save invalid plugin', async () => {
      component.pluginForm.patchValue({
        plugin: {
          id: '',
          name: '',
          description: '',
          version: '',
          category: '',
          author: '',
          icon: ''
        }
      });

      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.savePlugin();

      expect(pluginEditorServiceMock.savePlugin).not.toHaveBeenCalled();
      expect(wailsMock.logToFile).toHaveBeenCalled();
    });

    it('should handle save errors', async () => {
      component.pluginForm.patchValue({
        plugin: {
          id: 'test',
          name: 'Test',
          description: 'Test description long enough',
          version: '1.0.0',
          category: models.PluginCategory.PluginCategoryAnalysis,
          author: 'Test',
          icon: ''
        },
        runtime: {
          entrypoint: 'test.py'
        }
      });

      pluginEditorServiceMock.savePlugin.mockRejectedValue('Save error');
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.savePlugin();

      expect(wailsMock.logToFile).toHaveBeenCalledWith(expect.stringContaining('Save error'));
    });
  });

  describe('cancel', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should navigate to plugin list', () => {
      component.cancel();
      expect(routerMock.navigate).toHaveBeenCalledWith(['/plugin-list']);
    });
  });

  describe('onPluginNameChange', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should auto-generate plugin ID in create mode', async () => {
      component.mode.set('create');
      pluginEditorServiceMock.generatePluginID.mockReturnValue('my-test-plugin');

      await component.onPluginNameChange('My Test Plugin');

      expect(pluginEditorServiceMock.generatePluginID).toHaveBeenCalledWith('My Test Plugin');
      expect(component.pluginForm.get('plugin.id')?.value).toBe('my-test-plugin');
    });

    it('should not auto-generate if ID was manually edited', async () => {
      component.mode.set('create');
      component.pluginForm.get('plugin.id')?.markAsDirty();

      await component.onPluginNameChange('My Test Plugin');

      expect(pluginEditorServiceMock.generatePluginID).not.toHaveBeenCalled();
    });

    it('should not auto-generate in edit mode', async () => {
      component.mode.set('edit');

      await component.onPluginNameChange('My Test Plugin');

      expect(pluginEditorServiceMock.generatePluginID).not.toHaveBeenCalled();
    });
  });

  describe('form validation', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should validate plugin ID pattern', () => {
      const idControl = component.pluginForm.get('plugin.id');

      idControl?.setValue('my-plugin');
      expect(idControl?.valid).toBe(true);

      idControl?.setValue('My-Plugin');
      expect(idControl?.valid).toBe(false);

      idControl?.setValue('123-plugin');
      expect(idControl?.valid).toBe(false);
    });

    it('should validate version pattern', () => {
      const versionControl = component.pluginForm.get('plugin.version');

      versionControl?.setValue('1.0.0');
      expect(versionControl?.valid).toBe(true);

      versionControl?.setValue('1.0');
      expect(versionControl?.valid).toBe(false);

      versionControl?.setValue('v1.0.0');
      expect(versionControl?.valid).toBe(false);
    });

    it('should validate description minimum length', () => {
      const descControl = component.pluginForm.get('plugin.description');

      descControl?.setValue('Short');
      expect(descControl?.hasError('minlength')).toBe(true);

      descControl?.setValue('This is a long enough description');
      expect(descControl?.hasError('minlength')).toBe(false);
    });

    it('should validate input name pattern', () => {
      component.addInput();
      const nameControl = component.inputs.at(0).get('name');

      nameControl?.setValue('input_file');
      expect(nameControl?.valid).toBe(true);

      nameControl?.setValue('inputFile');
      expect(nameControl?.valid).toBe(true);

      nameControl?.setValue('_invalid');
      expect(nameControl?.valid).toBe(false);

      nameControl?.setValue('1invalid');
      expect(nameControl?.valid).toBe(false);
    });
  });

  describe('form population', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should populate form from definition', () => {
      const definition = new models.PluginDefinition({
        plugin: new models.PluginMetadata({
          id: 'test-plugin',
          name: 'Test Plugin',
          description: 'Test description',
          version: '2.0.0',
          category: models.PluginCategory.PluginCategoryAnalysis
        }),
        runtime: new models.PluginRuntimeV2({
          environments: ['r'],
          entrypoint: 'test.R'
        }),
        inputs: [
          new models.PluginInputV2({
            name: 'input1',
            label: 'Input 1',
            type: models.PluginInputType.PluginInputTypeFile,
            required: true
          })
        ],
        outputs: [
          new models.PluginOutputV2({
            name: 'output1',
            path: 'output.csv',
            type: 'data',
            format: 'csv'
          })
        ],
        plots: [],
        execution: new models.PluginExecution({
          argsMapping: {},
          outputDir: '--output_dir'
        })
      });

      component['populateForm'](definition);

      expect(component.pluginForm.get('plugin.id')?.value).toBe('test-plugin');
      expect(component.pluginForm.get('plugin.name')?.value).toBe('Test Plugin');
      expect(component.environments.at(0).value).toBe('r');
      expect(component.inputs.length).toBe(1);
      expect(component.outputs.length).toBe(1);
    });
  });

  describe('loading states', () => {
    beforeEach(() => {
      fixture.detectChanges();
    });

    it('should set loading state when loading plugin', async () => {
      wailsMock.getPlugin.mockImplementation(() => {
        expect(component.loading()).toBe(true);
        return Promise.resolve({
          definition: new models.PluginDefinition({
            plugin: new models.PluginMetadata({
              id: 'test',
              name: 'Test',
              description: 'Test',
              version: '1.0.0',
              category: models.PluginCategory.PluginCategoryAnalysis
            }),
            runtime: new models.PluginRuntimeV2({ environments: ['python'], entrypoint: 'test.py' }),
            inputs: [],
            outputs: [],
            plots: [],
            execution: new models.PluginExecution({ argsMapping: {}, outputDir: '--output' })
          })
        });
      });
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.loadPlugin('test');

      expect(component.loading()).toBe(false);
    });

    it('should set validating state during preview generation', async () => {
      pluginEditorServiceMock.definitionToYAML.mockImplementation(() => {
        expect(component.validating()).toBe(true);
        return Promise.resolve('plugin:\n  id: test');
      });
      pluginEditorServiceMock.validatePlugin.mockResolvedValue({ valid: true, errors: [] });
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.generatePreview();

      expect(component.validating()).toBe(false);
    });

    it('should set saving state during save', async () => {
      component.pluginForm.patchValue({
        plugin: {
          id: 'test',
          name: 'Test',
          description: 'Test description long enough',
          version: '1.0.0',
          category: models.PluginCategory.PluginCategoryAnalysis,
          author: 'Test',
          icon: ''
        },
        runtime: {
          entrypoint: 'test.py'
        }
      });

      pluginEditorServiceMock.savePlugin.mockImplementation(() => {
        expect(component.saving()).toBe(true);
        return Promise.resolve();
      });
      wailsMock.logToFile.mockResolvedValue(undefined);

      await component.savePlugin();

      expect(component.saving()).toBe(false);
    });
  });
});
