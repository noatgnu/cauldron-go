package generator

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
	"github.com/noatgnu/cauldron-go/internal/parser"
)

//go:embed templates/spa/*
var spaTemplates embed.FS

type SPAConfig struct {
	PluginPath     string
	OutputDir      string
	CheckOnly      bool
	SkipCheck      bool
	NoBuild        bool
	PyodideVersion string
	WebRVersion    string
	GithubAction   bool
	ExamplesDir    string
}

type SPAGenerator struct {
	config     SPAConfig
	definition *models.PluginDefinition
	pluginDir  string
}

func NewSPAGenerator(config SPAConfig) *SPAGenerator {
	return &SPAGenerator{
		config: config,
	}
}

func (g *SPAGenerator) getPluginDir() string {
	info, err := os.Stat(g.config.PluginPath)
	if err != nil {
		return filepath.Dir(g.config.PluginPath)
	}
	if info.IsDir() {
		return g.config.PluginPath
	}
	return filepath.Dir(g.config.PluginPath)
}

func (g *SPAGenerator) isRPlugin() bool {
	if g.definition == nil {
		return false
	}
	return g.definition.Runtime.HasEnvironment("r") && !g.definition.Runtime.HasEnvironment("python")
}

func (g *SPAGenerator) isPythonPlugin() bool {
	if g.definition == nil {
		return false
	}
	return g.definition.Runtime.HasEnvironment("python")
}

func (g *SPAGenerator) CheckCompatibility() (bool, []string) {
	definition, err := parser.ParsePlugin(g.config.PluginPath)
	if err != nil {
		return false, []string{fmt.Sprintf("Failed to parse plugin: %v", err)}
	}

	pluginDir := g.getPluginDir()
	compat := CheckPyodideCompatibility(definition, pluginDir)

	return compat.Compatible, compat.Issues
}

func (g *SPAGenerator) Generate() error {
	definition, err := parser.ParsePlugin(g.config.PluginPath)
	if err != nil {
		return fmt.Errorf("failed to parse plugin: %w", err)
	}

	g.definition = definition
	g.pluginDir = g.getPluginDir()

	if !g.config.SkipCheck {
		if g.isRPlugin() {
			compat := CheckWebRCompatibility(definition, g.pluginDir)
			if !compat.Compatible {
				fmt.Println("Warning: Plugin has compatibility issues with WebR:")
				for _, issue := range compat.Issues {
					fmt.Printf("  - %s\n", issue)
				}
				fmt.Println("Continuing with generation anyway...")
			}

			if len(compat.MaybeSupport) > 0 {
				fmt.Println("Note: The following R packages may or may not be available in WebR:")
				for _, pkg := range compat.MaybeSupport {
					fmt.Printf("  - %s\n", pkg)
				}
			}
		} else {
			compat := CheckPyodideCompatibility(definition, g.pluginDir)
			if !compat.Compatible {
				fmt.Println("Warning: Plugin has compatibility issues with Pyodide:")
				for _, issue := range compat.Issues {
					fmt.Printf("  - %s\n", issue)
				}
				fmt.Println("Continuing with generation anyway...")
			}

			if len(compat.MaybeSupport) > 0 {
				fmt.Println("Note: The following packages may or may not be available in Pyodide:")
				for _, pkg := range compat.MaybeSupport {
					fmt.Printf("  - %s\n", pkg)
				}
			}
		}
	}

	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := g.generateAngularJSON(); err != nil {
		return fmt.Errorf("failed to generate angular.json: %w", err)
	}

	if err := g.generatePackageJSON(); err != nil {
		return fmt.Errorf("failed to generate package.json: %w", err)
	}

	if err := g.generateTsConfig(); err != nil {
		return fmt.Errorf("failed to generate tsconfig.json: %w", err)
	}

	if err := g.generateSrcFiles(); err != nil {
		return fmt.Errorf("failed to generate source files: %w", err)
	}

	if err := g.generateLockScript(); err != nil {
		return fmt.Errorf("failed to generate lock script: %w", err)
	}

	if err := g.copyExampleFiles(); err != nil {
		return fmt.Errorf("failed to copy example files: %w", err)
	}

	if g.config.GithubAction {
		if err := g.generateGithubWorkflow(); err != nil {
			return fmt.Errorf("failed to generate GitHub workflow: %w", err)
		}
	}

	if !g.config.NoBuild {
		fmt.Println("Running npm install...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = g.config.OutputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}

		fmt.Println("Building Angular app...")
		cmd = exec.Command("npm", "run", "build")
		cmd.Dir = g.config.OutputDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ng build failed: %w", err)
		}
	}

	return nil
}

func (g *SPAGenerator) generateAngularJSON() error {
	content := `{
  "$schema": "./node_modules/@angular/cli/lib/config/schema.json",
  "version": 1,
  "cli": {
    "packageManager": "npm"
  },
  "newProjectRoot": "projects",
  "projects": {
    "` + g.definition.Plugin.ID + `-spa": {
      "projectType": "application",
      "root": "",
      "sourceRoot": "src",
      "prefix": "app",
      "architect": {
        "build": {
          "builder": "@angular/build:application",
          "options": {
            "outputPath": "dist",
            "index": "src/index.html",
            "browser": "src/main.ts",
            "polyfills": [],
            "tsConfig": "tsconfig.app.json",
            "assets": [
              "src/favicon.ico",
              "src/assets"
            ],
            "styles": [
              "@angular/material/prebuilt-themes/azure-blue.css",
              "src/styles.scss"
            ],
            "scripts": []
          },
          "configurations": {
            "production": {
              "budgets": [
                {
                  "type": "initial",
                  "maximumWarning": "2mb",
                  "maximumError": "5mb"
                }
              ],
              "outputHashing": "all"
            },
            "development": {
              "optimization": false,
              "extractLicenses": false,
              "sourceMap": true
            }
          },
          "defaultConfiguration": "production"
        },
        "serve": {
          "builder": "@angular/build:dev-server",
          "configurations": {
            "production": {
              "buildTarget": "` + g.definition.Plugin.ID + `-spa:build:production"
            },
            "development": {
              "buildTarget": "` + g.definition.Plugin.ID + `-spa:build:development"
            }
          },
          "defaultConfiguration": "development"
        },
        "test": {
          "builder": "@angular/build:karma",
          "options": {
            "polyfills": [],
            "tsConfig": "tsconfig.spec.json",
            "karmaConfig": "karma.conf.js",
            "assets": [
              "src/favicon.ico",
              "src/assets"
            ],
            "styles": [
              "@angular/material/prebuilt-themes/azure-blue.css",
              "src/styles.scss"
            ],
            "scripts": []
          }
        }
      }
    }
  }
}`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "angular.json"), []byte(content), 0644)
}

func (g *SPAGenerator) generatePackageJSON() error {
	if g.isRPlugin() {
		return g.generatePackageJSONR()
	}
	return g.generatePackageJSONPython()
}

func (g *SPAGenerator) generatePackageJSONPython() error {
	packages := getRequiredPackages(g.definition, g.pluginDir)

	content := `{
  "name": "` + g.definition.Plugin.ID + `-spa",
  "version": "0.0.1",
  "scripts": {
    "ng": "ng",
    "start": "ng serve",
    "build": "ng build",
    "build:ci": "node scripts/generate-lock.mjs && ng build",
    "generate-lock": "node scripts/generate-lock.mjs",
    "test": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.spec.ts",
    "test:integration": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.integration.spec.ts",
    "test:all": "ng test --no-watch --no-progress --browsers=ChromeHeadless",
    "test:ci": "node scripts/generate-lock.mjs && ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.spec.ts",
    "test:ci:integration": "node scripts/generate-lock.mjs && ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.integration.spec.ts"
  },
  "private": true,
  "dependencies": {
    "@angular/animations": "^21.0.0",
    "@angular/cdk": "^21.0.1",
    "@angular/common": "^21.0.0",
    "@angular/compiler": "^21.0.0",
    "@angular/core": "^21.0.0",
    "@angular/forms": "^21.0.0",
    "@angular/material": "^21.0.1",
    "@angular/platform-browser": "^21.0.0",
    "@angular/router": "^21.0.0",
    "jszip": "^3.10.1",
    "rxjs": "~7.8.0",
    "tslib": "^2.3.0"
  },
  "devDependencies": {
    "@angular/build": "^21.0.1",
    "@angular/cli": "^21.0.1",
    "@angular/compiler-cli": "^21.0.0",
    "@types/jasmine": "~5.1.0",
    "jasmine-core": "~5.1.0",
    "karma": "~6.4.0",
    "karma-chrome-launcher": "~3.2.0",
    "karma-coverage": "~2.2.0",
    "karma-jasmine": "~5.1.0",
    "karma-jasmine-html-reporter": "~2.1.0",
    "puppeteer": "^24.0.0",
    "pyodide": "` + g.config.PyodideVersion + `",
    "typescript": "~5.9.2"
  },
  "pyodidePackages": ` + toJSONArray(packages) + `
}`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "package.json"), []byte(content), 0644)
}

func (g *SPAGenerator) generatePackageJSONR() error {
	packages := getRRequiredPackages(g.definition, g.pluginDir)

	content := `{
  "name": "` + g.definition.Plugin.ID + `-spa",
  "version": "0.0.1",
  "scripts": {
    "ng": "ng",
    "start": "ng serve",
    "build": "ng build",
    "build:ci": "ng build",
    "test": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.spec.ts",
    "test:integration": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.integration.spec.ts",
    "test:all": "ng test --no-watch --no-progress --browsers=ChromeHeadless",
    "test:ci": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.spec.ts",
    "test:ci:integration": "ng test --no-watch --no-progress --browsers=ChromeHeadless --include=src/app/app.integration.spec.ts"
  },
  "private": true,
  "dependencies": {
    "@angular/animations": "^21.0.0",
    "@angular/cdk": "^21.0.1",
    "@angular/common": "^21.0.0",
    "@angular/compiler": "^21.0.0",
    "@angular/core": "^21.0.0",
    "@angular/forms": "^21.0.0",
    "@angular/material": "^21.0.1",
    "@angular/platform-browser": "^21.0.0",
    "@angular/router": "^21.0.0",
    "jszip": "^3.10.1",
    "rxjs": "~7.8.0",
    "tslib": "^2.3.0"
  },
  "devDependencies": {
    "@angular/build": "^21.0.1",
    "@angular/cli": "^21.0.1",
    "@angular/compiler-cli": "^21.0.0",
    "@types/jasmine": "~5.1.0",
    "jasmine-core": "~5.1.0",
    "karma": "~6.4.0",
    "karma-chrome-launcher": "~3.2.0",
    "karma-coverage": "~2.2.0",
    "karma-jasmine": "~5.1.0",
    "karma-jasmine-html-reporter": "~2.1.0",
    "puppeteer": "^24.0.0",
    "typescript": "~5.9.2"
  },
  "webrPackages": ` + toJSONArray(packages) + `
}`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "package.json"), []byte(content), 0644)
}

