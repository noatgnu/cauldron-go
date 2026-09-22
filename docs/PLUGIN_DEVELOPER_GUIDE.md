# Plugin Developer Guide

This guide documents the `plugin.yaml` schema and how Cauldron loads, validates and executes a plugin. 
It reflects the schema loaded at runtime (`backend/models/plugin_v2.go` and `plugin.go`), not any tool-specific variant.

## Part 1: Plugin Schema Reference

A plugin is a directory under `plugins/` containing a `plugin.yaml` (or `plugin.yml`) plus an entrypoint script. 
Config resolution checks, in order: `plugin.<GOOS>.yaml`, then `plugin.<GOOS>.yml`, then `plugin.yaml`, then `plugin.yml`. 
The first file that exists wins, so a plugin can ship an OS-specific override (`GOOS` is `windows`, `linux`, or `darwin`).

The full field-by-field schema reference is generated directly from `schemas/plugin-schema.json`. 
Regenerate it with `go run ./cmd/schema-doc-generator` after changing the schema. This guide covers the parts it does not: runtime behavior, validation, scaffolding, step markers, and examples.

Two plugin systems exist in this codebase. This guide, `schemas/plugin-schema.json`, and every bundled plugin all describe the current one: `PluginDefinition`/`PluginRuntimeV2` in `backend/models/plugin_v2.go`, loaded by `PluginLoaderV2`. 
A second, older model (`Plugin`/`PluginConfig` in `backend/models/plugin.go`, with a flat `runtime: <string>` and a nested `script: {path: ...}`) still exist in the code but will be deprecated. 
New plugins should be written against the current schema.

Some main fields from the configuration file:

- `entrypoint` is the required field name. `script` is accepted as a deprecated fallback if `entrypoint` is absent.
- `environments` accepts `python`, `r`, `direct`, `docker`. Only `python`, `r`, `direct`, and `docker` are implemented in the script executor. `julia` and `node` are future target for integration. The first entry in the list is the primary environment and decides which executor runs the plugin.
- `argsMapping` does not need an entry for every input. The loader auto-fills `argsMapping[input.name] = "--" + input.name` for any input without one. Only add an explicit mapping for a different flag name, a transform, conditional inclusion with `when`, or `passAsValue`. Full behavior is in [COMMAND_EXECUTION_REFERENCE.md](COMMAND_EXECUTION_REFERENCE.md).
- `plots:` behavior, including which `component` values render and how the generic renderer works, is in [PLUGIN_PLOT_SYSTEM.md](PLUGIN_PLOT_SYSTEM.md) and [plugin-plots-guide.md](plugin-plots-guide.md).

## Part 2: Getting Started and Scaffolding

Run:

```
go run ./cmd/plugin-scaffolder
```

It prompts interactively for the plugin ID (lowercase and hyphenated, must not already exist under `plugins/`), name, description, version, author, category, and runtime (`python`, `r`, or `pythonWithR`).

It generates the following under `plugins/<id>/`:
- `plugin.yaml`
- `<script>.py` or `.R`, pre-populated with `@step` markers and `argparse`/`optparse` boilerplate for `--input_file`/`--output_folder`
- `.gitignore`
- `examples/README.md` and `examples/params.json`
- `.github/workflows/test-plugin.yml`
- For R-capable runtimes, `.github/scripts/generate-deps-graph.R` and `merge-deps-graph.py`

A freshly scaffolded plugin with no `repository:` field is classified `builtin` the first time the app loads it. See [PLUGIN_INSTALLATION.md](PLUGIN_INSTALLATION.md) for the builtin versus remote distinction. Rebuild via `build.sh`, or copy the folder next to a dev binary's `plugins/` directory and call `ReloadPluginsV2()`, to pick it up in a running instance.

## Part 3: Step Markers and Workflow Diagrams

`cmd/plugin-doc-generator` scans a plugin's entrypoint script for step markers and renders a Mermaid workflow diagram into the generated README. This is gated behind `diagram.enabled: true`, a doc-generator-only config section that has no effect on runtime execution.

### Grammar

```
^#+\s*@step(-if)?(?:\[([^\]]*)\])?\s*:\s*(.+)$
```

This works for any `#`-comment language (Python and R both qualify) and is matched per trimmed line.

