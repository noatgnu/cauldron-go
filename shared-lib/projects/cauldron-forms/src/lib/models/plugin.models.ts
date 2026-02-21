export interface FieldOption {
  value: string;
  label: string;
}

export interface FieldGroup {
  name: string;
  options: FieldOption[];
}

export interface VisibilityCondition {
  field: string;
  equals?: unknown;
  equalsAny?: unknown[];
}

export interface TableColumn {
  name: string;
  label: string;
  type?: 'text' | 'number';
  required?: boolean;
  description?: string;
}

export type SelectOption = string | FieldOption;

export interface PluginInputV2 {
  name: string;
  label: string;
  type: string;
  required: boolean;
  default?: unknown;
  options?: SelectOption[];
  optionsFromFile?: string;
  groups?: FieldGroup[];
  groupsFromFile?: string;
  description?: string;
  placeholder?: string;
  accept?: string;
  multiple?: boolean;
  sourceFile?: string;
  min?: number;
  max?: number;
  step?: number;
  visibleWhen?: VisibilityCondition;
  disableAnnotationManagement?: boolean;
  tableColumns?: TableColumn[];
}

export interface PluginOutputV2 {
  name: string;
  path: string;
  type: string;
  description: string;
  format: string;
}

export interface AnnotationConfig {
  samplesFrom?: string;
  annotationFile?: string;
}

export interface ExampleData {
  enabled: boolean;
  values: Record<string, unknown>;
}

export interface DockerConfig {
  image?: string;
  dockerfile?: string;
  platform?: string;
  buildArgs?: Record<string, string>;
}

export interface PluginRuntimeV2 {
  environments: string[];
  entrypoint: string;
  script?: string;
  docker?: DockerConfig;
}

export interface PluginMetadata {
  id: string;
  name: string;
  description: string;
  version: string;
  author?: string;
  category: string;
  subcategory?: string;
  icon?: string;
  repository?: string;
}

export interface PlotAxes {
  x: string;
  y: string;
  colorBy?: string;
  sizeBy?: string;
  labels?: string;
}

export interface PlotConfigData {
  axes: PlotAxes;
  imagePattern?: string;
  imagePatternType?: string;
}

export interface PlotCustomization {
  name: string;
  label: string;
  type: string;
  default?: unknown;
  min?: number;
  max?: number;
}

export interface PluginPlot {
  id: string;
  name: string;
  type: string;
  component: string;
  dataSource: string;
  config: PlotConfigData;
  customization: PlotCustomization[];
}

export interface Requirements {
  python?: string;
  r?: string;
  packages?: string[];
  pythonRequirementsFile?: string;
  rPackagesFile?: string;
}

export interface PluginExecution {
  argsMapping: Record<string, unknown>;
  outputDir: string;
  requirements?: Requirements;
  envVariables?: PluginInputV2[];
}

export interface PluginDefinition {
  plugin: PluginMetadata;
  runtime: PluginRuntimeV2;
  inputs: PluginInputV2[];
  outputs?: PluginOutputV2[];
  plots?: PluginPlot[];
  annotation?: AnnotationConfig;
  execution: PluginExecution;
  example?: ExampleData;
}

export interface PluginV2 {
  id: number;
  definition: PluginDefinition;
  folderPath: string;
  scriptPath: string;
  installSource: string;
  commitHash: string;
  repository: string;
  enabled: boolean;
}

export interface GenericTableEditorData {
  columns: TableColumn[];
  data?: Record<string, unknown>[];
  title?: string;
  mode: 'edit' | 'create';
}

export interface SampleAnnotationData {
  samples?: string[];
  annotation?: Annotation[];
  mode: 'edit' | 'create';
}

export interface Annotation {
  sample: string;
  condition: string;
  bioreplicate?: string;
  batch?: string;
  color?: string;
}