func (g *SPAGenerator) generateTsConfig() error {
	content := `{
  "compileOnSave": false,
  "compilerOptions": {
    "strict": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "experimentalDecorators": true,
    "moduleResolution": "bundler",
    "importHelpers": true,
    "target": "ES2022",
    "module": "ES2022"
  },
  "angularCompilerOptions": {
    "enableI18nLegacyMessageIdFormat": false,
    "strictInjectionParameters": true,
    "strictInputAccessModifiers": true,
    "strictTemplates": true
  }
}`
	if err := os.WriteFile(filepath.Join(g.config.OutputDir, "tsconfig.json"), []byte(content), 0644); err != nil {
		return err
	}

	appConfig := `{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "outDir": "./out-tsc/app",
    "types": []
  },
  "include": ["src/**/*.ts"],
  "exclude": ["src/**/*.spec.ts"]
}`
	if err := os.WriteFile(filepath.Join(g.config.OutputDir, "tsconfig.app.json"), []byte(appConfig), 0644); err != nil {
		return err
	}

	specConfig := `{
  "extends": "./tsconfig.json",
  "compilerOptions": {
    "outDir": "./out-tsc/spec",
    "types": ["jasmine"]
  },
  "include": ["src/**/*.spec.ts", "src/**/*.d.ts"]
}`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "tsconfig.spec.json"), []byte(specConfig), 0644)
}

func (g *SPAGenerator) generateSrcFiles() error {
	srcDir := filepath.Join(g.config.OutputDir, "src")
	appDir := filepath.Join(srcDir, "app")
	servicesDir := filepath.Join(appDir, "services")
	embeddedDir := filepath.Join(appDir, "embedded")
	assetsDir := filepath.Join(srcDir, "assets")

	for _, dir := range []string{srcDir, appDir, servicesDir, embeddedDir, assetsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if err := g.generateIndexHTML(); err != nil {
		return err
	}

	if err := g.generateMainTS(); err != nil {
		return err
	}

	if err := g.generateStylesCSS(); err != nil {
		return err
	}

	if g.isRPlugin() {
		if err := g.generateAppComponentR(); err != nil {
			return err
		}
		if err := g.generateWebRService(); err != nil {
			return err
		}
	} else {
		if err := g.generateAppComponent(); err != nil {
			return err
		}
		if err := g.generatePyodideService(); err != nil {
			return err
		}
	}

	if err := g.generateBrowserFileHandler(); err != nil {
		return err
	}

	if err := g.generatePluginConfig(); err != nil {
		return err
	}

	if err := g.embedPluginScript(); err != nil {
		return err
	}

	if err := g.generateKarmaConfig(); err != nil {
		return err
	}

	if err := g.generateAppSpec(); err != nil {
		return err
	}

	return nil
}

func (g *SPAGenerator) generateIndexHTML() error {
	content := `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>` + g.definition.Plugin.Name + `</title>
  <base href="/">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="icon" type="image/x-icon" href="favicon.ico">
  <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500&display=swap" rel="stylesheet">
  <link href="https://fonts.googleapis.com/icon?family=Material+Icons" rel="stylesheet">
</head>
<body class="mat-typography mat-app-background">
  <app-root></app-root>
</body>
</html>`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "index.html"), []byte(content), 0644)
}

func (g *SPAGenerator) generateMainTS() error {
	content := `import { bootstrapApplication } from '@angular/platform-browser';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { AppComponent } from './app/app';

bootstrapApplication(AppComponent, {
  providers: [
    provideAnimationsAsync()
  ]
}).catch(err => console.error(err));
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "main.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateStylesCSS() error {
	content := `html, body { height: 100%; }
body { margin: 0; font-family: Roboto, "Helvetica Neue", sans-serif; }

.mat-app-background {
  background-color: #fafafa;
}

.spacer {
  flex: 1 1 auto;
}

mat-toolbar a {
  color: inherit;
}
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "styles.scss"), []byte(content), 0644)
}

func (g *SPAGenerator) generateAppComponent() error {
	tsContent := `import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { PyodideService } from './services/pyodide.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION, PLUGIN_PACKAGES } from './embedded/plugin-config';
import { PLUGIN_SCRIPT } from './embedded/plugin-script';
import { PLUGIN_MODULES } from './embedded/plugin-modules';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatToolbarModule,
    MatCardModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatExpansionModule,
    MatListModule,
    MatTooltipModule
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class AppComponent implements OnInit {
  form!: FormGroup;
  plugin = PLUGIN_DEFINITION;

  pyodideReady = signal(false);
  loading = signal(false);
  progress = signal({ stage: '', percent: 0 });
  outputs = signal<{name: string, content: string, type: string}[]>([]);
  logs = signal<string[]>([]);
  error = signal<string | null>(null);
  fileData = new Map<string, {name: string, content: string}>();

  constructor(
    private fb: FormBuilder,
    private pyodide: PyodideService,
    private fileHandler: BrowserFileHandler
  ) {}

  ngOnInit() {
    this.buildForm();
    this.initializePyodide();
  }

  private buildForm() {
    const group: Record<string, any[]> = {};
    for (const input of this.plugin.inputs) {
      const validators = input.required ? [Validators.required] : [];
      const defaultValue = (input as any)['default'] !== undefined ? (input as any)['default'] : '';
      group[input.name] = [defaultValue, validators];
    }
    this.form = this.fb.group(group);
  }

  private async initializePyodide() {
    this.loading.set(true);
    this.pyodide.progress$.subscribe(p => this.progress.set(p));
    this.pyodide.output$.subscribe(line => {
      this.logs.update(logs => [...logs, line]);
    });

    try {
      await this.pyodide.initialize(PLUGIN_PACKAGES);
      this.pyodideReady.set(true);
    } catch (err: any) {
      this.error.set('Failed to initialize Pyodide: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  async openFile(inputName: string) {
    const file = await this.fileHandler.openFileDialog('Select File');
    if (file) {
      this.fileData.set(inputName, file);
      this.form.patchValue({ [inputName]: file.name });
    }
  }

  async run() {
    if (!this.form.valid) return;

    this.loading.set(true);
    this.error.set(null);
    this.outputs.set([]);
    this.logs.set([]);

    try {
      const params = { ...this.form.value };
      for (const [key, fileInfo] of this.fileData.entries()) {
        params[key] = fileInfo;
      }
      const argsMapping = this.plugin.execution?.argsMapping || {};
      const outputDirFlag = this.plugin.execution?.outputDir;
      const inputs = this.plugin.inputs || [];
      const result = await this.pyodide.execute(PLUGIN_SCRIPT, params, PLUGIN_MODULES, argsMapping, outputDirFlag, inputs);
      this.outputs.set(result.outputs);
    } catch (err: any) {
      this.error.set('Execution failed: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  downloadOutput(output: {name: string, content: string}) {
    const blob = new Blob([output.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = output.name;
    a.click();
    URL.revokeObjectURL(url);
  }

  normalizeOptions(options: (string | {value: string, label: string})[]): {value: string, label: string}[] {
    if (!options) return [];
    return options.map(opt => {
      if (typeof opt === 'string') {
        return { value: opt, label: opt };
      }
      return opt;
    });
  }

  async downloadAll() {
    const JSZip = (await import('jszip')).default;
    const zip = new JSZip();

    for (const output of this.outputs()) {
      zip.file(output.name, output.content);
    }

    const content = await zip.generateAsync({ type: 'blob' });
    const url = URL.createObjectURL(content);
    const a = document.createElement('a');
    a.href = url;
    a.download = '` + g.definition.Plugin.ID + `-results.zip';
    a.click();
    URL.revokeObjectURL(url);
  }

  hasExample(): boolean {
    return this.plugin.example?.enabled === true;
  }

  async loadExample() {
    if (!this.plugin.example?.values) return;

    const values = this.plugin.example.values;
    const formValues: Record<string, any> = {};
    const fileInputs = new Set(this.plugin.inputs.filter((i: any) => i.type === 'file').map((i: any) => i.name));

    for (const [key, value] of Object.entries(values)) {
      if (key.endsWith('_source')) continue;

      if (typeof value === 'string' && fileInputs.has(key)) {
        const assetPath = 'assets/' + value;
        try {
          const response = await fetch(assetPath);
          if (response.ok) {
            const content = await response.text();
            const fileName = value.split('/').pop() || 'example.txt';
            this.fileData.set(key, { name: fileName, content });
            formValues[key] = fileName;
          }
        } catch (e) {
          this.error.set('Failed to load example file: ' + value);
        }
      } else {
        formValues[key] = value;
      }
    }

    this.form.patchValue(formValues);
  }
}
`

	repoUrl := g.definition.Plugin.Repository
	repoButton := ""
	if repoUrl != "" {
		repoButton = `
  <span class="spacer"></span>
  <a mat-icon-button href="` + repoUrl + `" target="_blank" rel="noopener" matTooltip="View Plugin Repository">
    <mat-icon>code</mat-icon>
  </a>`
	}

	htmlContent := `<mat-toolbar color="primary">
  <span>` + g.definition.Plugin.Name + `</span>` + repoButton + `
</mat-toolbar>

<div class="container">
  <mat-card class="main-card">
    <mat-card-header>
      <mat-card-title>` + g.definition.Plugin.Name + `</mat-card-title>
      <mat-card-subtitle>` + strings.ReplaceAll(g.definition.Plugin.Description, "`", "'") + `</mat-card-subtitle>
    </mat-card-header>

    <mat-card-content>
      @if (!pyodideReady()) {
        <div class="loading-section">
          <mat-progress-bar mode="determinate" [value]="progress().percent"></mat-progress-bar>
          <p>{{ progress().stage }}</p>
        </div>
      } @else {
        <form [formGroup]="form" class="plugin-form">
          @for (input of plugin.inputs; track input.name) {
            <div class="form-field">
              @switch (input.type) {
                @case ('file') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput readonly [formControlName]="input.name">
                  </mat-form-field>
                  <button mat-stroked-button type="button" (click)="openFile(input.name)">
                    <mat-icon>upload_file</mat-icon>
                    Browse
                  </button>
                }
                @case ('text') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput [formControlName]="input.name">
                  </mat-form-field>
                }
                @case ('number') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput type="number" [formControlName]="input.name">
                  </mat-form-field>
                }
                @case ('boolean') {
                  <mat-checkbox [formControlName]="input.name">{{ input.label }}</mat-checkbox>
                }
                @case ('select') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <mat-select [formControlName]="input.name">
                      @for (opt of normalizeOptions($any(input).options || []); track opt.value) {
                        <mat-option [value]="opt.value">{{ opt.label }}</mat-option>
                      }
                    </mat-select>
                  </mat-form-field>
                }
              }
            </div>
          }
        </form>
      }

      @if (error()) {
        <div class="error-box">
          <mat-icon>error</mat-icon>
          <span>{{ error() }}</span>
        </div>
      }

      @if (outputs().length > 0) {
        <mat-expansion-panel class="results-panel" expanded>
          <mat-expansion-panel-header>
            <mat-panel-title>Results</mat-panel-title>
          </mat-expansion-panel-header>

          <mat-list>
            @for (output of outputs(); track output.name) {
              <mat-list-item>
                <mat-icon matListItemIcon>description</mat-icon>
                <span matListItemTitle>{{ output.name }}</span>
                <button mat-icon-button (click)="downloadOutput(output)">
                  <mat-icon>download</mat-icon>
                </button>
              </mat-list-item>
            }
          </mat-list>

          <button mat-raised-button color="accent" (click)="downloadAll()">
            <mat-icon>folder_zip</mat-icon>
            Download All
          </button>
        </mat-expansion-panel>
      }

      @if (logs().length > 0) {
        <mat-expansion-panel class="logs-panel">
          <mat-expansion-panel-header>
            <mat-panel-title>Execution Log</mat-panel-title>
          </mat-expansion-panel-header>
          <pre class="log-output">{{ logs().join('\n') }}</pre>
        </mat-expansion-panel>
      }
    </mat-card-content>

    <mat-card-actions>
      @if (hasExample()) {
        <button mat-stroked-button (click)="loadExample()" [disabled]="!pyodideReady() || loading()">
          <mat-icon>folder_open</mat-icon>
          Load Example
        </button>
      }
      <button mat-raised-button color="primary"
              [disabled]="!pyodideReady() || !form.valid || loading()"
              (click)="run()">
        @if (loading()) {
          <mat-spinner diameter="20"></mat-spinner>
        } @else {
          <mat-icon>play_arrow</mat-icon>
        }
        Run
      </button>
    </mat-card-actions>
  </mat-card>
</div>
`

	scssContent := `.container {
  padding: 20px;
  max-width: 900px;
  margin: 0 auto;
}

.main-card {
  margin-top: 20px;
}

.plugin-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-field {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.full-width {
  flex: 1;
}

.loading-section {
  text-align: center;
  padding: 40px;
}

.error-box {
  background: #ffebee;
  color: #c62828;
  padding: 16px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 16px 0;
}

.results-panel, .logs-panel {
  margin-top: 16px;
}

.log-output {
  background: #263238;
  color: #aed581;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
}

mat-card-actions {
  padding: 16px;
  display: flex;
  gap: 8px;
}
`

	appDir := filepath.Join(g.config.OutputDir, "src", "app")
	if err := os.WriteFile(filepath.Join(appDir, "app.ts"), []byte(tsContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "app.html"), []byte(htmlContent), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, "app.scss"), []byte(scssContent), 0644)
}

