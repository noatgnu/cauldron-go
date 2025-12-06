export type PluginRuntime = 'python' | 'r' | 'pythonWithR';

export type PluginInputType = 'file' | 'text' | 'number' | 'boolean' | 'select' | 'multiselect';

export interface PluginInput {
  name: string;
  label: string;
  type: PluginInputType;
  required: boolean;
  default?: any;
  options?: string[];
  description?: string;
  placeholder?: string;
}

export interface PluginOutput {
  name: string;
  description: string;
}

export interface PluginScript {
  path: string;
}

export interface PluginConfig {
  name: string;
  description: string;
  version: string;
  author?: string;
  runtime: PluginRuntime;
  script: PluginScript;
  inputs: PluginInput[];
  outputs?: PluginOutput[];
}

export interface Plugin {
  id: string;
  config: PluginConfig;
  folderPath: string;
  scriptPath: string;
}

export interface PluginExecutionRequest {
  pluginId: string;
  parameters: Record<string, any>;
}
