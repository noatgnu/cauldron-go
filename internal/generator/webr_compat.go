package generator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/noatgnu/cauldron-go/backend/models"
)

var webrPackageCache = struct {
	loaded       bool
	baseR        map[string]bool
	repoPackages map[string]bool
}{
	baseR:        make(map[string]bool),
	repoPackages: make(map[string]bool),
}

func loadWebrPackageCache() {
	if webrPackageCache.loaded {
		return
	}
	webrPackageCache.loaded = true

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get("https://webr.r-wasm.org/latest/vfs/usr/lib/R/library/.Rinstignore")
	if err == nil && resp.StatusCode == http.StatusOK {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			pkg := strings.TrimSpace(scanner.Text())
			if pkg != "" && !strings.HasPrefix(pkg, "#") {
				webrPackageCache.baseR[pkg] = true
			}
		}
		resp.Body.Close()
	}

	baseRPkgs := []string{"base", "compiler", "datasets", "graphics", "grDevices", "grid", "methods", "parallel", "splines", "stats", "stats4", "tcltk", "tools", "utils"}
	for _, pkg := range baseRPkgs {
		webrPackageCache.baseR[pkg] = true
	}

	versions := []string{"4.5", "4.4"}
	for _, ver := range versions {
		url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/PACKAGES", ver)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Package:") {
				pkg := strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
				webrPackageCache.repoPackages[pkg] = true
			}
		}
		resp.Body.Close()
		break
	}
}

func isBaseRPackage(pkg string) bool {
	loadWebrPackageCache()
	return webrPackageCache.baseR[pkg]
}

func isWebrRepoPackage(pkg string) bool {
	loadWebrPackageCache()
	return webrPackageCache.repoPackages[pkg]
}

var webrIncompatiblePackages = map[string]string{}

type WebRCompatibility struct {
	Compatible        bool
	Issues            []string
	Packages          []string
	Unsupported       []string
	MaybeSupport      []string
	BiocPackages      []string
	AvailableOnline   []string
	UnavailableOnline []string
}

func CheckWebRCompatibility(definition *models.PluginDefinition, pluginDir string) WebRCompatibility {
	result := WebRCompatibility{
		Compatible:        true,
		Issues:            []string{},
		Packages:          []string{},
		Unsupported:       []string{},
		MaybeSupport:      []string{},
		BiocPackages:      []string{},
		AvailableOnline:   []string{},
		UnavailableOnline: []string{},
	}

	if !definition.Runtime.HasEnvironment("r") {
		result.Issues = append(result.Issues, "Plugin does not support R runtime")
		result.Compatible = false
		return result
	}

	if definition.Runtime.HasEnvironment("python") && !definition.Runtime.HasEnvironment("r") {
		result.Issues = append(result.Issues, "Plugin is Python-only, use Pyodide instead")
		result.Compatible = false
		return result
	}

	packages := getRRequiredPackages(definition, pluginDir)
	result.Packages = packages

	available, unavailable := CheckWebRPackageAvailability(packages)
	result.AvailableOnline = available
	result.UnavailableOnline = unavailable

	for _, pkg := range unavailable {
		result.Issues = append(result.Issues, fmt.Sprintf("%s: package not available in WebR repositories", pkg))
		result.Unsupported = append(result.Unsupported, pkg)
		result.Compatible = false
	}

	for _, input := range definition.Inputs {
		if input.Type == "directory" {
			result.Issues = append(result.Issues, "Plugin uses directory input which is not well-supported in browsers")
		}
	}

	return result
}

func CheckWebRPackageAvailability(packages []string) (available []string, unavailable []string) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, pkg := range packages {
		pkgName := strings.TrimSpace(pkg)
		if isBaseRPackage(pkgName) {
			available = append(available, pkgName)
			continue
		}

		if checkWebRRepo(client, pkgName) {
			available = append(available, pkgName)
		} else if hasWasm, _ := checkBiocRUniverseWasm(client, pkgName); hasWasm {
			available = append(available, pkgName)
		} else if hasWasm, _ := checkRUniverseWasm(client, pkgName); hasWasm {
			available = append(available, pkgName)
		} else {
			unavailable = append(unavailable, pkgName)
		}
	}

	return available, unavailable
}

