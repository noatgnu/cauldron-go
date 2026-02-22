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

var webrBuiltinPackages = map[string]bool{
	"stats":        true,
	"graphics":     true,
	"grDevices":    true,
	"utils":        true,
	"datasets":     true,
	"methods":      true,
	"base":         true,
	"tools":        true,
	"parallel":     true,
	"compiler":     true,
	"Matrix":       true,
	"lattice":      true,
	"grid":         true,
	"splines":      true,
	"survival":     true,
	"MASS":         true,
	"class":        true,
	"nnet":         true,
	"spatial":      true,
	"cluster":      true,
	"codetools":    true,
	"foreign":      true,
	"KernSmooth":   true,
	"rpart":        true,
	"nlme":         true,
	"mgcv":         true,
	"boot":         true,
	"dplyr":        true,
	"tidyr":        true,
	"ggplot2":      true,
	"tidyverse":    true,
	"purrr":        true,
	"tibble":       true,
	"stringr":      true,
	"readr":        true,
	"forcats":      true,
	"lubridate":    true,
	"jsonlite":     true,
	"httr":         true,
	"xml2":         true,
	"rvest":        true,
	"haven":        true,
	"readxl":       true,
	"writexl":      true,
	"scales":       true,
	"viridis":      true,
	"RColorBrewer": true,
	"plotly":       true,
	"shiny":        true,
	"knitr":        true,
	"rmarkdown":    true,
	"testthat":     true,
	"devtools":     true,
	"roxygen2":     true,
	"pkgdown":      true,
	"usethis":      true,
	"cli":          true,
	"rlang":        true,
	"glue":         true,
	"vctrs":        true,
	"pillar":       true,
	"crayon":       true,
	"withr":        true,
	"remotes":      true,
	"pak":          true,
	"data.table":   true,
	"Rcpp":         true,
	"magrittr":     true,
}

var webrIncompatiblePackages = map[string]string{
	"RCurl":     "RCurl requires libcurl which has limited support in WebR",
	"rgdal":     "rgdal requires GDAL libraries not available in WebR",
	"rgeos":     "rgeos requires GEOS libraries not available in WebR",
	"rJava":     "rJava requires JVM which is not available in browsers",
	"RODBC":     "RODBC requires ODBC drivers not available in browsers",
	"RMySQL":    "RMySQL requires MySQL client libraries",
	"RPostgres": "RPostgres requires PostgreSQL client libraries",
	"odbc":      "odbc requires ODBC drivers not available in browsers",
}

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

	for _, pkg := range packages {
		pkgName := strings.TrimSpace(pkg)

		if reason, found := webrIncompatiblePackages[pkgName]; found {
			result.Issues = append(result.Issues, fmt.Sprintf("%s: %s", pkgName, reason))
			result.Unsupported = append(result.Unsupported, pkgName)
			result.Compatible = false
		} else if !webrBuiltinPackages[pkgName] {
			result.MaybeSupport = append(result.MaybeSupport, pkgName)
		}
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
		if webrBuiltinPackages[pkgName] {
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
	url := fmt.Sprintf("https://repo.r-wasm.org/bin/emscripten/contrib/4.4/%s/DESCRIPTION", pkg)
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
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

		if webrBuiltinPackages[pkgName] {
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
