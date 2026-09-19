package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func TestParseRScript_StepMarkers(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "run.R", `
# @step: Loading pg_matrix...
x <- 1
# @step-if: Loading stats file for QC plots...
y <- 2
# @step: Writing normalised output matrices...
`)

	markerSteps, legacySteps, err := parseRScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parseRScript error: %v", err)
	}

	if len(legacySteps) != 0 {
		t.Errorf("expected no legacy steps found, got %d: %+v", len(legacySteps), legacySteps)
	}

	if len(markerSteps) != 3 {
		t.Fatalf("expected 3 marker steps, got %d: %+v", len(markerSteps), markerSteps)
	}

	if markerSteps[0].Label != "Loading pg_matrix..." || markerSteps[0].Type != "process" {
		t.Errorf("step 0 = %+v, want label 'Loading pg_matrix...' type 'process'", markerSteps[0])
	}
	if markerSteps[1].Label != "Loading stats file for QC plots..." || markerSteps[1].Type != "decision" {
		t.Errorf("step 1 = %+v, want label 'Loading stats file for QC plots...' type 'decision'", markerSteps[1])
	}
	if markerSteps[2].Label != "Writing normalised output matrices..." || markerSteps[2].Type != "process" {
		t.Errorf("step 2 = %+v, want label 'Writing normalised output matrices...' type 'process'", markerSteps[2])
	}
}

func TestParseRScript_LegacyFallbackWhenNoMarkers(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "run.R", `
message("[1/2] Reading data...")
x <- 1
message("[2/2] Writing output...")
`)

	markerSteps, legacySteps, err := parseRScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parseRScript error: %v", err)
	}

	if len(markerSteps) != 0 {
		t.Errorf("expected no marker steps, got %d: %+v", len(markerSteps), markerSteps)
	}
	if len(legacySteps) != 2 {
		t.Fatalf("expected 2 legacy steps, got %d: %+v", len(legacySteps), legacySteps)
	}
	if legacySteps[0].Label != "Reading data..." {
		t.Errorf("legacy step 0 label = %q, want 'Reading data...'", legacySteps[0].Label)
	}
	if legacySteps[1].Label != "Writing output..." {
		t.Errorf("legacy step 1 label = %q, want 'Writing output...'", legacySteps[1].Label)
	}
}

func TestParseRScript_MarkerLabelWithPunctuationIntact(t *testing.T) {
	// Regression test: the legacy convention truncates labels at the first
	// quote/paren it finds. The @step marker convention must not have this bug.
	dir := t.TempDir()
	path := writeScript(t, dir, "run.R", `
# @step: Filtering proteins (contaminants and minimum peptides)
x <- 1
`)

	markerSteps, _, err := parseRScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parseRScript error: %v", err)
	}
	if len(markerSteps) != 1 {
		t.Fatalf("expected 1 marker step, got %d: %+v", len(markerSteps), markerSteps)
	}

	want := "Filtering proteins (contaminants and minimum peptides)"
	if markerSteps[0].Label != want {
		t.Errorf("label = %q, want %q", markerSteps[0].Label, want)
	}
}

func TestParseRScript_FollowsSourcedFiles(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "helper.R", `
# @step: Helper step one
`)
	path := writeScript(t, dir, "run.R", `
source("helper.R")
# @step: Main step two
`)

	markerSteps, _, err := parseRScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parseRScript error: %v", err)
	}
	if len(markerSteps) != 2 {
		t.Fatalf("expected 2 marker steps across sourced files, got %d: %+v", len(markerSteps), markerSteps)
	}
	if markerSteps[0].Label != "Helper step one" || markerSteps[1].Label != "Main step two" {
		t.Errorf("unexpected step order/labels: %+v", markerSteps)
	}
}

func TestParsePythonScript_StepMarkers(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "run.py", `
# @step: Loading input file...
x = 1
# @step-if: Optional QC plot...
`)

	markerSteps, legacySteps, err := parsePythonScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parsePythonScript error: %v", err)
	}
	if len(legacySteps) != 0 {
		t.Errorf("expected no legacy steps found, got %d: %+v", len(legacySteps), legacySteps)
	}
	if len(markerSteps) != 2 {
		t.Fatalf("expected 2 marker steps, got %d: %+v", len(markerSteps), markerSteps)
	}
	if markerSteps[0].Type != "process" || markerSteps[1].Type != "decision" {
		t.Errorf("unexpected step types: %+v", markerSteps)
	}
}

func TestParsePythonScript_LegacyFallbackWhenNoMarkers(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "run.py", `
print("[1/1] Processing...")
`)

	markerSteps, legacySteps, err := parsePythonScript(path, dir, make(map[string]bool))
	if err != nil {
		t.Fatalf("parsePythonScript error: %v", err)
	}
	if len(markerSteps) != 0 {
		t.Errorf("expected no marker steps, got %d: %+v", len(markerSteps), markerSteps)
	}
	if len(legacySteps) != 1 || legacySteps[0].Label != "Processing..." {
		t.Errorf("expected 1 legacy step 'Processing...', got %+v", legacySteps)
	}
}

