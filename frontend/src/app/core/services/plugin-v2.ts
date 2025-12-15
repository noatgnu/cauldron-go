import { Injectable } from '@angular/core';
import { GetPluginsV2, GetPluginV2, ExecutePluginV2, ReloadPluginsV2 } from '../../../wailsjs/go/main/App';
import { models } from '../../../wailsjs/go/models';

@Injectable({
  providedIn: 'root'
})
export class PluginV2Service {
  constructor() { }

  async getAllPlugins(): Promise<models.PluginV2[]> {
    return GetPluginsV2();
  }

  async getPlugin(id: number): Promise<models.PluginV2> {
    return GetPluginV2(id);
  }

  async executePlugin(pluginId: number, parameters: Record<string, any>): Promise<string> {
    const plugin = await this.getPlugin(pluginId);
    const convertedParameters = { ...parameters };

    for (const input of plugin.definition.inputs) {
      if (input.type === 'file' && convertedParameters[input.name]) {
        const value = convertedParameters[input.name];
        if (typeof value === 'string') {
          convertedParameters[input.name] = value.replace(/\\/g, '/');
        }
      }
    }

    const request = new models.PluginExecutionRequestV2({
      pluginId,
      parameters: convertedParameters
    });
    return ExecutePluginV2(request);
  }

  async reloadPlugins(): Promise<void> {
    return ReloadPluginsV2();
  }

  getPluginsByCategory(plugins: models.PluginV2[]): Map<string, models.PluginV2[]> {
    const categoryMap = new Map<string, models.PluginV2[]>();

    for (const plugin of plugins) {
      const category = plugin.definition.plugin.category || 'uncategorized';
      if (!categoryMap.has(category)) {
        categoryMap.set(category, []);
      }
      categoryMap.get(category)!.push(plugin);
    }

    return categoryMap;
  }

  filterPluginsByRuntime(plugins: models.PluginV2[], runtime: string): models.PluginV2[] {
    return plugins.filter(p => p.definition.runtime.type === runtime);
  }

  searchPlugins(plugins: models.PluginV2[], query: string): models.PluginV2[] {
    const lowerQuery = query.toLowerCase();
    return plugins.filter(p =>
      p.definition.plugin.name.toLowerCase().includes(lowerQuery) ||
      p.definition.plugin.description.toLowerCase().includes(lowerQuery) ||
      p.definition.plugin.id.toLowerCase().includes(lowerQuery)
    );
  }
}