func (g *SPAGenerator) generateAppComponentR() error {
	tsContent := `import { Component, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { WebRService } from './services/webr.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION, PLUGIN_PACKAGES, PLUGIN_PACKAGE_REPOS } from './embedded/plugin-config';
import { PLUGIN_SCRIPT } from './embedded/plugin-script';
import { PLUGIN_MODULES } from './embedded/plugin-modules';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatToolbarModule,
    MatCardModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatCheckboxModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatIconModule,
    MatExpansionModule,
    MatListModule,
    MatTooltipModule
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class AppComponent implements OnInit {
  form!: FormGroup;
  plugin = PLUGIN_DEFINITION;

  webrReady = signal(false);
  loading = signal(false);
  progress = signal({ stage: '', percent: 0 });
  outputs = signal<{name: string, content: string, type: string}[]>([]);
  logs = signal<string[]>([]);
  error = signal<string | null>(null);
  fileData = new Map<string, {name: string, content: string}>();

  constructor(
    private fb: FormBuilder,
    private webr: WebRService,
    private fileHandler: BrowserFileHandler
  ) {}

  ngOnInit() {
    this.buildForm();
    this.initializeWebR();
  }

  private buildForm() {
    const group: Record<string, any[]> = {};
    for (const input of this.plugin.inputs) {
      const validators = input.required ? [Validators.required] : [];
      const defaultValue = (input as any)['default'] !== undefined ? (input as any)['default'] : '';
      group[input.name] = [defaultValue, validators];
    }
    this.form = this.fb.group(group);
  }

  private async initializeWebR() {
    this.loading.set(true);
    this.webr.progress$.subscribe(p => this.progress.set(p));
    this.webr.output$.subscribe(line => {
      this.logs.update(logs => [...logs, line]);
    });

    try {
      await this.webr.initialize(PLUGIN_PACKAGES, PLUGIN_PACKAGE_REPOS);
      this.webrReady.set(true);
    } catch (err: any) {
      this.error.set('Failed to initialize WebR: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  async openFile(inputName: string) {
    const file = await this.fileHandler.openFileDialog('Select File');
    if (file) {
      this.fileData.set(inputName, file);
      this.form.patchValue({ [inputName]: file.name });
    }
  }

  async run() {
    if (!this.form.valid) return;

    this.loading.set(true);
    this.error.set(null);
    this.outputs.set([]);
    this.logs.set([]);

    try {
      const params = { ...this.form.value };
      for (const [key, fileInfo] of this.fileData.entries()) {
        params[key] = fileInfo;
      }
      const argsMapping = this.plugin.execution?.argsMapping || {};
      const outputDirFlag = this.plugin.execution?.outputDir;
      const inputs = this.plugin.inputs || [];
      const result = await this.webr.execute(PLUGIN_SCRIPT, params, PLUGIN_MODULES, argsMapping, outputDirFlag, inputs);
      this.outputs.set(result.outputs);
    } catch (err: any) {
      this.error.set('Execution failed: ' + err.message);
    } finally {
      this.loading.set(false);
    }
  }

  downloadOutput(output: {name: string, content: string}) {
    const blob = new Blob([output.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = output.name;
    a.click();
    URL.revokeObjectURL(url);
  }

  normalizeOptions(options: (string | {value: string, label: string})[]): {value: string, label: string}[] {
    if (!options) return [];
    return options.map(opt => {
      if (typeof opt === 'string') {
        return { value: opt, label: opt };
      }
      return opt;
    });
  }

  async downloadAll() {
    const JSZip = (await import('jszip')).default;
    const zip = new JSZip();

    for (const output of this.outputs()) {
      zip.file(output.name, output.content);
    }

    const content = await zip.generateAsync({ type: 'blob' });
    const url = URL.createObjectURL(content);
    const a = document.createElement('a');
    a.href = url;
    a.download = '` + g.definition.Plugin.ID + `-results.zip';
    a.click();
    URL.revokeObjectURL(url);
  }

  hasExample(): boolean {
    return this.plugin.example?.enabled === true;
  }

  async loadExample() {
    if (!this.plugin.example?.values) return;

    const values = this.plugin.example.values;
    const formValues: Record<string, any> = {};
    const fileInputs = new Set(this.plugin.inputs.filter((i: any) => i.type === 'file').map((i: any) => i.name));

    for (const [key, value] of Object.entries(values)) {
      if (key.endsWith('_source')) continue;

      if (typeof value === 'string' && fileInputs.has(key)) {
        const assetPath = 'assets/' + value;
        try {
          const response = await fetch(assetPath);
          if (response.ok) {
            const content = await response.text();
            const fileName = value.split('/').pop() || 'example.txt';
            this.fileData.set(key, { name: fileName, content });
            formValues[key] = fileName;
          }
        } catch (e) {
          this.error.set('Failed to load example file: ' + value);
        }
      } else {
        formValues[key] = value;
      }
    }

    this.form.patchValue(formValues);
  }
}
`

	repoUrl := g.definition.Plugin.Repository
	repoButton := ""
	if repoUrl != "" {
		repoButton = `
  <span class="spacer"></span>
  <a mat-icon-button href="` + repoUrl + `" target="_blank" rel="noopener" matTooltip="View Plugin Repository">
    <mat-icon>code</mat-icon>
  </a>`
	}

	htmlContent := `<mat-toolbar color="primary">
  <span>` + g.definition.Plugin.Name + `</span>` + repoButton + `
</mat-toolbar>

<div class="container">
  <mat-card class="main-card">
    <mat-card-header>
      <mat-card-title>` + g.definition.Plugin.Name + `</mat-card-title>
      <mat-card-subtitle>` + strings.ReplaceAll(g.definition.Plugin.Description, "`", "'") + `</mat-card-subtitle>
    </mat-card-header>

    <mat-card-content>
      @if (!webrReady()) {
        <div class="loading-section">
          <mat-progress-bar mode="determinate" [value]="progress().percent"></mat-progress-bar>
          <p>{{ progress().stage }}</p>
        </div>
      } @else {
        <form [formGroup]="form" class="plugin-form">
          @for (input of plugin.inputs; track input.name) {
            <div class="form-field">
              @switch (input.type) {
                @case ('file') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput readonly [formControlName]="input.name">
                  </mat-form-field>
                  <button mat-stroked-button type="button" (click)="openFile(input.name)">
                    <mat-icon>upload_file</mat-icon>
                    Browse
                  </button>
                }
                @case ('text') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput [formControlName]="input.name">
                  </mat-form-field>
                }
                @case ('number') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <input matInput type="number" [formControlName]="input.name">
                  </mat-form-field>
                }
                @case ('boolean') {
                  <mat-checkbox [formControlName]="input.name">{{ input.label }}</mat-checkbox>
                }
                @case ('select') {
                  <mat-form-field appearance="outline" class="full-width">
                    <mat-label>{{ input.label }}</mat-label>
                    <mat-select [formControlName]="input.name">
                      @for (opt of normalizeOptions($any(input).options || []); track opt.value) {
                        <mat-option [value]="opt.value">{{ opt.label }}</mat-option>
                      }
                    </mat-select>
                  </mat-form-field>
                }
              }
            </div>
          }
        </form>
      }

      @if (error()) {
        <div class="error-box">
          <mat-icon>error</mat-icon>
          <span>{{ error() }}</span>
        </div>
      }

      @if (outputs().length > 0) {
        <mat-expansion-panel class="results-panel" expanded>
          <mat-expansion-panel-header>
            <mat-panel-title>Results</mat-panel-title>
          </mat-expansion-panel-header>

          <mat-list>
            @for (output of outputs(); track output.name) {
              <mat-list-item>
                <mat-icon matListItemIcon>description</mat-icon>
                <span matListItemTitle>{{ output.name }}</span>
                <button mat-icon-button (click)="downloadOutput(output)">
                  <mat-icon>download</mat-icon>
                </button>
              </mat-list-item>
            }
          </mat-list>

          <button mat-raised-button color="accent" (click)="downloadAll()">
            <mat-icon>folder_zip</mat-icon>
            Download All
          </button>
        </mat-expansion-panel>
      }

      @if (logs().length > 0) {
        <mat-expansion-panel class="logs-panel">
          <mat-expansion-panel-header>
            <mat-panel-title>Execution Log</mat-panel-title>
          </mat-expansion-panel-header>
          <pre class="log-output">{{ logs().join('\n') }}</pre>
        </mat-expansion-panel>
      }
    </mat-card-content>

    <mat-card-actions>
      @if (hasExample()) {
        <button mat-stroked-button (click)="loadExample()" [disabled]="!webrReady() || loading()">
          <mat-icon>folder_open</mat-icon>
          Load Example
        </button>
      }
      <button mat-raised-button color="primary"
              [disabled]="!webrReady() || !form.valid || loading()"
              (click)="run()">
        @if (loading()) {
          <mat-spinner diameter="20"></mat-spinner>
        } @else {
          <mat-icon>play_arrow</mat-icon>
        }
        Run
      </button>
    </mat-card-actions>
  </mat-card>
</div>
`

	scssContent := `.container {
  padding: 20px;
  max-width: 900px;
  margin: 0 auto;
}

.main-card {
  margin-top: 20px;
}

.plugin-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-field {
  display: flex;
  gap: 8px;
  align-items: flex-start;
}

.full-width {
  flex: 1;
}

.loading-section {
  text-align: center;
  padding: 40px;
}

.error-box {
  background: #ffebee;
  color: #c62828;
  padding: 16px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 16px 0;
}

.results-panel, .logs-panel {
  margin-top: 16px;
}

.log-output {
  background: #263238;
  color: #aed581;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
}

mat-card-actions {
  padding: 16px;
  display: flex;
  gap: 8px;
}
`

	appDir := filepath.Join(g.config.OutputDir, "src", "app")
	if err := os.WriteFile(filepath.Join(appDir, "app.ts"), []byte(tsContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, "app.html"), []byte(htmlContent), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, "app.scss"), []byte(scssContent), 0644)
}