func checkBiocRUniverseWasm(client *http.Client, pkg string) (bool, string) {
	url := fmt.Sprintf("https://bioc.r-universe.dev/api/packages/%s", pkg)
	resp, err := client.Get(url)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	var info struct {
		Package  string `json:"Package"`
		Version  string `json:"Version"`
		Binaries []struct {
			OS   string `json:"os"`
			Arch string `json:"arch"`
		} `json:"_binaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return false, ""
	}

	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			return true, "https://bioc.r-universe.dev"
		}
	}
	return false, ""
}

func checkRUniverseWasm(client *http.Client, pkg string) (bool, string) {
	url := fmt.Sprintf("https://r-universe.dev/api/packages/%s", pkg)
	resp, err := client.Get(url)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, ""
	}

	var info struct {
		Package  string `json:"Package"`
		Version  string `json:"Version"`
		Binaries []struct {
			OS   string `json:"os"`
			Arch string `json:"arch"`
		} `json:"_binaries"`
		Owner string `json:"_owner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return false, ""
	}

	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			return true, fmt.Sprintf("https://%s.r-universe.dev", info.Owner)
		}
	}
	return false, ""
}

func checkWebRRepo(client *http.Client, pkg string) bool {
	versions := []string{"4.5", "4.4"}
	for _, ver := range versions {
		url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/%s/DESCRIPTION", ver, pkg)
		resp, err := client.Head(url)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return true
		}
	}
	return false
}

func getRRequiredPackages(definition *models.PluginDefinition, pluginDir string) []string {
	seen := make(map[string]bool)
	var packages []string

	addPackage := func(pkg string) {
		pkgName := strings.TrimSpace(pkg)
		pkgLower := strings.ToLower(pkgName)

		if !seen[pkgLower] {
			seen[pkgLower] = true
			packages = append(packages, pkgName)
		}
	}

	defaultRPkgPath := filepath.Join(pluginDir, "r-packages.txt")
	if pkgs, err := parseRPackagesFile(defaultRPkgPath); err == nil {
		for _, pkg := range pkgs {
			addPackage(pkg)
		}
	}

	if definition.Execution.Requirements.RPackagesFile != "" {
		reqPath := definition.Execution.Requirements.RPackagesFile
		if !filepath.IsAbs(reqPath) {
			reqPath = filepath.Join(pluginDir, reqPath)
		}

		if pkgs, err := parseRPackagesFile(reqPath); err == nil {
			for _, pkg := range pkgs {
				addPackage(pkg)
			}
		}
	}

	for _, pkg := range definition.Execution.Requirements.Packages {
		addPackage(pkg)
	}

	return packages
}

func parseRPackagesFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		packages = append(packages, line)
	}

	return packages, scanner.Err()
}

type PackageAvailability struct {
	Name      string
	Available bool
	RepoURL   string
}

func CheckWebRPackageAvailabilityWithRepos(packages []string) []PackageAvailability {
	client := &http.Client{Timeout: 10 * time.Second}
	var results []PackageAvailability

	for _, pkg := range packages {
		pkgName := strings.TrimSpace(pkg)
		result := PackageAvailability{Name: pkgName, Available: false}

		if isBaseRPackage(pkgName) {
			result.Available = true
			result.RepoURL = "https://repo.r-wasm.org"
		} else if checkWebRRepo(client, pkgName) {
			result.Available = true
			result.RepoURL = "https://repo.r-wasm.org"
		} else if hasWasm, repoURL := checkBiocRUniverseWasm(client, pkgName); hasWasm {
			result.Available = true
			result.RepoURL = repoURL
		} else if hasWasm, repoURL := checkRUniverseWasm(client, pkgName); hasWasm {
			result.Available = true
			result.RepoURL = repoURL
		}

		results = append(results, result)
	}

	return results
}

type PackageInfo struct {
	Name         string
	RepoURL      string
	Dependencies []string
}

func ResolvePackageDependencies(packages []string) ([]PackageAvailability, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	allPackages := make(map[string]*PackageInfo)
	dependencyGraph := make(map[string][]string)

	var resolveRecursive func(pkg string) error
	resolveRecursive = func(pkg string) error {
		pkgName := strings.TrimSpace(pkg)
		if pkgName == "" {
			return nil
		}

		if isBaseRPackage(pkgName) {
			return nil
		}

		if _, exists := allPackages[pkgName]; exists {
			return nil
		}

		info := &PackageInfo{Name: pkgName}
		allPackages[pkgName] = info

		deps, repoURL := getPackageDependencies(client, pkgName)
		info.RepoURL = repoURL
		info.Dependencies = deps
		dependencyGraph[pkgName] = deps

		for _, dep := range deps {
			if err := resolveRecursive(dep); err != nil {
				return err
			}
		}

		return nil
	}

	for _, pkg := range packages {
		if err := resolveRecursive(pkg); err != nil {
			return nil, err
		}
	}

	ordered := topologicalSort(dependencyGraph)

	var results []PackageAvailability
	seen := make(map[string]bool)

	for _, pkgName := range ordered {
		if seen[pkgName] || isBaseRPackage(pkgName) {
			continue
		}
		seen[pkgName] = true

		info := allPackages[pkgName]
		result := PackageAvailability{
			Name:      pkgName,
			Available: info.RepoURL != "",
			RepoURL:   info.RepoURL,
		}
		results = append(results, result)
	}

	for _, pkg := range packages {
		pkgName := strings.TrimSpace(pkg)
		if !seen[pkgName] && !isBaseRPackage(pkgName) {
			seen[pkgName] = true
			info := allPackages[pkgName]
			repoURL := ""
			if info != nil {
				repoURL = info.RepoURL
			}
			results = append(results, PackageAvailability{
				Name:      pkgName,
				Available: repoURL != "",
				RepoURL:   repoURL,
			})
		}
	}

	return results, nil
}

