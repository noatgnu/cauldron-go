import { Component, OnInit, signal, computed, inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTabsModule } from '@angular/material/tabs';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatDividerModule } from '@angular/material/divider';
import { MatExpansionModule } from '@angular/material/expansion';
import { FormsModule } from '@angular/forms';
import { Wails, CustomEnvVar } from '../../../core/services/wails';
import { PluginV2Service } from '../../../core/services/plugin-v2';
import { NotificationService } from '../../../core/services/notification.service';
import * as models from '../../../../../bindings/github.com/noatgnu/cauldron-go/backend/models/models';
import { DynamicFormComponent } from '../../../components/dynamic-form/dynamic-form';

@Component({
  selector: 'app-settings-env',
  standalone: true,
  imports: [
    CommonModule,
    MatTabsModule,
    MatCardModule,
    MatIconModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatListModule,
    MatDividerModule,
    MatExpansionModule,
    FormsModule,
    DynamicFormComponent
  ],
  templateUrl: './settings-env.html',
  styleUrl: './settings-env.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class SettingsEnv implements OnInit {
  private wails = inject(Wails);
  private pluginService = inject(PluginV2Service);
  private notification = inject(NotificationService);

  loading = signal(true);
  globalVars = signal<CustomEnvVar[]>([]);
  plugins = signal<models.PluginV2[]>([]);
  
  // Mapping of pluginId -> Map of Key -> Value
  pluginOverrides = signal<Map<number, Record<string, string>>>(new Map());

  newGlobalKey = signal('');
  newGlobalValue = signal('');

  filteredPlugins = computed(() => {
    return this.plugins().sort((a, b) => 
      a.definition.plugin.name.localeCompare(b.definition.plugin.name)
    );
  });

  async ngOnInit() {
    await this.loadAll();
  }

  async loadAll() {
    this.loading.set(true);
    try {
      const [globals, plugins] = await Promise.all([
        this.wails.getGlobalCustomEnvVars(),
        this.pluginService.getAllPlugins()
      ]);

      this.globalVars.set(globals || []);
      this.plugins.set(plugins || []);

      // Load overrides for each plugin
      const overrideMap = new Map<number, Record<string, string>>();
      for (const p of plugins) {
        const vars = await this.wails.getCustomEnvVars(p.id);
        const record: Record<string, string> = {};
        vars.forEach(v => record[v.Key] = v.Value);
        overrideMap.set(p.id, record);
      }
      this.pluginOverrides.set(overrideMap);

    } catch (error) {
      console.error('Failed to load environment variables:', error);
      this.notification.showError('Failed to load data');
    } finally {
      this.loading.set(false);
    }
  }

  async addGlobalVar() {
    const key = this.newGlobalKey().trim();
    const value = this.newGlobalValue().trim();
    if (!key || !value) return;

    try {
      await this.wails.saveCustomEnvVar({
        PluginID: 0,
        Key: key,
        Value: value
      } as any);
      this.newGlobalKey.set('');
      this.newGlobalValue.set('');
      await this.loadAll();
      this.notification.showSuccess('Global variable added');
    } catch (error) {
      this.notification.showError('Failed to save variable');
    }
  }

  async deleteGlobalVar(id: number) {
    try {
      await this.wails.deleteCustomEnvVar(id);
      await this.loadAll();
      this.notification.showSuccess('Variable removed');
    } catch (error) {
      this.notification.showError('Failed to delete variable');
    }
  }

  getPluginEnvWrapper(plugin: models.PluginV2): models.PluginV2 | null {
    const envVars = plugin.definition.execution?.envVariables || [];
    if (envVars.length === 0) return null;

    return {
      ...plugin,
      definition: {
        ...plugin.definition,
        inputs: envVars,
        example: { enabled: false, values: {} }
      }
    } as models.PluginV2;
  }

  getPluginFormValues(pluginId: number): Record<string, any> {
    return this.pluginOverrides().get(pluginId) || {};
  }

  async savePluginVars(pluginId: number, values: Record<string, any>) {
    try {
      for (const [key, value] of Object.entries(values)) {
        await this.wails.saveCustomEnvVar({
          PluginID: pluginId,
          Key: key,
          Value: String(value)
        } as any);
      }
      await this.loadAll();
      this.notification.showSuccess('Plugin settings saved');
    } catch (error) {
      this.notification.showError('Failed to save plugin variables');
    }
  }
}
