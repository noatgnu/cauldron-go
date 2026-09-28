package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
)

var diagramIDPattern = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func diagramSafeID(s string) string {
	safe := diagramIDPattern.ReplaceAllString(s, "_")
	if safe == "" {
		return "n"
	}
	return safe
}

func diagramSafeLabel(s string) string {
	return strings.ReplaceAll(s, `"`, "#quot;")
}

func quotedLabel(parts ...string) string {
	return `"` + diagramSafeLabel(strings.Join(parts, "\n")) + `"`
}

func stageNodeID(stageIndex int) string {
	return fmt.Sprintf("S%d", stageIndex)
}

func stagePortID(stageIndex int, direction, name string) string {
	return fmt.Sprintf("S%d_%s_%s", stageIndex, direction, diagramSafeID(name))
}

type recipeBinding struct {
	InputName    string
	SourceStage  int
	SourceOutput string
}

func parseStageBindings(raw models.JSONMap) []recipeBinding {
	bindings := make([]recipeBinding, 0, len(raw))
	for inputName, v := range raw {
		bindingMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		stageIdx, _ := bindingMap["stage"].(float64)
		outputName, _ := bindingMap["output"].(string)
		bindings = append(bindings, recipeBinding{
			InputName:    inputName,
			SourceStage:  int(stageIdx),
			SourceOutput: outputName,
		})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].InputName < bindings[j].InputName })
	return bindings
}

func statusClass(status CompatibilityStatus) string {
	switch status {
	case CompatibilityMissing:
		return "missing"
	case CompatibilityIncompatible:
		return "incompatible"
	case CompatibilityCompatibleVersionDiffers:
		return "versionDiffers"
	default:
		return "compatible"
	}
}

// diagramStage is the mode-agnostic shape both a persisted recipe's stages
// and a raw (not-yet-downloaded) recipe export's stages are normalized into
// before rendering, so the Mermaid-writing code below only has one input
// shape to deal with.
type diagramStage struct {
	Index         int
	PluginID      string
	PluginVersion string
	Params        map[string]interface{}
	Bindings      []recipeBinding
	Status        CompatibilityStatus
}

func diagramStagesFromRecipeStages(stages []models.RecipeStage, statusByStage map[int]CompatibilityStatus) []diagramStage {
	result := make([]diagramStage, len(stages))
	for i, s := range stages {
		result[i] = diagramStage{
			Index:         s.StageIndex,
			PluginID:      s.PluginID,
			PluginVersion: s.PluginVersion,
			Params:        s.Params,
			Bindings:      parseStageBindings(s.Bindings),
			Status:        statusByStage[s.StageIndex],
		}
	}
	return result
}

func (r *RecipeService) diagramStagesFromData(stages []RecipeStageData) []diagramStage {
	result := make([]diagramStage, len(stages))
	for i, s := range stages {
		bindings := make([]recipeBinding, 0, len(s.Bindings))
		for inputName, b := range s.Bindings {
			bindings = append(bindings, recipeBinding{InputName: inputName, SourceStage: b.Stage, SourceOutput: b.Output})
		}
		sort.Slice(bindings, func(a, c int) bool { return bindings[a].InputName < bindings[c].InputName })

		status := CompatibilityMissing
		if installed, err := r.pluginLoader.GetPluginByStringID(s.PluginID); err == nil {
			if installed.Definition.Plugin.Version == s.PluginVersion {
				status = CompatibilityCompatible
			} else {
				status = CompatibilityCompatibleVersionDiffers
			}
		}

		result[i] = diagramStage{
			Index:         i,
			PluginID:      s.PluginID,
			PluginVersion: s.PluginVersion,
			Params:        s.Params,
			Bindings:      bindings,
			Status:        status,
		}
	}
	return result
}

// GenerateRecipeDiagram renders a saved recipe's stage graph as Mermaid
// flowchart source. Stages whose index is in expandedStages are drawn as a
// subgraph showing that stage's own step diagram (or input/output ports as
// a fallback); every other stage is drawn as a single collapsed box. An
// empty expandedStages renders every stage collapsed.
func (r *RecipeService) GenerateRecipeDiagram(recipeID string, expandedStages []int) (string, error) {
	stages, err := r.GetRecipeStages(recipeID)
	if err != nil {
		return "", err
	}
	if len(stages) == 0 {
		return "", fmt.Errorf("recipe has no stages")
	}

	report, err := r.CheckCompatibility(recipeID)
	if err != nil {
		return "", err
	}
	statusByStage := make(map[int]CompatibilityStatus, len(report.Stages))
	for _, s := range report.Stages {
		statusByStage[s.StageIndex] = s.Status
	}

	return r.renderDiagram(diagramStagesFromRecipeStages(stages, statusByStage), expandedStages), nil
}