func TestGenerateMermaidDiagram_DecisionShape(t *testing.T) {
	steps := []WorkflowStep{
		{ID: "step1", Label: "Load data", Type: "process"},
		{ID: "step2", Label: "Optional QC", Type: "decision"},
	}

	diagram := generateMermaidDiagram(steps)

	if !strings.Contains(diagram, `step1["Load data"]`) {
		t.Errorf(`expected process node 'step1["Load data"]' in diagram:`+"\n%s", diagram)
	}
	if !strings.Contains(diagram, `step2{"Optional QC"}`) {
		t.Errorf(`expected decision node 'step2{"Optional QC"}' in diagram:`+"\n%s", diagram)
	}
}

func TestGenerateMermaidDiagram_LabelWithParensIsQuoted(t *testing.T) {
	// Regression test: Mermaid's flowchart grammar treats unquoted parentheses
	// inside [..] as syntax, not text, and fails to parse the node otherwise.
	// This reproduces the exact label that broke rendering in a real plugin.
	steps := []WorkflowStep{
		{ID: "step1", Label: "Filtering proteins (contaminants and minimum peptides)", Type: "process"},
	}

	diagram := generateMermaidDiagram(steps)

	want := `step1["Filtering proteins (contaminants and minimum peptides)"]`
	if !strings.Contains(diagram, want) {
		t.Errorf("expected quoted node %q in diagram:\n%s", want, diagram)
	}
}

func TestGenerateMermaidDiagram_LabelWithDoubleQuoteIsEscaped(t *testing.T) {
	steps := []WorkflowStep{
		{ID: "step1", Label: `Loading "raw" data`, Type: "process"},
	}

	diagram := generateMermaidDiagram(steps)

	want := `step1["Loading #quot;raw#quot; data"]`
	if !strings.Contains(diagram, want) {
		t.Errorf("expected escaped node %q in diagram:\n%s", want, diagram)
	}
}

func TestGenerateDiagramSection_PrefersMarkersOverLegacyInSameFile(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "run.R", `
message("[1/1] Legacy-only label")
# @step: Marker label
`)

	plugin := PluginConfig{
		Runtime: PluginRuntime{Environments: []string{"r"}, Entrypoint: "run.R"},
		Diagram: &DiagramConfig{Enabled: true},
	}

	got := generateDiagramSection(plugin, dir)
	if !strings.Contains(got, "Marker label") {
		t.Errorf("expected diagram to use the marker-based label, got:\n%s", got)
	}
	if strings.Contains(got, "Legacy-only label") {
		t.Errorf("expected diagram to ignore the legacy label when markers are present, got:\n%s", got)
	}
}

func TestGenerateDiagramSection(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "run.R", `
# @step: Loading data...
# @step: Processing...
`)

	base := PluginConfig{
		Runtime: PluginRuntime{Environments: []string{"r"}, Entrypoint: "run.R"},
	}

	if got := generateDiagramSection(base, dir); got != "" {
		t.Errorf("expected empty section when diagram is nil, got:\n%s", got)
	}

	disabled := base
	disabled.Diagram = &DiagramConfig{Enabled: false}
	if got := generateDiagramSection(disabled, dir); got != "" {
		t.Errorf("expected empty section when diagram.enabled is false, got:\n%s", got)
	}

	enabled := base
	enabled.Diagram = &DiagramConfig{Enabled: true}
	got := generateDiagramSection(enabled, dir)
	if !strings.Contains(got, "## Workflow Diagram") {
		t.Errorf("expected a Workflow Diagram section, got:\n%s", got)
	}
	if !strings.Contains(got, "```mermaid") {
		t.Errorf("expected a mermaid code block, got:\n%s", got)
	}

	missingEntrypoint := enabled
	missingEntrypoint.Runtime.Entrypoint = "does-not-exist.R"
	if got := generateDiagramSection(missingEntrypoint, dir); got != "" {
		t.Errorf("expected empty section when entrypoint is missing, got:\n%s", got)
	}
}

