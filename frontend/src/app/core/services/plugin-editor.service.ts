import { Injectable } from '@angular/core';
import { Wails } from './wails';
import { models } from '../../../wailsjs/go/models';

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

export interface PluginTemplate {
  id: string;
  name: string;
  category: string;
  definition: models.PluginDefinition;
}

@Injectable({
  providedIn: 'root'
})
export class PluginEditorService {
  constructor(private wails: Wails) {}

  async savePlugin(pluginID: string, definition: models.PluginDefinition): Promise<void> {
    const yaml = await this.definitionToYAML(definition);
    await this.wails.savePluginYAML(pluginID, yaml);
    await this.wails.logToFile(`[PluginEditor] Saved plugin: ${pluginID}`);
  }

  async validatePlugin(definition: models.PluginDefinition): Promise<ValidationResult> {
    try {
      const yaml = await this.definitionToYAML(definition);
      const result = await this.wails.validatePluginYAML(yaml);
      return result;
    } catch (error) {
      return {
        valid: false,
        errors: [`Validation error: ${error}`]
      };
    }
  }

  async deletePlugin(pluginID: string): Promise<void> {
    await this.wails.deletePlugin(pluginID);
    await this.wails.logToFile(`[PluginEditor] Deleted plugin: ${pluginID}`);
  }

  async definitionToYAML(definition: models.PluginDefinition): Promise<string> {
    return await this.wails.convertPluginToYAML(definition);
  }

  async yamlToDefinition(yaml: string): Promise<models.PluginDefinition> {
    return await this.wails.parsePluginYAML(yaml);
  }

  async getTemplates(): Promise<PluginTemplate[]> {
    const plugins = await this.wails.getPluginTemplates();
    return plugins.map((plugin: any) => ({
      id: plugin.definition.plugin.id,
      name: plugin.definition.plugin.name,
      category: plugin.definition.plugin.category,
      definition: plugin.definition
    }));
  }

  async createFromTemplate(sourcePluginID: string, newPluginID: string, newName: string): Promise<models.PluginDefinition> {
    const templates = await this.getTemplates();
    const template = templates.find(t => t.id === sourcePluginID);

    if (!template) {
      throw new Error(`Template plugin not found: ${sourcePluginID}`);
    }

    const definition = JSON.parse(JSON.stringify(template.definition));
    definition.plugin.id = newPluginID;
    definition.plugin.name = newName;
    definition.plugin.version = '1.0.0';

    return definition;
  }

  generatePluginID(name: string): string {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
  }

  validateInputName(name: string): boolean {
    return /^[a-zA-Z][a-zA-Z0-9_]*$/.test(name);
  }

  validatePluginID(id: string): boolean {
    return /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/.test(id);
  }

  validateVersion(version: string): boolean {
    return /^\d+\.\d+\.\d+$/.test(version);
  }

  getAvailableInputTypes(): Array<{value: string, label: string, description: string}> {
    return [
      { value: 'file', label: 'File', description: 'File upload input' },
      { value: 'text', label: 'Text', description: 'Single-line text input' },
      { value: 'number', label: 'Number', description: 'Numeric input with min/max' },
      { value: 'boolean', label: 'Boolean', description: 'Checkbox/toggle' },
      { value: 'select', label: 'Select', description: 'Dropdown with options' },
      { value: 'multiselect-grouped', label: 'Multi-select Grouped', description: 'Multi-select with option groups' },
      { value: 'column-selector', label: 'Column Selector', description: 'Select columns from a data file' }
    ];
  }

  getAvailableCategories(): string[] {
    return ['analysis', 'preprocessing', 'visualization', 'utilities'];
  }

  getAvailableRuntimeTypes(): Array<{value: string, label: string}> {
    return [
      { value: 'python', label: 'Python' },
      { value: 'r', label: 'R' },
      { value: 'pythonWithR', label: 'Python with R' },
      { value: 'direct', label: 'Direct Executable' }
    ];
  }

  getAvailableOutputTypes(): string[] {
    return ['data', 'image', 'plot', 'log'];
  }

  getAvailableOutputFormats(): string[] {
    return ['tsv', 'csv', 'json', 'txt', 'svg', 'png', 'pdf'];
  }

  getAvailableTransforms(): string[] {
    return ['comma-join', 'space-join', 'json-encode'];
  }

  getAvailableWhenConditions(): string[] {
    return ['true', 'false', 'not-empty', 'empty'];
  }
}