// GenerateRecipeDiagramFromData renders a diagram directly from a recipe
// export envelope (the same shape used by ImportRecipeFromData/the registry
// download payload), without requiring the recipe to already be saved
// locally. Each stage's plugin is resolved against whatever is currently
// installed; a stage whose plugin isn't installed yet is rendered as
// "missing" rather than failing the whole diagram, so this doubles as a
// preview of what a registry recipe still needs before it can run.
func (r *RecipeService) GenerateRecipeDiagramFromData(raw []byte, expandedStages []int) (string, error) {
	var data RecipeData
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", fmt.Errorf("failed to parse recipe data: %w", err)
	}
	if len(data.Stages) == 0 {
		return "", fmt.Errorf("recipe data has no stages")
	}

	return r.renderDiagram(r.diagramStagesFromData(data.Stages), expandedStages), nil
}

func (r *RecipeService) renderDiagram(stages []diagramStage, expandedStages []int) string {
	expanded := make(map[int]bool, len(expandedStages))
	for _, idx := range expandedStages {
		expanded[idx] = true
	}

	var b strings.Builder
	b.WriteString("flowchart TD\n")
	r.writeStages(&b, stages, expanded)

	for _, s := range stages {
		fmt.Fprintf(&b, "    class %s %s\n", stageNodeID(s.Index), statusClass(s.Status))
	}

	b.WriteString("    classDef compatible fill:#ecfdf5,stroke:#059669,color:#065f46\n")
	b.WriteString("    classDef versionDiffers fill:#fffbeb,stroke:#d97706,color:#92400e\n")
	b.WriteString("    classDef incompatible fill:#fff1f2,stroke:#e11d48,color:#9f1239\n")
	b.WriteString("    classDef missing fill:#fef2f2,stroke:#b91c1c,color:#7f1d1d,stroke-dasharray: 4 3\n")

	return b.String()
}

func stageTitle(s diagramStage) string {
	title := fmt.Sprintf("Stage %d: %s", s.Index+1, s.PluginID)
	if s.PluginVersion != "" {
		title += " v" + s.PluginVersion
	}
	return title
}

// writeStages renders every stage, expanding only those listed in expanded.
// A collapsed stage and an expanded stage that falls back to plain ports
// both ultimately reference the same node id (stageNodeID), so bindings
// between two collapsed stages render as a single labeled edge exactly like
// before; a binding touching a port-expanded stage renders as an unlabeled
// dotted edge straight to that specific port instead.
func (r *RecipeService) writeStages(b *strings.Builder, stages []diagramStage, expanded map[int]bool) {
	usesPorts := make(map[int]bool, len(stages))

	for _, s := range stages {
		nodeID := stageNodeID(s.Index)

		if !expanded[s.Index] {
			fmt.Fprintf(b, "    %s[%s]\n", nodeID, quotedLabel(stageTitle(s)))
			continue
		}

		fmt.Fprintf(b, "    subgraph %s_group [%s]\n", nodeID, quotedLabel(stageTitle(s)))

		installed, err := r.pluginLoader.GetPluginByStringID(s.PluginID)
		if err != nil {
			fmt.Fprintf(b, "        %s[%s]\n", nodeID, quotedLabel("plugin not installed"))
			b.WriteString("    end\n")
			continue
		}

		fmt.Fprintf(b, "        %s(%s)\n", nodeID, quotedLabel(s.PluginID))

		if stepLines, ok := pluginStepDiagramLines(installed); ok {
			for _, line := range renamespaceMermaidLines(stepLines, fmt.Sprintf("S%d_step_", s.Index)) {
				fmt.Fprintf(b, "        %s\n", line)
			}
			b.WriteString("    end\n")
			continue
		}

		usesPorts[s.Index] = true
		boundInputs := map[string]bool{}
		for _, binding := range s.Bindings {
			boundInputs[binding.InputName] = true
		}

		inputNames := pluginInputNames(installed.Definition.Inputs)
		outputNames := pluginOutputNames(installed.Definition.Outputs)

		for _, name := range inputNames {
			portID := stagePortID(s.Index, "in", name)
			portLabel := name
			if !boundInputs[name] {
				if _, hasParam := s.Params[name]; hasParam {
					portLabel += " (param)"
				}
			}
			fmt.Fprintf(b, "        %s([%s])\n", portID, quotedLabel(portLabel))
			fmt.Fprintf(b, "        %s --> %s\n", portID, nodeID)
		}
		for _, name := range outputNames {
			portID := stagePortID(s.Index, "out", name)
			fmt.Fprintf(b, "        %s([%s])\n", portID, quotedLabel(name))
			fmt.Fprintf(b, "        %s --> %s\n", nodeID, portID)
		}

		b.WriteString("    end\n")
	}

	for _, s := range stages {
		for _, binding := range s.Bindings {
			from := stageNodeID(binding.SourceStage)
			fromIsPort := usesPorts[binding.SourceStage]
			if fromIsPort {
				from = stagePortID(binding.SourceStage, "out", binding.SourceOutput)
			}
			to := stageNodeID(s.Index)
			toIsPort := usesPorts[s.Index]
			if toIsPort {
				to = stagePortID(s.Index, "in", binding.InputName)
			}
			if fromIsPort || toIsPort {
				fmt.Fprintf(b, "    %s -.-> %s\n", from, to)
			} else {
				fmt.Fprintf(b, "    %s -->|%s| %s\n", from, quotedLabel(binding.SourceOutput), to)
			}
		}
	}
}