- `# @step: Loading input data` produces a process node, rendered as a box.
- `# @step-if: Has peptide-level data?` produces a decision node, rendered as a diamond.
- An optional bracketed attribute block looks like `# @step[id=foo,from=bar,loop-to=baz]: Label`, with comma-separated `key=value` pairs:
  - `id=<anchor>` gives the step a stable ID that other steps can reference. Omit it and an auto ID (`step1`, `step2`, and so on) is assigned.
  - `from=<ref>[+<ref>...]` declares explicit incoming edges, replacing the default behavior of chaining from the previous step in file order. Each `<ref>` is `anchor` or `anchor:label`, where the label becomes the Mermaid edge label. Join multiple refs with `+` to express a merge point.
  - `loop-to=<ref>[+<ref>...]` adds an additional outgoing edge back to an earlier step, for loop-back arrows. It uses the same `anchor[:label]` syntax.

Example, branch and merge:

```r
# @step[id=load]: Loading abundance data
# @step-if[id=check,from=load]: Has peptide-level data?
# @step[id=msqrob2,from=check:yes]: Running msqrob2 differential expression
# @step[id=limma,from=check:no]: Running limma differential expression
# @step[from=msqrob2+limma]: Differential expression complete
```

Example, loop:

```r
# @step[id=fetch]: Fetch next batch
# @step-if[id=more,from=fetch]: More batches remaining?
# @step[id=process,from=more:yes,loop-to=fetch]: Process batch
# @step[from=more:no]: All batches processed
```

Steps with no incoming edge get a synthetic `Start --> stepN` edge. Steps with no outgoing edge get `stepN --> End`. Plain markers with no attributes are unaffected by any of this. They keep chaining sequentially exactly as before.

### Legacy fallback

This is only used when a script has zero `@step`/`@step-if` markers anywhere:

- R: `message(...[N/M]... "label")`
- Python: `print(...[N/M]... "label")` or `logger.info(...[N/M]... "label")`

These produce a flat sequential chain with no branching support. Marker-based steps always take priority when both exist in the same script.

`source("file.R")` in R, and local `import`/`from ... import` in Python (best effort), are followed recursively, so markers in sourced or imported files are picked up too.

Mermaid label safety is automatic. A literal `"` in a label is escaped to `#quot;` for you, so you do not need to escape it yourself.

## Part 4: Validation Rules

Run `./bin/plugin-validator <plugin.yaml-or-dir>` before committing a plugin. Hard errors:

- `plugin.id`, `.name`, `.description`, `.version`, and `.category` are required.
- `runtime.environments` is required, and each entry must be in the valid set.
- `runtime.entrypoint` (or the deprecated `script`) is required. For non-`direct` and non-`docker` runtimes, the file must exist relative to the plugin directory.
- Every input needs `name`, `label`, and `type`. `type` must be a recognized value.
- `select` needs `options` or `optionsFromFile`. `multiselect-grouped` needs `groups` or `groupsFromFile`. `column-selector` needs `sourceFile`. `color-map` needs `keysFrom`.
- `execution.outputDir` is required.
- If `requirements.pythonRequirementsFile` or `rPackagesFile` is given, the file must exist.

Warnings, which are non-fatal: no inputs defined; `visibleWhen.field` referencing a non-existent input; `diagram.enabled: true` with no step markers anywhere in the entrypoint script.

## Part 5: Worked Examples

`plugins/cv-plot/plugin.yaml` is a simple example: mostly optional inputs, two image outputs, and an `example:` block.

