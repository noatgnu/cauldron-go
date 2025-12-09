export namespace models {
	
	export class Config {
	    resultStoragePath: string;
	    outputDirectory: string;
	    pythonPath: string;
	    rPath: string;
	    rLibPath: string;
	    curtainBackendUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resultStoragePath = source["resultStoragePath"];
	        this.outputDirectory = source["outputDirectory"];
	        this.pythonPath = source["pythonPath"];
	        this.rPath = source["rPath"];
	        this.rLibPath = source["rLibPath"];
	        this.curtainBackendUrl = source["curtainBackendUrl"];
	    }
	}
	export class ExampleData {
	    enabled: boolean;
	    values: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new ExampleData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.values = source["values"];
	    }
	}
	export class Job {
	    id: string;
	    type: string;
	    name: string;
	    status: string;
	    progress: number;
	    command: string;
	    args: string[];
	    parameters: Record<string, any>;
	    pythonEnvPath?: string;
	    pythonEnvType?: string;
	    rEnvPath?: string;
	    rEnvType?: string;
	    outputPath: string;
	    terminalOutput: string[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    startedAt?: any;
	    // Go type: time
	    completedAt?: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.parameters = source["parameters"];
	        this.pythonEnvPath = source["pythonEnvPath"];
	        this.pythonEnvType = source["pythonEnvType"];
	        this.rEnvPath = source["rEnvPath"];
	        this.rEnvType = source["rEnvType"];
	        this.outputPath = source["outputPath"];
	        this.terminalOutput = source["terminalOutput"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JobRequest {
	    type: string;
	    name: string;
	    inputFiles: string[];
	    parameters: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new JobRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.inputFiles = source["inputFiles"];
	        this.parameters = source["parameters"];
	    }
	}
	export class PlotAxes {
	    x: string;
	    y: string;
	    colorBy?: string;
	    sizeBy?: string;
	    labels?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlotAxes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.colorBy = source["colorBy"];
	        this.sizeBy = source["sizeBy"];
	        this.labels = source["labels"];
	    }
	}
	export class PlotConfigData {
	    axes: PlotAxes;
	
	    static createFrom(source: any = {}) {
	        return new PlotConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.axes = this.convertValues(source["axes"], PlotAxes);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PlotCustomization {
	    name: string;
	    label: string;
	    type: string;
	    default?: any;
	    min?: number;
	    max?: number;
	
	    static createFrom(source: any = {}) {
	        return new PlotCustomization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.default = source["default"];
	        this.min = source["min"];
	        this.max = source["max"];
	    }
	}
	export class PluginOutput {
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginOutput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class PluginInput {
	    name: string;
	    label: string;
	    type: string;
	    required: boolean;
	    default?: any;
	    options?: string[];
	    description?: string;
	    placeholder?: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.required = source["required"];
	        this.default = source["default"];
	        this.options = source["options"];
	        this.description = source["description"];
	        this.placeholder = source["placeholder"];
	    }
	}
	export class PluginScript {
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginScript(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	    }
	}
	export class PluginConfig {
	    name: string;
	    description: string;
	    version: string;
	    author?: string;
	    runtime: string;
	    script: PluginScript;
	    inputs: PluginInput[];
	    outputs?: PluginOutput[];
	
	    static createFrom(source: any = {}) {
	        return new PluginConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.runtime = source["runtime"];
	        this.script = this.convertValues(source["script"], PluginScript);
	        this.inputs = this.convertValues(source["inputs"], PluginInput);
	        this.outputs = this.convertValues(source["outputs"], PluginOutput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Plugin {
	    id: string;
	    config: PluginConfig;
	    folderPath: string;
	    scriptPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Plugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.config = this.convertValues(source["config"], PluginConfig);
	        this.folderPath = source["folderPath"];
	        this.scriptPath = source["scriptPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class Requirements {
	    python?: string;
	    r?: string;
	    packages?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Requirements(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.python = source["python"];
	        this.r = source["r"];
	        this.packages = source["packages"];
	    }
	}
	export class PluginExecution {
	    argsMapping: Record<string, any>;
	    outputDir: string;
	    requirements?: Requirements;
	
	    static createFrom(source: any = {}) {
	        return new PluginExecution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.argsMapping = source["argsMapping"];
	        this.outputDir = source["outputDir"];
	        this.requirements = this.convertValues(source["requirements"], Requirements);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PluginPlot {
	    id: string;
	    name: string;
	    type: string;
	    component: string;
	    dataSource: string;
	    config: PlotConfigData;
	    customization: PlotCustomization[];
	
	    static createFrom(source: any = {}) {
	        return new PluginPlot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.component = source["component"];
	        this.dataSource = source["dataSource"];
	        this.config = this.convertValues(source["config"], PlotConfigData);
	        this.customization = this.convertValues(source["customization"], PlotCustomization);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PluginOutputV2 {
	    name: string;
	    path: string;
	    type: string;
	    description: string;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginOutputV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.format = source["format"];
	    }
	}
	export class VisibilityCondition {
	    field: string;
	    equals?: any;
	    equalsAny?: any[];
	
	    static createFrom(source: any = {}) {
	        return new VisibilityCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.equals = source["equals"];
	        this.equalsAny = source["equalsAny"];
	    }
	}
	export class PluginInputV2 {
	    name: string;
	    label: string;
	    type: string;
	    required: boolean;
	    default?: any;
	    options?: string[];
	    description?: string;
	    placeholder?: string;
	    accept?: string;
	    multiple?: boolean;
	    sourceFile?: string;
	    min?: number;
	    max?: number;
	    step?: number;
	    visibleWhen?: VisibilityCondition;
	
	    static createFrom(source: any = {}) {
	        return new PluginInputV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.required = source["required"];
	        this.default = source["default"];
	        this.options = source["options"];
	        this.description = source["description"];
	        this.placeholder = source["placeholder"];
	        this.accept = source["accept"];
	        this.multiple = source["multiple"];
	        this.sourceFile = source["sourceFile"];
	        this.min = source["min"];
	        this.max = source["max"];
	        this.step = source["step"];
	        this.visibleWhen = this.convertValues(source["visibleWhen"], VisibilityCondition);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PluginRuntimeV2 {
	    type: string;
	    script: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginRuntimeV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.script = source["script"];
	    }
	}
	export class PluginMetadata {
	    id: string;
	    name: string;
	    description: string;
	    version: string;
	    author?: string;
	    category: string;
	    icon?: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.author = source["author"];
	        this.category = source["category"];
	        this.icon = source["icon"];
	    }
	}
	export class PluginDefinition {
	    plugin: PluginMetadata;
	    runtime: PluginRuntimeV2;
	    inputs: PluginInputV2[];
	    outputs?: PluginOutputV2[];
	    plots?: PluginPlot[];
	    execution: PluginExecution;
	    example?: ExampleData;
	
	    static createFrom(source: any = {}) {
	        return new PluginDefinition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugin = this.convertValues(source["plugin"], PluginMetadata);
	        this.runtime = this.convertValues(source["runtime"], PluginRuntimeV2);
	        this.inputs = this.convertValues(source["inputs"], PluginInputV2);
	        this.outputs = this.convertValues(source["outputs"], PluginOutputV2);
	        this.plots = this.convertValues(source["plots"], PluginPlot);
	        this.execution = this.convertValues(source["execution"], PluginExecution);
	        this.example = this.convertValues(source["example"], ExampleData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PluginExecutionRequest {
	    pluginId: string;
	    parameters: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PluginExecutionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.parameters = source["parameters"];
	    }
	}
	export class PluginExecutionRequestV2 {
	    pluginId: string;
	    parameters: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PluginExecutionRequestV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pluginId = source["pluginId"];
	        this.parameters = source["parameters"];
	    }
	}
	
	
	
	
	
	
	
	
	export class PluginV2 {
	    definition: PluginDefinition;
	    folderPath: string;
	    scriptPath: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginV2(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.definition = this.convertValues(source["definition"], PluginDefinition);
	        this.folderPath = source["folderPath"];
	        this.scriptPath = source["scriptPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace services {
	
	export class DataFilePreview {
	    headers: string[];
	    rows: string[][];
	    totalRows: number;
	    fileType: string;
	
	    static createFrom(source: any = {}) {
	        return new DataFilePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headers = source["headers"];
	        this.rows = source["rows"];
	        this.totalRows = source["totalRows"];
	        this.fileType = source["fileType"];
	    }
	}
	export class ImportedFile {
	    ID: number;
	    Name: string;
	    Path: string;
	    Size: number;
	    ImportedAt: number;
	    FileType: string;
	    Preview: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Path = source["Path"];
	        this.Size = source["Size"];
	        this.ImportedAt = source["ImportedAt"];
	        this.FileType = source["FileType"];
	        this.Preview = source["Preview"];
	    }
	}
	export class PythonEnvironment {
	    name: string;
	    path: string;
	    type: string;
	    version: string;
	    isVirtual: boolean;
	    hasPackages: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PythonEnvironment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.version = source["version"];
	        this.isVirtual = source["isVirtual"];
	        this.hasPackages = source["hasPackages"];
	    }
	}
	export class REnvironment {
	    name: string;
	    path: string;
	    type: string;
	    version: string;
	    libPath: string;
	    hasPackages: boolean;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new REnvironment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.type = source["type"];
	        this.version = source["version"];
	        this.libPath = source["libPath"];
	        this.hasPackages = source["hasPackages"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class VirtualEnvironment {
	    ID: number;
	    Name: string;
	    Path: string;
	    BasePythonPath: string;
	    CreatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new VirtualEnvironment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Name = source["Name"];
	        this.Path = source["Path"];
	        this.BasePythonPath = source["BasePythonPath"];
	        this.CreatedAt = source["CreatedAt"];
	    }
	}

}

