package webrpkg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type PackageInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Repository   string   `json:"repository"`
	Dependencies []string `json:"dependencies"`
	Available    bool     `json:"available"`
}

type PackageCache struct {
	WebrVersion  string                 `json:"webr_version"`
	RVersion     string                 `json:"r_version"`
	RMajorMinor  string                 `json:"r_major_minor"`
	LastUpdated  time.Time              `json:"last_updated"`
	BasePackages []string               `json:"base_packages"`
	RepoPackages []string               `json:"repo_packages"`
	Packages     map[string]PackageInfo `json:"packages"`
}

func getCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	cacheDir := filepath.Join(homeDir, ".cache", "cauldron", "webrpkg")
	os.MkdirAll(cacheDir, 0755)
	return cacheDir
}

func getCachePath(webrVersion string) string {
	cacheDir := getCacheDir()
	if cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, fmt.Sprintf("webr-%s.json", webrVersion))
}

func loadCache(webrVersion string) (*PackageCache, error) {
	cachePath := getCachePath(webrVersion)
	if cachePath == "" {
		return nil, fmt.Errorf("cache directory not available")
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cache PackageCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	if time.Since(cache.LastUpdated) > 24*time.Hour {
		return nil, fmt.Errorf("cache expired")
	}

	return &cache, nil
}

func saveCache(cache *PackageCache) error {
	cachePath := getCachePath(cache.WebrVersion)
	if cachePath == "" {
		return fmt.Errorf("cache directory not available")
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

type ResolverResult struct {
	WebrVersion   string        `json:"webr_version"`
	RVersion      string        `json:"r_version"`
	RMajorMinor   string        `json:"r_major_minor"`
	RequestedPkgs []string      `json:"requested_packages"`
	ResolvedPkgs  []PackageInfo `json:"resolved_packages"`
	InstallOrder  []string      `json:"install_order"`
	Unavailable   []string      `json:"unavailable"`
	BaseRPackages []string      `json:"base_r_packages"`
}

type Resolver struct {
	client       *http.Client
	webrVersion  string
	rVersion     string
	rMajorMinor  string
	basePackages map[string]bool
	repoPackages map[string]bool
	biocPackages map[string]string
	pkgCache     map[string]*PackageInfo
	depGraph     map[string][]string
	mu           sync.RWMutex
	useCache     bool
	cacheLoaded  bool
}

func NewResolver(webrVersion string) (*Resolver, error) {
	return NewResolverWithCache(webrVersion, true)
}

func NewResolverWithCache(webrVersion string, useCache bool) (*Resolver, error) {
	r := &Resolver{
		client:       &http.Client{Timeout: 30 * time.Second},
		webrVersion:  webrVersion,
		basePackages: make(map[string]bool),
		repoPackages: make(map[string]bool),
		biocPackages: make(map[string]string),
		pkgCache:     make(map[string]*PackageInfo),
		depGraph:     make(map[string][]string),
		useCache:     useCache,
	}

	if useCache {
		if cache, err := loadCache(webrVersion); err == nil {
			r.rVersion = cache.RVersion
			r.rMajorMinor = cache.RMajorMinor
			for _, pkg := range cache.BasePackages {
				r.basePackages[pkg] = true
			}
			for _, pkg := range cache.RepoPackages {
				r.repoPackages[pkg] = true
			}
			for name, info := range cache.Packages {
				infoCopy := info
				r.pkgCache[name] = &infoCopy
				r.depGraph[name] = info.Dependencies
			}
			r.cacheLoaded = true
			return r, nil
		}
	}

	if err := r.detectRVersion(); err != nil {
		return nil, fmt.Errorf("failed to detect R version: %w", err)
	}

	if err := r.loadBasePackages(); err != nil {
		return nil, fmt.Errorf("failed to load base packages: %w", err)
	}

	if err := r.loadRepoPackages(); err != nil {
		return nil, fmt.Errorf("failed to load repo packages: %w", err)
	}

	return r, nil
}

func (r *Resolver) SaveCache() error {
	cache := &PackageCache{
		WebrVersion: r.webrVersion,
		RVersion:    r.rVersion,
		RMajorMinor: r.rMajorMinor,
		LastUpdated: time.Now(),
		Packages:    make(map[string]PackageInfo),
	}

	r.mu.RLock()
	for pkg := range r.basePackages {
		cache.BasePackages = append(cache.BasePackages, pkg)
	}
	for pkg := range r.repoPackages {
		cache.RepoPackages = append(cache.RepoPackages, pkg)
	}
	for name, info := range r.pkgCache {
		if info != nil {
			cache.Packages[name] = *info
		}
	}
	r.mu.RUnlock()

	return saveCache(cache)
}

func (r *Resolver) IsCacheLoaded() bool {
	return r.cacheLoaded
}

func (r *Resolver) GetRVersion() string {
	return r.rVersion
}

func (r *Resolver) GetRMajorMinor() string {
	return r.rMajorMinor
}

func (r *Resolver) detectRVersion() error {
	versions := []string{"4.5", "4.4", "4.3"}
	for _, ver := range versions {
		pkgURL := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/PACKAGES", ver)
		resp, err := r.client.Head(pkgURL)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			r.rVersion = ver + ".0"
			r.rMajorMinor = ver
			return nil
		}
	}

	return fmt.Errorf("could not detect R version for WebR %s", r.webrVersion)
}

func (r *Resolver) loadBasePackages() error {
	baseList := fmt.Sprintf("https://webr.r-wasm.org/v%s/vfs/usr/lib/R/library/", r.webrVersion)
	resp, err := r.client.Get(baseList)
	if err == nil && resp.StatusCode == http.StatusOK {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, `href="`) {
				start := strings.Index(line, `href="`) + 6
				end := strings.Index(line[start:], `"`)
				if end > 0 {
					pkg := strings.TrimSuffix(line[start:start+end], "/")
					if pkg != "" && pkg != ".." && !strings.HasPrefix(pkg, ".") {
						r.basePackages[pkg] = true
					}
				}
			}
		}
		resp.Body.Close()
	}

	defaultBase := []string{
		"base", "compiler", "datasets", "graphics", "grDevices", "grid",
		"methods", "parallel", "splines", "stats", "stats4", "tcltk",
		"tools", "utils",
	}
	for _, pkg := range defaultBase {
		r.basePackages[pkg] = true
	}

	return nil
}