func getPackageDependencies(client *http.Client, pkg string) ([]string, string) {
	if checkWebRRepo(client, pkg) {
		deps := getWebRRepoDependencies(client, pkg)
		return deps, "https://repo.r-wasm.org"
	}

	deps, repoURL := getBiocDependencies(client, pkg)
	if repoURL != "" {
		return deps, repoURL
	}

	deps, repoURL = getCRANDependencies(client, pkg)
	if repoURL != "" {
		return deps, repoURL
	}

	return nil, ""
}

func getWebRRepoDependencies(client *http.Client, pkg string) []string {
	versions := []string{"4.5", "4.4"}
	for _, ver := range versions {
		url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/%s/DESCRIPTION", ver, pkg)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		var deps []string
		var currentField string
		var fieldValue strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				if currentField == "Depends" || currentField == "Imports" || currentField == "LinkingTo" {
					fieldValue.WriteString(strings.TrimSpace(line))
				}
			} else if strings.Contains(line, ":") {
				if currentField != "" && fieldValue.Len() > 0 {
					deps = append(deps, parseDepString(fieldValue.String())...)
				}
				parts := strings.SplitN(line, ":", 2)
				currentField = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					fieldValue.Reset()
					fieldValue.WriteString(strings.TrimSpace(parts[1]))
				}
			}
		}

		if currentField != "" && fieldValue.Len() > 0 {
			deps = append(deps, parseDepString(fieldValue.String())...)
		}

		return deps
	}
	return nil
}

func getBiocDependencies(client *http.Client, pkg string) ([]string, string) {
	url := fmt.Sprintf("https://bioc.r-universe.dev/api/packages/%s", pkg)
	resp, err := client.Get(url)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var info struct {
		Package      string `json:"Package"`
		Dependencies []struct {
			Package string `json:"package"`
			Role    string `json:"role"`
		} `json:"_dependencies"`
		Binaries []struct {
			OS string `json:"os"`
		} `json:"_binaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, ""
	}

	hasWasm := false
	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			hasWasm = true
			break
		}
	}
	if !hasWasm {
		return nil, ""
	}

	var deps []string
	for _, dep := range info.Dependencies {
		if dep.Role == "Depends" || dep.Role == "Imports" || dep.Role == "LinkingTo" {
			if dep.Package != "R" && !isBaseRPackage(dep.Package) {
				deps = append(deps, dep.Package)
			}
		}
	}

	return deps, "https://bioc.r-universe.dev"
}

func getCRANDependencies(client *http.Client, pkg string) ([]string, string) {
	url := fmt.Sprintf("https://r-universe.dev/api/packages/%s", pkg)
	resp, err := client.Get(url)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var info struct {
		Package      string `json:"Package"`
		Owner        string `json:"_owner"`
		Dependencies []struct {
			Package string `json:"package"`
			Role    string `json:"role"`
		} `json:"_dependencies"`
		Binaries []struct {
			OS string `json:"os"`
		} `json:"_binaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, ""
	}

	hasWasm := false
	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			hasWasm = true
			break
		}
	}
	if !hasWasm {
		return nil, ""
	}

	var deps []string
	for _, dep := range info.Dependencies {
		if dep.Role == "Depends" || dep.Role == "Imports" || dep.Role == "LinkingTo" {
			if dep.Package != "R" && !isBaseRPackage(dep.Package) {
				deps = append(deps, dep.Package)
			}
		}
	}

	repoURL := "https://repo.r-wasm.org"
	if info.Owner != "" {
		repoURL = fmt.Sprintf("https://%s.r-universe.dev", info.Owner)
	}

	return deps, repoURL
}

func parseDepString(depStr string) []string {
	if depStr == "" {
		return nil
	}

	var deps []string
	parts := strings.Split(depStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "("); idx > 0 {
			part = strings.TrimSpace(part[:idx])
		}
		if part != "" && part != "R" && !isBaseRPackage(part) {
			deps = append(deps, part)
		}
	}
	return deps
}

func topologicalSort(graph map[string][]string) []string {
	inDegree := make(map[string]int)
	for node := range graph {
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
		for _, dep := range graph[node] {
			inDegree[dep]++
		}
	}

	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dep := range graph[node] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
