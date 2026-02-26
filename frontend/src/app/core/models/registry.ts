export interface RegistryAuthor {
  id: number;
  name: string;
  email?: string;
}

export interface RegistryCategory {
  id: number;
  name: string;
  description?: string;
}

export interface RegistryTag {
  id: number;
  name: string;
}

export interface RegistryRuntime {
  id: number;
  plugin: string;
  environments: string[];
  entrypoint: string;
  docker?: {
    image?: string;
    dockerfile?: string;
    build_args?: Record<string, string>;
  } | null;
}

export interface RegistryInput {
  id: number;
  plugin: string;
  name: string;
  label: string;
  type: string;
  required: boolean;
  default?: string | null;
  description?: string | null;
  placeholder?: string | null;
  file_types?: string[] | null;
  accept?: string | null;
  multiple: boolean;
  sourceFile?: string | null;
  min?: number | null;
  max?: number | null;
  step?: number | null;
  options?: any | null;
  optionsFromFile?: string | null;
  groups?: any | null;
  groupsFromFile?: string | null;
  visibleWhen?: any | null;
  disableAnnotationManagement: boolean;
  tableColumns?: any | null;
}

export interface RegistryOutput {
  id: number;
  plugin: string;
  name: string;
  path: string;
  type: string;
  description?: string | null;
  format?: string | null;
}

export interface RegistryEnvVariable {
  id: number;
  plugin: string;
  name: string;
  label: string;
  type: string;
  required: boolean;
  default?: string | null;
  description?: string | null;
  placeholder?: string | null;
  accept?: string | null;
  multiple: boolean;
  sourceFile?: string | null;
  min?: number | null;
  max?: number | null;
  step?: number | null;
}

export interface RegistryExecution {
  id: number;
  plugin: string;
  argsMapping?: Record<string, any> | null;
  outputDir?: string | null;
  requirements?: any | null;
}

export interface RegistryPlot {
  id: number;
  plugin: string;
  plot_id: string;
  name: string;
  type: string;
  component: string;
  dataSource: string;
  config?: any | null;
  customization?: any | null;
}

export interface RegistryAnnotation {
  id: number;
  plugin: string;
  samplesFrom?: string | null;
  annotationFile?: string | null;
}

export interface RegistryExample {
  id: number;
  plugin: string;
  enabled: boolean;
  values?: Record<string, any> | null;
}

export interface RegistryPlugin {
  id: string;
  name: string;
  description: string;
  version: string;
  author: RegistryAuthor | null;
  category: RegistryCategory | null;
  subcategory?: string | null;
  icon?: string | null;
  repository?: string | null;
  commit_hash?: string | null;
  recommended_commit?: string | null;
  latest_stable_tag?: string | null;
  readme?: string | null;
  diagram_enabled: boolean;
  citation_enabled: boolean;
  requires_authentication: boolean;
  status: 'pending' | 'approved' | 'rejected';
  submitted_by?: number | null;
  created_at: string;
  updated_at: string;
  tags: RegistryTag[];
  runtime?: RegistryRuntime | null;
  inputs: RegistryInput[];
  outputs: RegistryOutput[];
  env_variables: RegistryEnvVariable[];
  execution?: RegistryExecution | null;
  plots: RegistryPlot[];
  annotation?: RegistryAnnotation | null;
  example?: RegistryExample | null;
}

export interface RegistryPluginListResponse {
  count: number;
  next: string | null;
  previous: string | null;
  results: RegistryPlugin[];
}

export interface RegistryCategoryListResponse {
  count: number;
  next: string | null;
  previous: string | null;
  results: RegistryCategory[];
}

export interface PluginUpdateInfo {
  plugin_id: string;
  current_commit: string;
  latest_commit: string;
  recommended_commit: string;
  latest_stable_tag: string | null;
  has_update: boolean;
  changelog_url: string | null;
}