func (r *Resolver) loadRepoPackages() error {
	url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/PACKAGES", r.rMajorMinor)
	resp, err := r.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch PACKAGES: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Package:") {
			pkg := strings.TrimSpace(strings.TrimPrefix(line, "Package:"))
			r.repoPackages[pkg] = true
		}
	}

	return nil
}

func (r *Resolver) IsBasePackage(pkg string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.basePackages[pkg]
}

func (r *Resolver) IsAvailable(pkg string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.basePackages[pkg] || r.repoPackages[pkg] || r.biocPackages[pkg] != ""
}

func (r *Resolver) GetRepository(pkg string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.basePackages[pkg] {
		return "base"
	}
	if r.repoPackages[pkg] {
		return "https://repo.r-wasm.org"
	}
	if repo, ok := r.biocPackages[pkg]; ok && repo != "" {
		return repo
	}
	return ""
}

func (r *Resolver) resolveDependencies(pkg string) (*PackageInfo, error) {
	r.mu.RLock()
	if cached, ok := r.pkgCache[pkg]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	info := &PackageInfo{
		Name:      pkg,
		Available: false,
	}

	if r.IsBasePackage(pkg) {
		info.Available = true
		info.Repository = "base"
		r.mu.Lock()
		r.pkgCache[pkg] = info
		r.mu.Unlock()
		return info, nil
	}

	r.mu.RLock()
	inRepo := r.repoPackages[pkg]
	r.mu.RUnlock()

	if inRepo {
		deps, version := r.getRepoPackageDeps(pkg)
		info.Available = true
		info.Repository = "https://repo.r-wasm.org"
		info.Version = version
		info.Dependencies = deps
		r.mu.Lock()
		r.pkgCache[pkg] = info
		r.depGraph[pkg] = deps
		r.mu.Unlock()
		return info, nil
	}

	deps, repo := r.getBiocPackageDeps(pkg)
	if repo != "" {
		info.Available = true
		info.Repository = repo
		info.Dependencies = deps
		r.mu.Lock()
		r.pkgCache[pkg] = info
		r.depGraph[pkg] = deps
		r.biocPackages[pkg] = repo
		r.mu.Unlock()
		return info, nil
	}

	deps, repo = r.getRUniversePackageDeps(pkg)
	if repo != "" {
		info.Available = true
		info.Repository = repo
		info.Dependencies = deps
		r.mu.Lock()
		r.pkgCache[pkg] = info
		r.depGraph[pkg] = deps
		r.mu.Unlock()
		return info, nil
	}

	r.mu.Lock()
	r.pkgCache[pkg] = info
	r.mu.Unlock()
	return info, nil
}

