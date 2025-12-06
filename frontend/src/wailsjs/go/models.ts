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

