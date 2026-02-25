import { PluginDefinition } from '@cauldron/forms';

export interface WebrPackage {
  name: string;
  repo: string;
}

export interface Environment {
  production: boolean;
  runtime: 'pyodide' | 'webr';
  pyodideVersion: string;
  pyodidePackages: string[];
  webrVersion: string;
  webrPackages: WebrPackage[];
  pluginDefinition: PluginDefinition;
  pluginScript: string;
  pluginModules: Record<string, string>;
  hasExample: boolean;
  exampleBasePath: string;
}

export const environment: Environment = {
  production: false,
  runtime: 'pyodide',
  pyodideVersion: '0.27.5',
  pyodidePackages: [],
  webrVersion: '0.5.0',
  webrPackages: [],
  pluginDefinition: {
    plugin: {
      id: 'placeholder',
      name: 'Placeholder Plugin',
      description: 'This is a placeholder plugin definition',
      version: '1.0.0',
      category: 'analysis'
    },
    runtime: {
      environments: ['python'],
      entrypoint: 'main.py'
    },
    inputs: [],
    outputs: [],
    execution: {
      argsMapping: {},
      outputDir: '--output'
    }
  },
  pluginScript: '',
  pluginModules: {},
  hasExample: false,
  exampleBasePath: 'assets/examples/'
};