func (r *Resolver) getRepoPackageDeps(pkg string) ([]string, string) {
	url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/%s/%s/DESCRIPTION", r.rMajorMinor, pkg)
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}

	var deps []string
	var version string
	scanner := bufio.NewScanner(resp.Body)
	var currentField string
	var fieldValue strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if currentField == "Depends" || currentField == "Imports" || currentField == "LinkingTo" {
				fieldValue.WriteString(" ")
				fieldValue.WriteString(strings.TrimSpace(line))
			}
		} else if strings.Contains(line, ":") {
			if fieldValue.Len() > 0 {
				deps = append(deps, r.parseDepString(fieldValue.String())...)
			}
			parts := strings.SplitN(line, ":", 2)
			currentField = strings.TrimSpace(parts[0])
			fieldValue.Reset()
			if len(parts) > 1 {
				fieldValue.WriteString(strings.TrimSpace(parts[1]))
			}
			if currentField == "Version" && len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}
		}
	}

	if fieldValue.Len() > 0 {
		deps = append(deps, r.parseDepString(fieldValue.String())...)
	}

	return deps, version
}

func (r *Resolver) getBiocPackageDeps(pkg string) ([]string, string) {
	url := fmt.Sprintf("https://bioc.r-universe.dev/api/packages/%s", pkg)
	resp, err := r.client.Get(url)
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
			R  string `json:"r"`
		} `json:"_binaries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, ""
	}

	hasCompatibleWasm := false
	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			if bin.R == "" || strings.HasPrefix(bin.R, r.rMajorMinor) {
				hasCompatibleWasm = true
				break
			}
		}
	}

	if !hasCompatibleWasm {
		return nil, ""
	}

	var deps []string
	for _, dep := range info.Dependencies {
		if dep.Role == "Depends" || dep.Role == "Imports" || dep.Role == "LinkingTo" {
			if dep.Package != "R" && !r.IsBasePackage(dep.Package) {
				deps = append(deps, dep.Package)
			}
		}
	}

	return deps, "https://bioc.r-universe.dev"
}

func (r *Resolver) getRUniversePackageDeps(pkg string) ([]string, string) {
	url := fmt.Sprintf("https://r-universe.dev/api/packages/%s", pkg)
	resp, err := r.client.Get(url)
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
			R  string `json:"r"`
		} `json:"_binaries"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, ""
	}

	hasCompatibleWasm := false
	for _, bin := range info.Binaries {
		if bin.OS == "wasm" {
			if bin.R == "" || strings.HasPrefix(bin.R, r.rMajorMinor) {
				hasCompatibleWasm = true
				break
			}
		}
	}

	if !hasCompatibleWasm {
		return nil, ""
	}

	var deps []string
	for _, dep := range info.Dependencies {
		if dep.Role == "Depends" || dep.Role == "Imports" || dep.Role == "LinkingTo" {
			if dep.Package != "R" && !r.IsBasePackage(dep.Package) {
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

func (r *Resolver) parseDepString(depStr string) []string {
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
		if part != "" && part != "R" && !r.IsBasePackage(part) {
			deps = append(deps, part)
		}
	}
	return deps
}

func (r *Resolver) Resolve(packages []string) (*ResolverResult, error) {
	result := &ResolverResult{
		WebrVersion:   r.webrVersion,
		RVersion:      r.rVersion,
		RMajorMinor:   r.rMajorMinor,
		RequestedPkgs: packages,
	}

	r.mu.RLock()
	for pkg := range r.basePackages {
		result.BaseRPackages = append(result.BaseRPackages, pkg)
	}
	r.mu.RUnlock()
	sort.Strings(result.BaseRPackages)

	allPkgs := make(map[string]bool)
	var resolveRecursive func(pkg string) error
	resolveRecursive = func(pkg string) error {
		if allPkgs[pkg] || r.IsBasePackage(pkg) {
			return nil
		}
		allPkgs[pkg] = true

		info, err := r.resolveDependencies(pkg)
		if err != nil {
			return err
		}

		for _, dep := range info.Dependencies {
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

	unavailableSet := make(map[string]bool)
	r.mu.RLock()
	for pkg := range allPkgs {
		info := r.pkgCache[pkg]
		if info != nil && !info.Available {
			unavailableSet[pkg] = true
		}
	}
	r.mu.RUnlock()

	var propagateUnavailable func(pkg string) bool
	propagateUnavailable = func(pkg string) bool {
		if r.IsBasePackage(pkg) {
			return true
		}
		if unavailableSet[pkg] {
			return false
		}
		r.mu.RLock()
		info := r.pkgCache[pkg]
		r.mu.RUnlock()
		if info == nil {
			return false
		}
		for _, dep := range info.Dependencies {
			if !propagateUnavailable(dep) {
				unavailableSet[pkg] = true
				return false
			}
		}
		return true
	}

	for pkg := range allPkgs {
		propagateUnavailable(pkg)
	}

	r.mu.RLock()
	for pkg := range allPkgs {
		info := r.pkgCache[pkg]
		if info != nil {
			pkgCopy := *info
			if unavailableSet[pkg] {
				pkgCopy.Available = false
			}
			result.ResolvedPkgs = append(result.ResolvedPkgs, pkgCopy)
			if unavailableSet[pkg] {
				result.Unavailable = append(result.Unavailable, pkg)
			}
		}
	}
	r.mu.RUnlock()

	result.InstallOrder = r.topologicalSortFiltered(unavailableSet)

	if r.useCache && !r.cacheLoaded {
		r.SaveCache()
	}

	return result, nil
}

func (r *Resolver) topologicalSort() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inDegree := make(map[string]int)
	for node := range r.depGraph {
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
		for _, dep := range r.depGraph[node] {
			if !r.basePackages[dep] {
				inDegree[dep]++
			}
		}
	}

	var queue []string
	for node, degree := range inDegree {
		if degree == 0 && !r.basePackages[node] {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dep := range r.depGraph[node] {
			if r.basePackages[dep] {
				continue
			}
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

func (r *Resolver) topologicalSortFiltered(exclude map[string]bool) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inDegree := make(map[string]int)
	for node := range r.depGraph {
		if exclude[node] {
			continue
		}
		if _, exists := inDegree[node]; !exists {
			inDegree[node] = 0
		}
		for _, dep := range r.depGraph[node] {
			if !r.basePackages[dep] && !exclude[dep] {
				inDegree[dep]++
			}
		}
	}

	var queue []string
	for node, degree := range inDegree {
		if degree == 0 && !r.basePackages[node] && !exclude[node] {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dep := range r.depGraph[node] {
			if r.basePackages[dep] || exclude[dep] {
				continue
			}
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}

	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}