func (g *SPAGenerator) generatePyodideService() error {
	content := `import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

declare var loadPyodide: any;

export interface ExecutionResult {
  outputs: {name: string, content: string, type: string}[];
  stdout: string;
  stderr: string;
}

const PYODIDE_VERSION = '` + g.config.PyodideVersion + `';
const PYODIDE_CDN = 'https://cdn.jsdelivr.net/pyodide/v' + PYODIDE_VERSION + '/full/';
const LOCK_FILE_PATH = 'assets/pyodide-lock.json';

@Injectable({ providedIn: 'root' })
export class PyodideService {
  private pyodide: any = null;

  progress$ = new Subject<{stage: string, percent: number}>();
  output$ = new Subject<string>();

  async initialize(packages: string[]): Promise<void> {
    this.progress$.next({ stage: 'Loading Pyodide...', percent: 10 });

    const script = document.createElement('script');
    script.src = PYODIDE_CDN + 'pyodide.js';
    document.head.appendChild(script);

    await new Promise<void>((resolve) => {
      script.onload = () => resolve();
    });

    const isTestEnvironment = typeof (window as any).jasmine !== 'undefined';
    const lockFileExists = !isTestEnvironment && await this.checkLockFile();

    if (lockFileExists) {
      this.progress$.next({ stage: 'Loading packages from lock file...', percent: 30 });
      try {
        const lockFileURL = new URL(LOCK_FILE_PATH, window.location.href).href;
        this.pyodide = await (window as any).loadPyodide({
          indexURL: PYODIDE_CDN,
          lockFileURL: lockFileURL
        });

        this.progress$.next({ stage: 'Loading packages...', percent: 60 });
        const packageNames = packages.map(p => p.split(/[<>=]/)[0]);
        await this.pyodide.loadPackage(packageNames);

        const testPkg = packageNames[0];
        try {
          this.pyodide.pyimport(testPkg);
          this.progress$.next({ stage: 'Ready', percent: 100 });
          return;
        } catch {
          console.warn('Package ' + testPkg + ' not available, falling back to micropip');
          this.pyodide = null;
        }
      } catch (e) {
        console.warn('Failed to load from lock file, installing fresh:', e);
        this.pyodide = null;
      }
    }

    if (!this.pyodide) {
      this.pyodide = await (window as any).loadPyodide({
        indexURL: PYODIDE_CDN
      });
    }

    this.progress$.next({ stage: 'Installing packages...', percent: 40 });

    await this.pyodide.loadPackage('micropip');
    const micropip = this.pyodide.pyimport('micropip');

    const total = packages.length;
    for (let i = 0; i < packages.length; i++) {
      const pkg = packages[i];
      const pkgName = pkg.split(/[<>=]/)[0];
      this.progress$.next({
        stage: 'Installing ' + pkgName + '...',
        percent: 40 + (i / total) * 50
      });
      try {
        await micropip.install(pkg);
      } catch (e) {
        console.warn('Failed to install ' + pkg + ':', e);
      }
    }

    this.progress$.next({ stage: 'Ready', percent: 100 });
  }

  private async checkLockFile(): Promise<boolean> {
    try {
      const response = await fetch(LOCK_FILE_PATH, { method: 'HEAD' });
      return response.ok;
    } catch {
      return false;
    }
  }

  async execute(script: string, params: Record<string, any>, modules: Record<string, string> = {}, argsMapping?: Record<string, any>, outputDirFlag?: string, inputs?: any[]): Promise<ExecutionResult> {
    const outputs: {name: string, content: string, type: string}[] = [];
    let stdout = '';
    let stderr = '';

    this.pyodide.setStdout({
      batched: (text: string) => {
        stdout += text + '\\n';
        this.output$.next(text);
      }
    });

    this.pyodide.setStderr({
      batched: (text: string) => {
        stderr += text + '\\n';
        this.output$.next('[stderr] ' + text);
      }
    });

    const fs = this.pyodide.FS;

    for (const [moduleName, moduleContent] of Object.entries(modules)) {
      fs.writeFile('/' + moduleName + '.py', moduleContent);
    }

    try { fs.mkdir('/input'); } catch {}
    try { fs.mkdir('/output'); } catch {}

    for (const [key, value] of Object.entries(params)) {
      if (value && typeof value === 'object' && 'name' in value && 'content' in value) {
        const filePath = '/input/' + value.name;
        fs.writeFile(filePath, value.content);
        params[key] = filePath;
      }
    }

    const args = this.buildArgs(params, argsMapping || {}, outputDirFlag, inputs);

    this.pyodide.globals.set('__params__', this.pyodide.toPy(params));

    fs.writeFile('/plugin_script.py', script);

    const wrappedScript = 'import sys\n' +
      'sys.argv = ' + JSON.stringify(['script.py', ...args]) + '\n' +
      'def __run_plugin__():\n' +
      '    try:\n' +
      '        exec(open("/plugin_script.py").read(), globals())\n' +
      '    except SystemExit as e:\n' +
      '        if e.code != 0:\n' +
      '            raise\n' +
      '__run_plugin__()\n';

    await this.pyodide.runPythonAsync(wrappedScript);

    try {
      const outputFiles = fs.readdir('/output');
      for (const file of outputFiles) {
        if (file === '.' || file === '..') continue;
        const content = fs.readFile('/output/' + file, { encoding: 'utf8' });
        outputs.push({
          name: file,
          content: content,
          type: this.getFileType(file)
        });
      }
    } catch {}

    return { outputs, stdout, stderr };
  }

  private buildArgs(params: Record<string, any>, argsMapping: Record<string, any>, outputDirFlag?: string, inputs?: any[]): string[] {
    const args: string[] = [];

    const inputMap = new Map<string, any>();
    if (inputs) {
      for (const input of inputs) {
        inputMap.set(input.name, input);
      }
    }

    for (const [paramName, value] of Object.entries(params)) {
      const mapping = argsMapping[paramName];
      if (!mapping) continue;

      const input = inputMap.get(paramName);
      const inputType = input?.type;

      let flag: string;
      let transform: string | undefined;
      let when: string | undefined;
      let fixedValue: string | undefined;
      let passAsValue = false;

      if (typeof mapping === 'string') {
        flag = mapping;
      } else {
        flag = mapping.flag;
        transform = mapping.transform;
        when = mapping.when;
        fixedValue = mapping.value;
        passAsValue = mapping.passAsValue === true;
      }

      if (!flag) continue;

      if (when !== undefined) {
        const shouldInclude = this.evaluateCondition(value, when);
        if (!shouldInclude) continue;
        args.push(flag);
        continue;
      }

      if (inputType === 'boolean' && fixedValue === undefined) {
        const boolVal = value === true || value === 'true';
        if (passAsValue) {
          args.push(flag, String(boolVal));
        } else if (boolVal) {
          args.push(flag);
        }
        continue;
      }

      if (value === undefined || value === null || value === '') continue;

      if (fixedValue !== undefined) {
        args.push(flag, fixedValue);
        continue;
      }

      const transformedValue = this.transformValue(value, transform);
      args.push(flag, transformedValue);
    }

    if (outputDirFlag) {
      args.push(outputDirFlag, '/output');
    }

    return args;
  }

  private evaluateCondition(value: any, condition: string): boolean {
    const valueStr = String(value ?? '');

    switch (condition) {
      case 'true':
        return value === true || valueStr === 'true' || valueStr === '1';
      case 'false':
        return value === false || valueStr === 'false' || valueStr === '0';
      case 'not-empty':
        return valueStr !== '';
      case 'empty':
        return valueStr === '';
      default:
        return valueStr === condition;
    }
  }

  private transformValue(value: any, transform?: string): string {
    if (!transform) {
      return String(value);
    }

    switch (transform) {
      case 'comma-join':
        if (Array.isArray(value)) {
          return value.join(',');
        }
        return String(value);

      case 'space-join':
        if (Array.isArray(value)) {
          return value.join(' ');
        }
        return String(value);

      case 'json-encode':
        return JSON.stringify(value);

      default:
        return String(value);
    }
  }

  private getFileType(filename: string): string {
    const ext = filename.split('.').pop()?.toLowerCase();
    switch (ext) {
      case 'csv': case 'tsv': case 'txt': return 'text';
      case 'png': case 'jpg': case 'jpeg': case 'svg': return 'image';
      case 'json': return 'json';
      default: return 'binary';
    }
  }
}
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "services", "pyodide.service.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateWebRService() error {
	content := `import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

export interface ExecutionResult {
  outputs: {name: string, content: string, type: string}[];
  stdout: string;
  stderr: string;
}

const WEBR_VERSION = '` + g.config.WebRVersion + `';
const WEBR_CDN = 'https://webr.r-wasm.org/v' + WEBR_VERSION + '/';
const BIOC_REPO = 'https://bioconductor.r-universe.dev';
const RUNIVERSE_REPO = 'https://r-universe.dev';

@Injectable({ providedIn: 'root' })
export class WebRService {
  private webR: any = null;

  progress$ = new Subject<{stage: string, percent: number}>();
  output$ = new Subject<string>();

  async initialize(packages: string[], packageRepos: {name: string, repo: string}[] = []): Promise<void> {
    this.progress$.next({ stage: 'Loading WebR...', percent: 10 });

    const { WebR } = await import(WEBR_CDN + 'webr.mjs');

    this.progress$.next({ stage: 'Initializing R...', percent: 20 });

    this.webR = new WebR();
    await this.webR.init();

    this.progress$.next({ stage: 'Installing packages...', percent: 40 });

    const repoMap = new Map<string, string>();
    for (const pr of packageRepos) {
      repoMap.set(pr.name, pr.repo);
    }

    const defaultRepos = ['https://repo.r-wasm.org'];
    const total = packages.length;
    for (let i = 0; i < packages.length; i++) {
      const pkg = packages[i];
      this.progress$.next({
        stage: 'Installing ' + pkg + '...',
        percent: 40 + (i / total) * 50
      });

      const pkgRepo = repoMap.get(pkg);
      const repos = pkgRepo ? [pkgRepo, ...defaultRepos] : defaultRepos;

      try {
        await this.webR.installPackages([pkg], { repos: repos, quiet: false });
      } catch (e) {
        console.warn('Failed to install ' + pkg + ':', e);
      }
    }

    this.progress$.next({ stage: 'Ready', percent: 100 });
  }

  async execute(script: string, params: Record<string, any>, modules: Record<string, string> = {}, argsMapping?: Record<string, any>, outputDirFlag?: string, inputs?: any[]): Promise<ExecutionResult> {
    const outputs: {name: string, content: string, type: string}[] = [];
    let stdout = '';
    let stderr = '';

    const shelter = await new this.webR.Shelter();

    try {
      await this.webR.FS.mkdir('/input');
    } catch {}
    try {
      await this.webR.FS.mkdir('/output');
    } catch {}

    const encoder = new TextEncoder();
    for (const [key, value] of Object.entries(params)) {
      if (value && typeof value === 'object' && 'name' in value && 'content' in value) {
        const filePath = '/input/' + value.name;
        await this.webR.FS.writeFile(filePath, encoder.encode(value.content));
        params[key] = filePath;
      }
    }

    const args = this.buildArgs(params, argsMapping || {}, outputDirFlag, inputs);

    await this.webR.FS.writeFile('/plugin_script.R', encoder.encode(script));

    const argsVector = 'c("Rscript", ' + args.map(a => '"' + a.replace(/"/g, '\\\\"') + '"').join(', ') + ')';
    const wrappedScript = 'commandArgs <- function(trailingOnly = FALSE) { ' +
      'args <- ' + argsVector + '; ' +
      'if (trailingOnly) args[-1] else args ' +
      '}; ' +
      'tryCatch({ ' +
      'source("/plugin_script.R", local = FALSE); ' +
      '}, error = function(e) { ' +
      'message("Error: ", conditionMessage(e)); ' +
      'invisible(NULL) ' +
      '})';

    const result = await shelter.captureR(wrappedScript, {
      withAutoprint: true,
      captureStreams: true,
      captureConditions: false
    });

    if (result.output) {
      for (const item of result.output) {
        if (item.type === 'stdout') {
          stdout += item.data + '\n';
          this.output$.next(item.data);
        } else if (item.type === 'stderr') {
          stderr += item.data + '\n';
          this.output$.next('[stderr] ' + item.data);
        }
      }
    }

    try {
      const outputFiles = await this.webR.evalRString('paste(list.files("/output"), collapse = "\\n")');
      const files = outputFiles.split('\n').filter((f: string) => f.trim() !== '');

      for (const file of files) {
        try {
          const content = await this.webR.FS.readFile('/output/' + file);
          const textContent = new TextDecoder().decode(content);
          outputs.push({
            name: file,
            content: textContent,
            type: this.getFileType(file)
          });
        } catch (e) {
          console.warn('Failed to read output file:', file, e);
        }
      }
    } catch (e) {
      console.warn('Failed to list output files:', e);
    }

    shelter.purge();

    return { outputs, stdout, stderr };
  }

  private buildArgs(params: Record<string, any>, argsMapping: Record<string, any>, outputDirFlag?: string, inputs?: any[]): string[] {
    const args: string[] = [];

    const inputMap = new Map<string, any>();
    if (inputs) {
      for (const input of inputs) {
        inputMap.set(input.name, input);
      }
    }

    for (const [paramName, value] of Object.entries(params)) {
      const mapping = argsMapping[paramName];
      if (!mapping) continue;

      const input = inputMap.get(paramName);
      const inputType = input?.type;

      let flag: string;
      let transform: string | undefined;
      let when: string | undefined;
      let fixedValue: string | undefined;
      let passAsValue = false;

      if (typeof mapping === 'string') {
        flag = mapping;
      } else {
        flag = mapping.flag;
        transform = mapping.transform;
        when = mapping.when;
        fixedValue = mapping.value;
        passAsValue = mapping.passAsValue === true;
      }

      if (!flag) continue;

      if (when !== undefined) {
        const shouldInclude = this.evaluateCondition(value, when);
        if (!shouldInclude) continue;
        args.push(flag);
        continue;
      }

      if (inputType === 'boolean' && fixedValue === undefined) {
        const boolVal = value === true || value === 'true';
        if (passAsValue) {
          args.push(flag, String(boolVal));
        } else if (boolVal) {
          args.push(flag);
        }
        continue;
      }

      if (value === undefined || value === null || value === '') continue;

      if (fixedValue !== undefined) {
        args.push(flag, fixedValue);
        continue;
      }

      const transformedValue = this.transformValue(value, transform);
      args.push(flag, transformedValue);
    }

    if (outputDirFlag) {
      args.push(outputDirFlag, '/output');
    }

    return args;
  }

  private evaluateCondition(value: any, condition: string): boolean {
    const valueStr = String(value ?? '');

    switch (condition) {
      case 'true':
        return value === true || valueStr === 'true' || valueStr === '1';
      case 'false':
        return value === false || valueStr === 'false' || valueStr === '0';
      case 'not-empty':
        return valueStr !== '';
      case 'empty':
        return valueStr === '';
      default:
        return valueStr === condition;
    }
  }

  private transformValue(value: any, transform?: string): string {
    if (!transform) {
      return String(value);
    }

    switch (transform) {
      case 'comma-join':
        if (Array.isArray(value)) {
          return value.join(',');
        }
        return String(value);

      case 'space-join':
        if (Array.isArray(value)) {
          return value.join(' ');
        }
        return String(value);

      case 'json-encode':
        return JSON.stringify(value);

      default:
        return String(value);
    }
  }

  private getFileType(filename: string): string {
    const ext = filename.split('.').pop()?.toLowerCase();
    switch (ext) {
      case 'csv': case 'tsv': case 'txt': return 'text';
      case 'png': case 'jpg': case 'jpeg': case 'svg': return 'image';
      case 'json': return 'json';
      case 'rds': case 'rda': case 'rdata': return 'binary';
      default: return 'binary';
    }
  }
}
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "services", "webr.service.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateBrowserFileHandler() error {
	content := `import { Injectable } from '@angular/core';

export interface FileData {
  name: string;
  content: string;
}

@Injectable({ providedIn: 'root' })
export class BrowserFileHandler {
  private virtualFS = new Map<string, string>();

  async openFileDialog(title: string, accept?: string): Promise<FileData | null> {
    return new Promise((resolve) => {
      const input = document.createElement('input');
      input.type = 'file';
      if (accept) input.accept = accept;

      input.onchange = async () => {
        const file = input.files?.[0];
        if (!file) {
          resolve(null);
          return;
        }

        const content = await file.text();
        resolve({ name: file.name, content });
      };

      input.click();
    });
  }

  async readFile(path: string): Promise<string> {
    return this.virtualFS.get(path) || '';
  }

  writeFile(path: string, content: string): void {
    this.virtualFS.set(path, content);
  }
}
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "services", "browser-file-handler.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generatePluginConfig() error {
	definitionJSON, err := json.MarshalIndent(g.definition, "", "  ")
	if err != nil {
		return err
	}

	var packages []string
	var packageRepos []map[string]string
	if g.isRPlugin() {
		packages = getRRequiredPackages(g.definition, g.pluginDir)
		availability := CheckWebRPackageAvailabilityWithRepos(packages)
		for _, pkg := range availability {
			if pkg.Available {
				packageRepos = append(packageRepos, map[string]string{
					"name": pkg.Name,
					"repo": pkg.RepoURL,
				})
			} else {
				fmt.Printf("  Warning: Package %s has no WebR binary available\n", pkg.Name)
			}
		}
	} else {
		packages = getRequiredPackages(g.definition, g.pluginDir)
	}
	if packages == nil {
		packages = []string{}
	}
	packagesJSON, err := json.Marshal(packages)
	if err != nil {
		return err
	}

	if packageRepos == nil {
		packageRepos = []map[string]string{}
	}
	packageReposJSON, err := json.Marshal(packageRepos)
	if err != nil {
		return err
	}

	content := `export const PLUGIN_DEFINITION = ` + string(definitionJSON) + `;

export const PLUGIN_PACKAGES: string[] = ` + string(packagesJSON) + `;

export const PLUGIN_PACKAGE_REPOS: {name: string, repo: string}[] = ` + string(packageReposJSON) + `;
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "embedded", "plugin-config.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) embedPluginScript() error {
	scriptPath := g.definition.Runtime.GetEntrypoint()
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(g.pluginDir, scriptPath)
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin script: %w", err)
	}

	escapedContent := strings.ReplaceAll(string(scriptContent), "\\", "\\\\")
	escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")
	escapedContent = strings.ReplaceAll(escapedContent, "${", "\\${")

	content := "export const PLUGIN_SCRIPT = `" + escapedContent + "`;\n"
	if err := os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "embedded", "plugin-script.ts"), []byte(content), 0644); err != nil {
		return err
	}

	var modules map[string]string
	if g.isRPlugin() {
		modules, err = g.findLocalRModules(scriptPath)
	} else {
		modules, err = g.findLocalModules(scriptPath)
	}
	if err != nil {
		return fmt.Errorf("failed to find local modules: %w", err)
	}

	return g.embedAdditionalModules(modules)
}