```yaml
plugin:
  id: "cv-plot"
  name: "Coefficient of Variation Plot"
  description: "Generate coefficient of variation (CV) plots for DIA-NN data quality assessment"
  version: "1.0.0"
  author: "CauldronGO Team"
  category: "visualization"
  icon: "analytics"

runtime:
  environments: ["python"]
  entrypoint: "cv.py"

inputs:
  - name: "annotation_file"
    label: "Annotation File"
    type: "file"
    required: true
    accept: ".csv,.tsv,.txt"
    description: "Sample annotation file with conditions"
  - name: "log_file_path"
    label: "Log File"
    type: "file"
    required: false
    accept: ".txt,.log"
    description: "DIA-NN log file to extract sample names (optional if sample_names provided)"
  - name: "report_pr_file_path"
    label: "Protein Report File"
    type: "file"
    required: false
    accept: ".csv,.tsv,.txt"
  - name: "report_pg_file_path"
    label: "Protein Group Report File"
    type: "file"
    required: false
    accept: ".csv,.tsv,.txt"
  - name: "intensity_col"
    label: "Intensity Column"
    type: "text"
    default: "Intensity"
  - name: "sample_names"
    label: "Sample Names"
    type: "text"
    required: false
    placeholder: "Sample1,Sample2,Sample3"

outputs:
  - name: "pr_cv_plot"
    path: "pr_cv.svg"
    type: "image"
    format: "svg"
  - name: "pg_cv_plot"
    path: "pg_cv.svg"
    type: "image"
    format: "svg"

execution:
  argsMapping:
    log_file_path: "--log_file_path"
    report_pr_file_path: "--report_pr_file_path"
    report_pg_file_path: "--report_pg_file_path"
    intensity_col: "--intensity_col"
    annotation_file: "--annotation_file"
    sample_names: "--sample_names"
  outputDir: "--output_folder"
  requirements:
    python: ">=3.11"
    packages:
      - "pandas>=2.0.0"
      - "numpy>=1.24.0"
      - "scipy>=1.10.0"
      - "seaborn>=0.12.0"
      - "matplotlib>=3.7.0"

example:
  enabled: true
  values:
    annotation_file: "diann/annotation.txt"
    log_file_path: "diann/Reports.log.txt"
    report_pr_file_path: "diann/Reports.pr_matrix.tsv"
    report_pg_file_path: "diann/Reports.pg_matrix.tsv"
```

`plugins/pca-analysis/plugin.yaml` is another example but with visualization and user input options, showing a `column-selector`, a `number` with min, max, and step, a `boolean` flag gated with `when`, a `comma-join` transform, `annotation:`, and a `plots:` section.

```yaml
plugin:
  id: "pca-analysis"
  name: "PCA Analysis"
  description: "Principal Component Analysis for dimensionality reduction and visualization"
  version: "1.0.0"
  author: "CauldronGO Team"
  category: "analysis"
  icon: "scatter_plot"

runtime:
  environments: ["python"]
  entrypoint: "pca.py"

inputs:
  - name: "input_file"
    label: "Input File"
    type: "file"
    required: true
    accept: ".csv,.tsv,.txt"
  - name: "annotation_file"
    label: "Sample Annotation File"
    type: "file"
    required: false
    accept: ".csv,.tsv,.txt"
    description: "Optional annotation file for sample grouping and coloring (Sample, Condition, Batch, Color)"
  - name: "columns_name"
    label: "Sample Columns"
    type: "column-selector"
    required: true
    multiple: true
    sourceFile: "input_file"
  - name: "n_components"
    label: "Number of Components"
    type: "number"
    required: true
    default: 2
    min: 2
    max: 10
    step: 1
  - name: "log2"
    label: "Apply Log2 Transform"
    type: "boolean"
    default: false

outputs:
  - name: "pca_output"
    path: "pca_output.txt"
    type: "data"
    format: "tsv"
  - name: "explained_variance"
    path: "explained_variance_ratio.json"
    type: "data"
    format: "json"

annotation:
  samplesFrom: "columns_name"
  annotationFile: "annotation_file"

plots:
  - id: "pca-scatter"
    name: "PCA Scatter Plot"
    type: "scatter"
    component: "PcaPlot"
    dataSource: "pca_output"
    config:
      axes:
        x: "x_pca"
        y: "y_pca"
        colorBy: "condition"
        labels: "sample"
    customization:
      - { name: "title", label: "Plot Title", type: "text", default: "" }
      - { name: "showGrid", label: "Show Grid", type: "boolean", default: true }
      - { name: "markerSize", label: "Marker Size", type: "number", default: 10, min: 1, max: 50 }

execution:
  argsMapping:
    input_file: "--input_file"
    columns_name:
      flag: "--columns_name"
      transform: "comma-join"
    n_components: "--n_components"
    log2:
      flag: "--log2"
      when: "true"
  outputDir: "--output_folder"
  requirements:
    python: ">=3.11"
    packages: ["numpy>=1.24.0", "pandas>=2.0.0", "scikit-learn>=1.3.0"]

example:
  enabled: true
  values:
    input_file: "diann/imputed.data.txt"
    columns_name_source: "diann/imputed.data.txt"
    columns_name: ["sample1.raw", "sample2.raw", "sample3.raw"]
    n_components: 2
    log2: true
```

More examples can be found in the builtin bundled of plugins like `plugins/uniprot-fetcher/plugin.yaml` (a `select` with `optionsFromFile`, a `multiselect-grouped` with `groupsFromFile`, `visibleWhen`, and a `direct` runtime) and `plugins/wide-to-long/plugin.yaml` (a `direct` runtime with a compiled binary entrypoint).