var mermaidFencePattern = regexp.MustCompile("(?s)```mermaid\r?\n(.*?)```")
var quotedSegmentPattern = regexp.MustCompile(`"[^"]*"`)
var mermaidIdentifierPattern = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_]*\b`)

// pluginStepDiagramLines reads a plugin's own generated Start/End/step
// workflow diagram (produced at doc-gen time from its `# @step` comments,
// committed as a fenced ```mermaid block in the plugin's README.md) so it
// can be nested inside the recipe diagram's expanded view. Returns false if
// the plugin has no README, or the README has no such block.
func pluginStepDiagramLines(installed *models.PluginV2) ([]string, bool) {
	if installed.FolderPath == "" {
		return nil, false
	}
	raw, err := os.ReadFile(filepath.Join(installed.FolderPath, "README.md"))
	if err != nil {
		return nil, false
	}

	match := mermaidFencePattern.FindSubmatch(raw)
	if match == nil {
		return nil, false
	}

	var body []string
	for i, line := range strings.Split(strings.TrimRight(string(match[1]), "\n"), "\n") {
		if i == 0 && strings.HasPrefix(strings.TrimSpace(line), "flowchart") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		body = append(body, line)
	}
	if len(body) == 0 {
		return nil, false
	}
	return body, true
}

// renamespaceMermaidLines prefixes every bare node identifier in lines with
// prefix, so a step diagram generated independently for one plugin can be
// nested inside a larger diagram without its "Start"/"End"/"stepN" node ids
// colliding with another stage's identically-named nodes. Quoted label text
// is left untouched. Start([Start])/End([End]) are normalized to a quoted
// label first, since their label text is otherwise indistinguishable from
// their bare node id.
func renamespaceMermaidLines(lines []string, prefix string) []string {
	normalized := make([]string, len(lines))
	for i, line := range lines {
		line = strings.ReplaceAll(line, "Start([Start])", `Start(["Start"])`)
		line = strings.ReplaceAll(line, "End([End])", `End(["End"])`)
		normalized[i] = line
	}

	ids := map[string]bool{}
	for _, line := range normalized {
		for _, seg := range quotedSegmentPattern.Split(line, -1) {
			for _, id := range mermaidIdentifierPattern.FindAllString(seg, -1) {
				ids[id] = true
			}
		}
	}

	renamed := make([]string, len(normalized))
	for i, line := range normalized {
		quoted := quotedSegmentPattern.FindAllString(line, -1)
		segments := quotedSegmentPattern.Split(line, -1)

		var b strings.Builder
		for j, seg := range segments {
			b.WriteString(mermaidIdentifierPattern.ReplaceAllStringFunc(seg, func(id string) string {
				if ids[id] {
					return prefix + id
				}
				return id
			}))
			if j < len(quoted) {
				b.WriteString(quoted[j])
			}
		}
		renamed[i] = b.String()
	}
	return renamed
}

func pluginInputNames(inputs []models.PluginInputV2) []string {
	names := make([]string, 0, len(inputs))
	for _, in := range inputs {
		names = append(names, in.Name)
	}
	sort.Strings(names)
	return names
}

func pluginOutputNames(outputs []models.PluginOutputV2) []string {
	names := make([]string, 0, len(outputs))
	for _, out := range outputs {
		names = append(names, out.Name)
	}
	sort.Strings(names)
	return names
}