func (g *SPAGenerator) findLocalModules(entrypointPath string) (map[string]string, error) {
	modules := make(map[string]string)

	entrypointContent, err := os.ReadFile(entrypointPath)
	if err != nil {
		return nil, err
	}

	localImports := extractLocalImports(string(entrypointContent))

	for _, importName := range localImports {
		modulePath := filepath.Join(g.pluginDir, importName+".py")
		if _, err := os.Stat(modulePath); err == nil {
			moduleContent, err := os.ReadFile(modulePath)
			if err != nil {
				continue
			}
			modules[importName] = string(moduleContent)

			subImports := extractLocalImports(string(moduleContent))
			for _, subImport := range subImports {
				subModulePath := filepath.Join(g.pluginDir, subImport+".py")
				if _, err := os.Stat(subModulePath); err == nil {
					if _, exists := modules[subImport]; !exists {
						subContent, err := os.ReadFile(subModulePath)
						if err != nil {
							continue
						}
						modules[subImport] = string(subContent)
					}
				}
			}
		}
	}

	return modules, nil
}

func extractLocalImports(content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "from ") && strings.Contains(line, " import ") {
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 2 {
				moduleName := parts[1]
				if !strings.Contains(moduleName, ".") {
					imports = append(imports, moduleName)
				}
			}
		} else if strings.HasPrefix(line, "import ") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) >= 2 {
				moduleName := strings.Split(parts[1], ",")[0]
				moduleName = strings.Split(moduleName, " ")[0]
				moduleName = strings.TrimSpace(moduleName)
				if !strings.Contains(moduleName, ".") {
					imports = append(imports, moduleName)
				}
			}
		}
	}

	return imports
}