func TestGenerateMermaidDiagram_SimpleChainIsByteIdenticalToBaseline(t *testing.T) {
	steps := []WorkflowStep{
		{ID: "step1", Label: "Load data", Type: "process"},
		{ID: "step2", Label: "Optional QC", Type: "decision"},
	}

	got := generateMermaidDiagram(steps)
	want := "```mermaid\n" +
		"flowchart TD\n" +
		"    Start([Start]) --> step1\n" +
		`    step1["Load data"]` + "\n" +
		"    step1 --> step2\n" +
		`    step2{"Optional QC"}` + "\n" +
		"    step2 --> End([End])\n" +
		"```"

	if got != want {
		t.Errorf("simple chain diagram changed shape:\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestParseStepMarker_IdFromLoopToAttributes(t *testing.T) {
	stepID := 0

	step, ok := parseStepMarker(`# @step[id=load]: Loading data`, &stepID)
	if !ok || step.ID != "load" || !step.HasExplicitID {
		t.Fatalf("expected explicit id 'load', got %+v ok=%v", step, ok)
	}

	step, ok = parseStepMarker(`# @step-if[id=check]: Has peptides?`, &stepID)
	if !ok || step.ID != "check" || step.Type != "decision" {
		t.Fatalf("expected decision step id 'check', got %+v ok=%v", step, ok)
	}

	step, ok = parseStepMarker(`# @step[id=msqrob2,from=check:yes]: Run msqrob2`, &stepID)
	if !ok || len(step.From) != 1 || step.From[0].Anchor != "check" || step.From[0].Label != "yes" {
		t.Fatalf("expected from=check:yes, got %+v ok=%v", step, ok)
	}

	step, ok = parseStepMarker(`# @step[from=msqrob2+limma]: Complete`, &stepID)
	if !ok || len(step.From) != 2 || step.From[0].Anchor != "msqrob2" || step.From[1].Anchor != "limma" {
		t.Fatalf("expected merge from msqrob2+limma, got %+v ok=%v", step, ok)
	}
	if step.HasExplicitID {
		t.Errorf("expected auto-assigned id when id= is omitted, got explicit %q", step.ID)
	}

	step, ok = parseStepMarker(`# @step[id=process,from=more:yes,loop-to=fetch]: Process batch`, &stepID)
	if !ok || len(step.LoopTo) != 1 || step.LoopTo[0].Anchor != "fetch" {
		t.Fatalf("expected loop-to=fetch, got %+v ok=%v", step, ok)
	}
}

func TestGenerateMermaidDiagram_BranchAndMerge(t *testing.T) {
	steps := []WorkflowStep{
		{ID: "load", Label: "Loading data", Type: "process", HasExplicitID: true},
		{ID: "check", Label: "Has peptides?", Type: "decision", HasExplicitID: true},
		{ID: "msqrob2", Label: "Run msqrob2", Type: "process", HasExplicitID: true,
			From: []WorkflowEdgeRef{{Anchor: "check", Label: "yes"}}},
		{ID: "limma", Label: "Run limma", Type: "process", HasExplicitID: true,
			From: []WorkflowEdgeRef{{Anchor: "check", Label: "no"}}},
		{Label: "Complete", Type: "process",
			From: []WorkflowEdgeRef{{Anchor: "msqrob2"}, {Anchor: "limma"}}},
	}

	diagram := generateMermaidDiagram(steps)

	for _, want := range []string{
		`Start([Start]) --> load`,
		`check -->|"yes"| msqrob2`,
		`check -->|"no"| limma`,
		`msqrob2 --> step1`,
		`limma --> step1`,
		`step1["Complete"]`,
		`step1 --> End([End])`,
	} {
		if !strings.Contains(diagram, want) {
			t.Errorf("expected diagram to contain %q, got:\n%s", want, diagram)
		}
	}
}

func TestGenerateMermaidDiagram_LoopBack(t *testing.T) {
	steps := []WorkflowStep{
		{ID: "fetch", Label: "Fetch next batch", Type: "process", HasExplicitID: true},
		{ID: "more", Label: "More batches remaining?", Type: "decision", HasExplicitID: true},
		{ID: "process", Label: "Process batch", Type: "process", HasExplicitID: true,
			From:   []WorkflowEdgeRef{{Anchor: "more", Label: "yes"}},
			LoopTo: []WorkflowEdgeRef{{Anchor: "fetch"}}},
		{Label: "All batches processed", Type: "process",
			From: []WorkflowEdgeRef{{Anchor: "more", Label: "no"}}},
	}

	diagram := generateMermaidDiagram(steps)

	if !strings.Contains(diagram, "process --> fetch") {
		t.Errorf("expected a loop-back edge from process to fetch, got:\n%s", diagram)
	}
	if !strings.Contains(diagram, `more -->|"yes"| process`) {
		t.Errorf("expected the gated forward edge into the loop body, got:\n%s", diagram)
	}
}

func TestGenerateMermaidDiagram_UnresolvedAnchorDegradesGracefully(t *testing.T) {
	steps := []WorkflowStep{
		{Label: "Orphaned branch step", Type: "process",
			From: []WorkflowEdgeRef{{Anchor: "does-not-exist", Label: "yes"}}},
	}

	diagram := generateMermaidDiagram(steps)

	if !strings.Contains(diagram, `step1["Orphaned branch step"]`) {
		t.Errorf("expected the step itself to still render, got:\n%s", diagram)
	}
	if strings.Contains(diagram, "does-not-exist") {
		t.Errorf("expected the unresolved anchor to be silently dropped, got:\n%s", diagram)
	}
	if !strings.Contains(diagram, "Start([Start]) --> step1") {
		t.Errorf("expected the step with no resolved incoming edge to become a root, got:\n%s", diagram)
	}
}
