import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, FormGroup, FormArray, Validators } from '@angular/forms';
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
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { Wails, PluginEnvironmentBinding } from '../../core/services/wails';
import { Plugin, PluginInput, PluginExecutionRequest } from '../../core/models/plugin';
import { EnvironmentIndicator } from '../../components/environment-indicator/environment-indicator';
import { InstallPluginDialog } from '../../components/install-plugin-dialog/install-plugin-dialog';

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
    MatDialogModule,
    EnvironmentIndicator
  ],
  templateUrl: './plugins.html',
  styleUrl: './plugins.scss'
})
export class Plugins implements OnInit {
  protected plugins = signal<Plugin[]>([]);
  protected loading = signal(false);
  protected executing = signal<Record<string, boolean>>({});
  protected pluginsDirectory = signal('');
  protected bindings = signal<PluginEnvironmentBinding[]>([]);

  mainFormGroup!: FormGroup;

  constructor(
    private wails: Wails,
    private fb: FormBuilder,
    private dialog: MatDialog
  ) {}

  get pluginsFormArray(): FormArray {
    return this.mainFormGroup.get('plugins') as FormArray;
  }

  async ngOnInit() {
    this.mainFormGroup = this.fb.group({
      plugins: this.fb.array([])
    });

    await this.loadPlugins();
    await this.loadPluginsDirectory();

    this.wails.bindingsUpdated$.subscribe(() => {
      this.loadPlugins();
    });
  }

  async loadPlugins() {
    this.loading.set(true);

    try {
      const [plugins, bindings] = await Promise.all([
        this.wails.getPlugins(),
        this.wails.getAllPluginEnvironmentBindings()
      ]);

      await this.wails.logToFile(`[Plugins] Before sort: ${plugins.map(p => p.config.name).join(', ')}`);

      const sortedPlugins = [...plugins].sort((a, b) => {
        const catA = a.config.category || '';
        const catB = b.config.category || '';

        if (catA !== catB) {
          return catA.localeCompare(catB);
        }

        return a.config.name.localeCompare(b.config.name);
      });

      await this.wails.logToFile(`[Plugins] After sort: ${sortedPlugins.map(p => p.config.name).join(', ')}`);

      this.plugins.set(sortedPlugins);
      this.bindings.set(bindings || []);

      const pluginsArray = this.mainFormGroup.get('plugins') as FormArray;
      pluginsArray.clear();

      for (const plugin of sortedPlugins) {
        pluginsArray.push(this.createFormForPlugin(plugin));
      }
    } catch (error) {
      await this.wails.logToFile(`[Plugins] Failed to load plugins: ${error}`);
    } finally {
      this.loading.set(false);
    }
  }

  async loadPluginsDirectory() {
    try {
      const dir = await this.wails.getPluginsDirectory();
      this.pluginsDirectory.set(dir);
    } catch (error) {
      await this.wails.logToFile(`[Plugins] Failed to get plugins directory: ${error}`);
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

  async openFileForInput(index: number, inputName: string) {
    try {
      const path = await this.wails.openDataFileDialog();
      if (path) {
        const formGroup = this.pluginsFormArray.at(index) as FormGroup;
        formGroup.patchValue({ [inputName]: path });
      }
    } catch (error) {
      await this.wails.logToFile(`[Plugins] Failed to open file dialog: ${error}`);
    }
  }

  async executePlugin(index: number) {
    const plugin = this.plugins()[index];
    const form = this.pluginsFormArray.at(index) as FormGroup;

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
      await this.wails.logToFile(`[Plugins] Plugin execution started: ${jobId}`);
    } catch (error) {
      await this.wails.logToFile(`[Plugins] Failed to execute plugin: ${error}`);
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
      await this.wails.logToFile(`[Plugins] Failed to reload plugins: ${error}`);
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
      await this.wails.logToFile(`[Plugins] Failed to create sample plugin: ${error}`);
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
      await this.wails.logToFile(`[Plugins] Failed to open plugins folder: ${error}`);
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

  async openInstallDialog() {
    const dialogRef = this.dialog.open(InstallPluginDialog, {
      width: '600px',
      disableClose: false
    });

    dialogRef.afterClosed().subscribe(async (repoURL: string) => {
      if (repoURL) {
        this.loading.set(true);
        try {
          await this.wails.installPluginFromRepo(repoURL);
          await this.wails.logToFile(`[Plugins] Successfully installed plugin from: ${repoURL}`);
          await this.loadPlugins();
        } catch (error) {
          await this.wails.logToFile(`[Plugins] Failed to install plugin: ${error}`);
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  isBound(pluginId: string, runtime: string): boolean {
    const bindings = this.bindings();
    if (runtime === 'pythonWithR') {
      const hasPython = bindings.some(b => b.PluginID === pluginId && b.EnvironmentType === 'python');
      const hasR = bindings.some(b => b.PluginID === pluginId && b.EnvironmentType === 'r');
      return hasPython && hasR;
    }
    return bindings.some(b => b.PluginID === pluginId && b.EnvironmentType === runtime);
  }
}