func (g *SPAGenerator) findLocalRModules(entrypointPath string) (map[string]string, error) {
	modules := make(map[string]string)

	entrypointContent, err := os.ReadFile(entrypointPath)
	if err != nil {
		return nil, err
	}

	localSources := extractRSourceCalls(string(entrypointContent))

	for _, sourceName := range localSources {
		modulePath := filepath.Join(g.pluginDir, sourceName)
		if _, err := os.Stat(modulePath); err == nil {
			moduleContent, err := os.ReadFile(modulePath)
			if err != nil {
				continue
			}
			baseName := strings.TrimSuffix(sourceName, ".R")
			modules[baseName] = string(moduleContent)

			subSources := extractRSourceCalls(string(moduleContent))
			for _, subSource := range subSources {
				subModulePath := filepath.Join(g.pluginDir, subSource)
				if _, err := os.Stat(subModulePath); err == nil {
					subBaseName := strings.TrimSuffix(subSource, ".R")
					if _, exists := modules[subBaseName]; !exists {
						subContent, err := os.ReadFile(subModulePath)
						if err != nil {
							continue
						}
						modules[subBaseName] = string(subContent)
					}
				}
			}
		}
	}

	return modules, nil
}

func extractRSourceCalls(content string) []string {
	var sources []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "source(") || strings.Contains(line, " source(") {
			start := strings.Index(line, "source(")
			if start >= 0 {
				rest := line[start+7:]
				if idx := strings.Index(rest, ")"); idx > 0 {
					arg := rest[:idx]
					arg = strings.Trim(arg, "\"' ")
					if strings.HasSuffix(arg, ".R") {
						sources = append(sources, arg)
					}
				}
			}
		}
	}

	return sources
}

func (g *SPAGenerator) embedAdditionalModules(modules map[string]string) error {
	var builder strings.Builder
	builder.WriteString("export const PLUGIN_MODULES: Record<string, string> = {\n")

	for name, content := range modules {
		escapedContent := strings.ReplaceAll(content, "\\", "\\\\")
		escapedContent = strings.ReplaceAll(escapedContent, "`", "\\`")
		escapedContent = strings.ReplaceAll(escapedContent, "${", "\\${")
		builder.WriteString(fmt.Sprintf("  '%s': `%s`,\n", name, escapedContent))
	}

	builder.WriteString("};\n")

	return os.WriteFile(
		filepath.Join(g.config.OutputDir, "src", "app", "embedded", "plugin-modules.ts"),
		[]byte(builder.String()),
		0644,
	)
}

func (g *SPAGenerator) generateGithubWorkflow() error {
	workflowDir := filepath.Join(g.config.OutputDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	hasExample := g.definition.Example != nil && g.definition.Example.Enabled && g.hasExampleFileInputs()
	integrationTestStep := ""
	if hasExample {
		integrationTestStep = `
      - name: Run integration tests with example
        run: npm run test:ci:integration
        continue-on-error: false
        env:
          CHROME_BIN: /usr/bin/google-chrome
`
	}

	content := `name: Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Run unit tests
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Generate lock file and build
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	return os.WriteFile(filepath.Join(workflowDir, "deploy.yml"), []byte(content), 0644)
}

func GeneratePluginWorkflow(pluginDir string, pyodideVersion string) error {
	return GeneratePluginWorkflowWithVersions(pluginDir, pyodideVersion, "0.4.2")
}

func GeneratePluginWorkflowWithVersions(pluginDir string, pyodideVersion string, webrVersion string) error {
	pluginPath := filepath.Join(pluginDir, "plugin.yaml")
	definition, err := parser.ParsePlugin(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to parse plugin: %w", err)
	}

	isR := definition.Runtime.HasEnvironment("r") && !definition.Runtime.HasEnvironment("python")
	hasExample := definition.Example != nil && definition.Example.Enabled && hasExampleFileInputs(definition)

	workflowDir := filepath.Join(pluginDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return err
	}

	integrationTestStep := ""
	if hasExample {
		integrationTestStep = `
      - name: Run integration tests with example
        working-directory: spa
        run: npm run test:ci:integration
        continue-on-error: false
        env:
          CHROME_BIN: /usr/bin/google-chrome
`
	}

	var content string
	if isR {
		content = `name: Build SPA and Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

env:
  CAULDRON_VERSION: "master"
  WEBR_VERSION: "` + webrVersion + `"

jobs:
  build-spa:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout plugin
        uses: actions/checkout@v4

      - name: Checkout CauldronGO
        uses: actions/checkout@v4
        with:
          repository: noatgnu/cauldron-go
          ref: ${{ env.CAULDRON_VERSION }}
          path: cauldron-go

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Build plugin-to-spa
        working-directory: cauldron-go
        run: go build -o ../plugin-to-spa ./cmd/plugin-to-spa/

      - name: Generate SPA
        run: |
          ./plugin-to-spa \
            --plugin ./plugin.yaml \
            --output ./spa \
            --skip-check \
            --no-build \
            --webr-version ${{ env.WEBR_VERSION }} \
            --examples-dir ./examples:./cauldron-go/examples

      - name: Install SPA dependencies
        working-directory: spa
        run: npm install

      - name: Run unit tests
        working-directory: spa
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Build SPA
        working-directory: spa
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './spa/dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build-spa
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	} else {
		content = `name: Build SPA and Deploy to GitHub Pages

on:
  push:
    branches: [ main, master ]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: true

env:
  CAULDRON_VERSION: "master"
  PYODIDE_VERSION: "` + pyodideVersion + `"

jobs:
  build-spa:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout plugin
        uses: actions/checkout@v4

      - name: Checkout CauldronGO
        uses: actions/checkout@v4
        with:
          repository: noatgnu/cauldron-go
          ref: ${{ env.CAULDRON_VERSION }}
          path: cauldron-go

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Build plugin-to-spa
        working-directory: cauldron-go
        run: go build -o ../plugin-to-spa ./cmd/plugin-to-spa/

      - name: Generate SPA
        run: |
          ./plugin-to-spa \
            --plugin ./plugin.yaml \
            --output ./spa \
            --skip-check \
            --no-build \
            --pyodide-version ${{ env.PYODIDE_VERSION }} \
            --examples-dir ./examples:./cauldron-go/examples

      - name: Install SPA dependencies
        working-directory: spa
        run: npm install

      - name: Run unit tests
        working-directory: spa
        run: npm run test:ci
        env:
          CHROME_BIN: /usr/bin/google-chrome
` + integrationTestStep + `
      - name: Generate lock file and build SPA
        working-directory: spa
        run: npm run build:ci -- --base-href /${{ github.event.repository.name }}/

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: './spa/dist/browser'

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build-spa
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
`
	}
	return os.WriteFile(filepath.Join(workflowDir, "deploy-spa.yml"), []byte(content), 0644)
}

