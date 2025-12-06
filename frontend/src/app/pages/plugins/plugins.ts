import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Wails } from '../../core/services/wails';
import { Plugin, PluginInput, PluginExecutionRequest } from '../../core/models/plugin';
import { EnvironmentIndicator } from '../../components/environment-indicator/environment-indicator';

@Component({
  selector: 'app-plugins',
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatExpansionModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatChipsModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    EnvironmentIndicator
  ],
  templateUrl: './plugins.html',
  styleUrl: './plugins.scss',
})
export class Plugins implements OnInit {
  protected plugins = signal<Plugin[]>([]);
  protected loading = signal(false);
  protected executing = signal<Record<string, boolean>>({});
  protected pluginForms = signal<Record<string, FormGroup>>({});
  protected pluginsDirectory = signal('');

  constructor(
    private wails: Wails,
    private fb: FormBuilder
  ) {}

  async ngOnInit() {
    await this.loadPlugins();
    await this.loadPluginsDirectory();
  }

  async loadPlugins() {
    this.loading.set(true);
    try {
      const plugins = await this.wails.getPlugins();
      this.plugins.set(plugins);

      const forms: Record<string, FormGroup> = {};
      for (const plugin of plugins) {
        forms[plugin.id] = this.createFormForPlugin(plugin);
      }
      this.pluginForms.set(forms);
    } catch (error) {
      console.error('Failed to load plugins:', error);
    } finally {
      this.loading.set(false);
    }
  }

  async loadPluginsDirectory() {
    try {
      const dir = await this.wails.getPluginsDirectory();
      this.pluginsDirectory.set(dir);
    } catch (error) {
      console.error('Failed to get plugins directory:', error);
    }
  }

  createFormForPlugin(plugin: Plugin): FormGroup {
    const group: Record<string, any> = {};

    for (const input of plugin.config.inputs) {
      const validators = [];
      if (input.required) {
        validators.push(Validators.required);
      }

      let defaultValue: any = '';
      if (input.default !== undefined && input.default !== null) {
        defaultValue = input.default;
      } else if (input.type === 'boolean') {
        defaultValue = false;
      } else if (input.type === 'number') {
        defaultValue = 0;
      } else if (input.type === 'multiselect') {
        defaultValue = [];
      }

      group[input.name] = [defaultValue, validators];
    }

    return this.fb.group(group);
  }

  getForm(pluginId: string): FormGroup | undefined {
    return this.pluginForms()[pluginId];
  }

  async openFileForInput(pluginId: string, inputName: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        const form = this.getForm(pluginId);
        if (form) {
          form.patchValue({ [inputName]: path });
        }
      }
    } catch (error) {
      console.error('Failed to open file dialog:', error);
    }
  }

  async executePlugin(plugin: Plugin) {
    const form = this.getForm(plugin.id);
    if (!form || form.invalid) {
      return;
    }

    this.executing.update(state => ({ ...state, [plugin.id]: true }));

    try {
      const request: PluginExecutionRequest = {
        pluginId: plugin.id,
        parameters: form.value
      };

      const jobId = await this.wails.executePlugin(request);
      console.log('Plugin execution started:', jobId);
    } catch (error) {
      console.error('Failed to execute plugin:', error);
    } finally {
      this.executing.update(state => ({ ...state, [plugin.id]: false }));
    }
  }

  async reloadPlugins() {
    this.loading.set(true);
    try {
      await this.wails.reloadPlugins();
      await this.loadPlugins();
    } catch (error) {
      console.error('Failed to reload plugins:', error);
    } finally {
      this.loading.set(false);
    }
  }

  async createSamplePlugin() {
    this.loading.set(true);
    try {
      await this.wails.createSamplePlugin();
      await this.loadPlugins();
    } catch (error) {
      console.error('Failed to create sample plugin:', error);
    } finally {
      this.loading.set(false);
    }
  }

  async openPluginsFolder() {
    try {
      if (window.runtime && window.runtime.BrowserOpenURL) {
        window.runtime.BrowserOpenURL(`file://${this.pluginsDirectory()}`);
      }
    } catch (error) {
      console.error('Failed to open plugins folder:', error);
    }
  }

  getRuntimeIcon(runtime: string): string {
    switch (runtime) {
      case 'python': return 'code';
      case 'r': return 'analytics';
      case 'pythonWithR': return 'integration_instructions';
      default: return 'extension';
    }
  }

  getInputIcon(type: string): string {
    switch (type) {
      case 'file': return 'insert_drive_file';
      case 'number': return 'tag';
      case 'text': return 'text_fields';
      case 'boolean': return 'toggle_on';
      case 'select': return 'list';
      case 'multiselect': return 'checklist';
      default: return 'input';
    }
  }
}