func (g *SPAGenerator) generateLockScript() error {
	if g.isRPlugin() {
		return nil
	}

	scriptsDir := filepath.Join(g.config.OutputDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	packages := getRequiredPackages(g.definition, g.pluginDir)
	packagesJSON, _ := json.Marshal(packages)

	content := `import { loadPyodide } from 'pyodide';
import { writeFileSync, mkdirSync, existsSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PACKAGES = ` + string(packagesJSON) + `;
const LOCK_FILE = join(__dirname, '..', 'src', 'assets', 'pyodide-lock.json');

async function generateLock() {
  console.log('Loading Pyodide...');
  const pyodide = await loadPyodide();

  console.log('Loading micropip...');
  await pyodide.loadPackage('micropip');
  const micropip = pyodide.pyimport('micropip');

  console.log('Installing packages:', PACKAGES);
  for (const pkg of PACKAGES) {
    console.log('  Installing ' + pkg + '...');
    try {
      await micropip.install(pkg);
    } catch (e) {
      console.warn('  Warning: Failed to install ' + pkg + ':', e.message);
    }
  }

  console.log('Generating lock file...');
  const lockJson = micropip.freeze();

  const assetsDir = dirname(LOCK_FILE);
  if (!existsSync(assetsDir)) {
    mkdirSync(assetsDir, { recursive: true });
  }

  writeFileSync(LOCK_FILE, lockJson);
  console.log('Lock file generated:', LOCK_FILE);
}

generateLock().catch(console.error);
`
	return os.WriteFile(filepath.Join(scriptsDir, "generate-lock.mjs"), []byte(content), 0644)
}

func (g *SPAGenerator) copyExampleFiles() error {
	if g.definition.Example == nil || !g.definition.Example.Enabled {
		return nil
	}

	examplesDir := filepath.Join(g.config.OutputDir, "src", "assets", "examples")

	for _, value := range g.definition.Example.Values {
		strVal, ok := value.(string)
		if !ok {
			continue
		}

		srcPath := g.resolveExampleFilePath(strVal)
		if srcPath == "" {
			continue
		}

		normalizedPath := strVal
		if strings.HasPrefix(strVal, "examples/") || strings.HasPrefix(strVal, "examples\\") {
			normalizedPath = strVal[9:]
		}

		dstPath := filepath.Join(examplesDir, normalizedPath)

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		srcContent, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (g *SPAGenerator) resolveExampleFilePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		if _, err := os.Stat(filePath); err == nil {
			return filePath
		}
		return ""
	}

	normalizedPath := filePath
	if strings.HasPrefix(filePath, "examples/") || strings.HasPrefix(filePath, "examples\\") {
		normalizedPath = filePath[9:]
	}

	pluginExamplesPath := filepath.Join(g.pluginDir, "examples", normalizedPath)
	if _, err := os.Stat(pluginExamplesPath); err == nil {
		return pluginExamplesPath
	}

	if g.config.ExamplesDir != "" {
		for _, dir := range strings.Split(g.config.ExamplesDir, ":") {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}
			configExamplesPath := filepath.Join(dir, normalizedPath)
			if _, err := os.Stat(configExamplesPath); err == nil {
				return configExamplesPath
			}
		}
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		execExamplesPath := filepath.Join(execDir, "examples", normalizedPath)
		if _, err := os.Stat(execExamplesPath); err == nil {
			return execExamplesPath
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		cwdExamplesPath := filepath.Join(cwd, "examples", normalizedPath)
		if _, err := os.Stat(cwdExamplesPath); err == nil {
			return cwdExamplesPath
		}
	}

	return ""
}

func (g *SPAGenerator) generateKarmaConfig() error {
	content := `process.env.CHROME_BIN = process.env.CHROME_BIN || require('puppeteer').executablePath();

module.exports = function (config) {
  config.set({
    basePath: '',
    frameworks: ['jasmine', '@angular-devkit/build-angular'],
    plugins: [
      require('karma-jasmine'),
      require('karma-chrome-launcher'),
      require('karma-jasmine-html-reporter'),
      require('karma-coverage'),
      require('@angular/build/private')
    ],
    client: {
      jasmine: {
        timeoutInterval: 300000
      },
      clearContext: false
    },
    jasmineHtmlReporter: {
      suppressAll: true
    },
    coverageReporter: {
      dir: require('path').join(__dirname, './coverage'),
      subdir: '.',
      reporters: [
        { type: 'html' },
        { type: 'text-summary' }
      ]
    },
    reporters: ['progress', 'kjhtml'],
    browsers: ['ChromeHeadless'],
    browserNoActivityTimeout: 300000,
    captureTimeout: 300000,
    restartOnFileChange: true
  });
};
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "karma.conf.js"), []byte(content), 0644)
}

func (g *SPAGenerator) generateAppSpec() error {
	hasExample := g.definition.Example != nil && g.definition.Example.Enabled && g.hasExampleFileInputs()

	if err := g.generateIntegrationSpec(hasExample); err != nil {
		return err
	}

	if g.isRPlugin() {
		return g.generateAppSpecR(hasExample)
	}
	return g.generateAppSpecPython(hasExample)
}

func (g *SPAGenerator) hasExampleFileInputs() bool {
	if g.definition.Example == nil || g.definition.Example.Values == nil {
		return false
	}

	fileInputs := make(map[string]bool)
	for _, input := range g.definition.Inputs {
		if input.Type == "file" {
			fileInputs[input.Name] = true
		}
	}

	for key, value := range g.definition.Example.Values {
		if fileInputs[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				return true
			}
		}
	}
	return false
}

func hasExampleFileInputs(definition *models.PluginDefinition) bool {
	if definition.Example == nil || definition.Example.Values == nil {
		return false
	}

	fileInputs := make(map[string]bool)
	for _, input := range definition.Inputs {
		if input.Type == "file" {
			fileInputs[input.Name] = true
		}
	}

	for key, value := range definition.Example.Values {
		if fileInputs[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				return true
			}
		}
	}
	return false
}

func (g *SPAGenerator) generateAppSpecPython(hasExample bool) error {
	exampleTest := ""
	if hasExample {
		exampleTest = `
  describe('Example Loading', () => {
    it('should have example enabled', () => {
      expect(component.hasExample()).toBeTrue();
    });

    it('should load example data into form', async () => {
      spyOn(window, 'fetch').and.returnValue(Promise.resolve({
        ok: true,
        text: () => Promise.resolve('col1\tcol2\nval1\tval2')
      } as Response));

      await component.loadExample();

      expect(component.fileData.size).toBeGreaterThan(0);
    });
  });
`
	} else {
		exampleTest = `
  describe('Example Loading', () => {
    it('should not have example enabled', () => {
      expect(component.hasExample()).toBeFalse();
    });
  });
`
	}

	content := `import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { AppComponent } from './app';
import { PyodideService } from './services/pyodide.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION } from './embedded/plugin-config';
import { Subject } from 'rxjs';

describe('AppComponent', () => {
  let component: AppComponent;
  let fixture: ComponentFixture<AppComponent>;
  let mockPyodideService: jasmine.SpyObj<PyodideService>;

  beforeEach(async () => {
    mockPyodideService = jasmine.createSpyObj('PyodideService', ['initialize', 'execute'], {
      progress$: new Subject(),
      output$: new Subject()
    });
    mockPyodideService.initialize.and.returnValue(Promise.resolve());

    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideAnimationsAsync(),
        { provide: PyodideService, useValue: mockPyodideService },
        BrowserFileHandler
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AppComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create the app', () => {
    expect(component).toBeTruthy();
  });

  it('should have plugin definition', () => {
    expect(component.plugin).toBeDefined();
    expect(component.plugin.plugin.id).toBe('` + g.definition.Plugin.ID + `');
  });

  describe('Form Building', () => {
    it('should build form with all inputs', () => {
      expect(component.form).toBeDefined();

      for (const input of PLUGIN_DEFINITION.inputs) {
        expect(component.form.contains(input.name)).toBeTrue();
      }
    });

    it('should set default values for inputs that have defaults', () => {
      const inputsWithDefaults = PLUGIN_DEFINITION.inputs.filter((i: any) => i.default !== undefined);

      for (const input of inputsWithDefaults) {
        const control = component.form.get(input.name);
        expect(control?.value).toEqual((input as any).default);
      }
    });

    it('should have required validators on required fields', () => {
      const requiredInputs = PLUGIN_DEFINITION.inputs.filter((i: any) => i.required && i.type !== 'file');

      for (const input of requiredInputs) {
        const control = component.form.get(input.name);
        if (control) {
          control.setValue('');
          control.updateValueAndValidity();
          expect(control.hasError('required')).toBeTrue();
        }
      }
    });
  });

  describe('Pyodide Initialization', () => {
    it('should call initialize on pyodide service', async () => {
      await fixture.whenStable();
      expect(mockPyodideService.initialize).toHaveBeenCalled();
    });

    it('should set pyodideReady after initialization', async () => {
      await fixture.whenStable();
      expect(component.pyodideReady()).toBeTrue();
    });
  });

  describe('Option Normalization', () => {
    it('should normalize string options', () => {
      const options = ['opt1', 'opt2'];
      const normalized = component.normalizeOptions(options);

      expect(normalized).toEqual([
        { value: 'opt1', label: 'opt1' },
        { value: 'opt2', label: 'opt2' }
      ]);
    });

    it('should pass through object options', () => {
      const options = [
        { value: 'v1', label: 'Label 1' },
        { value: 'v2', label: 'Label 2' }
      ];
      const normalized = component.normalizeOptions(options);

      expect(normalized).toEqual(options);
    });
  });
` + exampleTest + `
});
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "app.spec.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateAppSpecR(hasExample bool) error {
	exampleTest := ""
	if hasExample {
		exampleTest = `
  describe('Example Loading', () => {
    it('should have example enabled', () => {
      expect(component.hasExample()).toBeTrue();
    });

    it('should load example data into form', async () => {
      spyOn(window, 'fetch').and.returnValue(Promise.resolve({
        ok: true,
        text: () => Promise.resolve('col1\tcol2\nval1\tval2')
      } as Response));

      await component.loadExample();

      expect(component.fileData.size).toBeGreaterThan(0);
    });
  });
`
	} else {
		exampleTest = `
  describe('Example Loading', () => {
    it('should not have example enabled', () => {
      expect(component.hasExample()).toBeFalse();
    });
  });
`
	}

	content := `import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { AppComponent } from './app';
import { WebRService } from './services/webr.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION } from './embedded/plugin-config';
import { Subject } from 'rxjs';

describe('AppComponent', () => {
  let component: AppComponent;
  let fixture: ComponentFixture<AppComponent>;
  let mockWebRService: jasmine.SpyObj<WebRService>;

  beforeEach(async () => {
    mockWebRService = jasmine.createSpyObj('WebRService', ['initialize', 'execute'], {
      progress$: new Subject(),
      output$: new Subject()
    });
    mockWebRService.initialize.and.returnValue(Promise.resolve());

    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideAnimationsAsync(),
        { provide: WebRService, useValue: mockWebRService },
        BrowserFileHandler
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(AppComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create the app', () => {
    expect(component).toBeTruthy();
  });

  it('should have plugin definition', () => {
    expect(component.plugin).toBeDefined();
    expect(component.plugin.plugin.id).toBe('` + g.definition.Plugin.ID + `');
  });

  describe('Form Building', () => {
    it('should build form with all inputs', () => {
      expect(component.form).toBeDefined();

      for (const input of PLUGIN_DEFINITION.inputs) {
        expect(component.form.contains(input.name)).toBeTrue();
      }
    });

    it('should set default values for inputs that have defaults', () => {
      const inputsWithDefaults = PLUGIN_DEFINITION.inputs.filter((i: any) => i.default !== undefined);

      for (const input of inputsWithDefaults) {
        const control = component.form.get(input.name);
        expect(control?.value).toEqual((input as any).default);
      }
    });

    it('should have required validators on required fields', () => {
      const requiredInputs = PLUGIN_DEFINITION.inputs.filter((i: any) => i.required && i.type !== 'file');

      for (const input of requiredInputs) {
        const control = component.form.get(input.name);
        if (control) {
          control.setValue('');
          control.updateValueAndValidity();
          expect(control.hasError('required')).toBeTrue();
        }
      }
    });
  });

  describe('WebR Initialization', () => {
    it('should call initialize on webr service', async () => {
      await fixture.whenStable();
      expect(mockWebRService.initialize).toHaveBeenCalled();
    });

    it('should set webrReady after initialization', async () => {
      await fixture.whenStable();
      expect(component.webrReady()).toBeTrue();
    });
  });

  describe('Option Normalization', () => {
    it('should normalize string options', () => {
      const options = ['opt1', 'opt2'];
      const normalized = component.normalizeOptions(options);

      expect(normalized).toEqual([
        { value: 'opt1', label: 'opt1' },
        { value: 'opt2', label: 'opt2' }
      ]);
    });

    it('should pass through object options', () => {
      const options = [
        { value: 'v1', label: 'Label 1' },
        { value: 'v2', label: 'Label 2' }
      ];
      const normalized = component.normalizeOptions(options);

      expect(normalized).toEqual(options);
    });
  });
` + exampleTest + `
});
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "app.spec.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateIntegrationSpec(hasExample bool) error {
	if !hasExample {
		return nil
	}

	if g.isRPlugin() {
		return g.generateIntegrationSpecR()
	}
	return g.generateIntegrationSpecPython()
}

func (g *SPAGenerator) generateIntegrationSpecPython() error {
	content := `import { TestBed } from '@angular/core/testing';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { FormBuilder } from '@angular/forms';
import { PyodideService } from './services/pyodide.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION, PLUGIN_PACKAGES } from './embedded/plugin-config';
import { PLUGIN_SCRIPT } from './embedded/plugin-script';
import { PLUGIN_MODULES } from './embedded/plugin-modules';

describe('Integration: Example Execution', () => {
  let pyodideService: PyodideService;
  let fileHandler: BrowserFileHandler;
  let fb: FormBuilder;

  beforeAll(async () => {
    await TestBed.configureTestingModule({
      providers: [
        provideAnimationsAsync(),
        PyodideService,
        BrowserFileHandler,
        FormBuilder
      ]
    }).compileComponents();

    pyodideService = TestBed.inject(PyodideService);
    fileHandler = TestBed.inject(BrowserFileHandler);
    fb = TestBed.inject(FormBuilder);

    console.log('Initializing Pyodide for integration test...');
    await pyodideService.initialize(PLUGIN_PACKAGES);
    console.log('Pyodide initialized successfully');
  }, 300000);

  it('should run example and produce output', async () => {
    const form = fb.group({});
    for (const input of PLUGIN_DEFINITION.inputs) {
      const defaultValue = (input as any).default !== undefined ? (input as any).default : '';
      form.addControl(input.name, fb.control(defaultValue));
    }

    const fileInputs = new Set<string>();
    for (const input of PLUGIN_DEFINITION.inputs) {
      if (input.type === 'file') {
        fileInputs.add(input.name);
      }
    }

    const fileData = new Map<string, {name: string, content: string}>();
    const exampleValues = PLUGIN_DEFINITION.example?.values || {};

    for (const [key, value] of Object.entries(exampleValues)) {
      if ((key as string).endsWith('_source')) continue;

      if (typeof value === 'string' && fileInputs.has(key)) {
        const fileName = value.split('/').pop() || 'example.txt';
        const assetPath = value.startsWith('examples/') ? 'assets/examples/' + value.substring(9) : 'assets/examples/' + value;
        const response = await fetch(assetPath);
        if (response.ok) {
          const content = await response.text();
          fileData.set(key, { name: fileName, content });
          form.patchValue({ [key]: fileName });
        }
      } else {
        form.patchValue({ [key]: value });
      }
    }

    expect(fileData.size).toBeGreaterThan(0);

    const params: Record<string, any> = { ...form.value };
    for (const [key, fileInfo] of fileData.entries()) {
      params[key] = fileInfo;
    }

    console.log('Running plugin with example data...');
    const argsMapping = PLUGIN_DEFINITION.execution?.argsMapping || {};
    const outputDirFlag = PLUGIN_DEFINITION.execution?.outputDir;
    const inputs = PLUGIN_DEFINITION.inputs || [];
    const result = await pyodideService.execute(PLUGIN_SCRIPT, params, PLUGIN_MODULES, argsMapping, outputDirFlag, inputs);
    console.log('Plugin execution completed');

    expect(result).toBeDefined();
    expect(result.outputs).toBeDefined();
    expect(result.outputs.length).toBeGreaterThan(0);

    for (const output of result.outputs) {
      expect(output.name).toBeTruthy();
      expect(output.content).toBeTruthy();
      console.log('Output generated: ' + output.name + ' (' + output.content.length + ' bytes)');
    }
  }, 300000);
});
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "app.integration.spec.ts"), []byte(content), 0644)
}

func (g *SPAGenerator) generateIntegrationSpecR() error {
	content := `import { TestBed } from '@angular/core/testing';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { FormBuilder } from '@angular/forms';
import { WebRService } from './services/webr.service';
import { BrowserFileHandler } from './services/browser-file-handler';
import { PLUGIN_DEFINITION, PLUGIN_PACKAGES, PLUGIN_PACKAGE_REPOS } from './embedded/plugin-config';
import { PLUGIN_SCRIPT } from './embedded/plugin-script';
import { PLUGIN_MODULES } from './embedded/plugin-modules';

describe('Integration: Example Execution', () => {
  let webrService: WebRService;
  let fileHandler: BrowserFileHandler;
  let fb: FormBuilder;

  beforeAll(async () => {
    await TestBed.configureTestingModule({
      providers: [
        provideAnimationsAsync(),
        WebRService,
        BrowserFileHandler,
        FormBuilder
      ]
    }).compileComponents();

    webrService = TestBed.inject(WebRService);
    fileHandler = TestBed.inject(BrowserFileHandler);
    fb = TestBed.inject(FormBuilder);

    console.log('Initializing WebR for integration test...');
    await webrService.initialize(PLUGIN_PACKAGES, PLUGIN_PACKAGE_REPOS);
    console.log('WebR initialized successfully');
  }, 300000);

  it('should run example and produce output', async () => {
    const form = fb.group({});
    for (const input of PLUGIN_DEFINITION.inputs) {
      const defaultValue = (input as any).default !== undefined ? (input as any).default : '';
      form.addControl(input.name, fb.control(defaultValue));
    }

    const fileInputs = new Set<string>();
    for (const input of PLUGIN_DEFINITION.inputs) {
      if (input.type === 'file') {
        fileInputs.add(input.name);
      }
    }

    const fileData = new Map<string, {name: string, content: string}>();
    const exampleValues = PLUGIN_DEFINITION.example?.values || {};

    for (const [key, value] of Object.entries(exampleValues)) {
      if ((key as string).endsWith('_source')) continue;

      if (typeof value === 'string' && fileInputs.has(key)) {
        const fileName = value.split('/').pop() || 'example.txt';
        const assetPath = value.startsWith('examples/') ? 'assets/examples/' + value.substring(9) : 'assets/examples/' + value;
        const response = await fetch(assetPath);
        if (response.ok) {
          const content = await response.text();
          fileData.set(key, { name: fileName, content });
          form.patchValue({ [key]: fileName });
        }
      } else {
        form.patchValue({ [key]: value });
      }
    }

    expect(fileData.size).toBeGreaterThan(0);

    const params: Record<string, any> = { ...form.value };
    for (const [key, fileInfo] of fileData.entries()) {
      params[key] = fileInfo;
    }

    console.log('Running plugin with example data...');
    const argsMapping = PLUGIN_DEFINITION.execution?.argsMapping || {};
    const outputDirFlag = PLUGIN_DEFINITION.execution?.outputDir;
    const inputs = PLUGIN_DEFINITION.inputs || [];
    const result = await webrService.execute(PLUGIN_SCRIPT, params, PLUGIN_MODULES, argsMapping, outputDirFlag, inputs);
    console.log('Plugin execution completed');

    expect(result).toBeDefined();
    expect(result.outputs).toBeDefined();
    expect(result.outputs.length).toBeGreaterThan(0);

    for (const output of result.outputs) {
      expect(output.name).toBeTruthy();
      expect(output.content).toBeTruthy();
      console.log('Output generated: ' + output.name + ' (' + output.content.length + ' bytes)');
    }
  }, 300000);
});
`
	return os.WriteFile(filepath.Join(g.config.OutputDir, "src", "app", "app.integration.spec.ts"), []byte(content), 0644)
}

func toJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	result, _ := json.Marshal(items)
	return string(result)
}
