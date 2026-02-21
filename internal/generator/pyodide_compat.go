package generator

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/noatgnu/cauldron-go/backend/models"
)

var pyodideBuiltinPackages = map[string]bool{
	"numpy":              true,
	"pandas":             true,
	"scipy":              true,
	"scikit-learn":       true,
	"sklearn":            true,
	"matplotlib":         true,
	"seaborn":            true,
	"biopython":          true,
	"networkx":           true,
	"sympy":              true,
	"pillow":             true,
	"PIL":                true,
	"statsmodels":        true,
	"click":              true,
	"pyyaml":             true,
	"yaml":               true,
	"requests":           true,
	"beautifulsoup4":     true,
	"bs4":                true,
	"lxml":               true,
	"regex":              true,
	"jsonschema":         true,
	"packaging":          true,
	"pyparsing":          true,
	"pytz":               true,
	"six":                true,
	"certifi":            true,
	"charset-normalizer": true,
	"idna":               true,
	"urllib3":            true,
	"cycler":             true,
	"kiwisolver":         true,
	"fonttools":          true,
	"contourpy":          true,
	"joblib":             true,
	"threadpoolctl":      true,
	"jinja2":             true,
	"markupsafe":         true,
	"openpyxl":           true,
	"xlrd":               true,
	"xlsxwriter":         true,
	"pyarrow":            true,
	"fastparquet":        true,
	"h5py":               true,
	"tables":             true,
	"sqlalchemy":         true,
	"psycopg2":           true,
	"pymysql":            true,
	"cryptography":       true,
	"pycryptodome":       true,
	"cffi":               true,
	"pycparser":          true,
	"msgpack":            true,
	"protobuf":           true,
	"grpcio":             true,
	"aiohttp":            true,
	"yarl":               true,
	"multidict":          true,
	"async-timeout":      true,
	"frozenlist":         true,
	"aiosignal":          true,
	"attrs":              true,
	"dateutil":           true,
	"python-dateutil":    true,
	"tqdm":               true,
	"colorama":           true,
	"termcolor":          true,
	"tabulate":           true,
	"humanize":           true,
}

var incompatiblePackages = map[string]string{
	"torch":         "PyTorch is too large and has native dependencies not supported by Pyodide",
	"tensorflow":    "TensorFlow has native dependencies not supported by Pyodide",
	"keras":         "Keras depends on TensorFlow which is not supported",
	"opencv-python": "OpenCV has native dependencies not supported by Pyodide",
	"cv2":           "OpenCV has native dependencies not supported by Pyodide",
	"dask":          "Dask requires multiprocessing which is limited in browsers",
	"ray":           "Ray requires multiprocessing and distributed computing not available in browsers",
	"pyspark":       "PySpark requires JVM and distributed computing not available in browsers",
	"numba":         "Numba requires LLVM which is not available in Pyodide",
	"pyqt5":         "PyQt5 requires native GUI libraries",
	"pyqt6":         "PyQt6 requires native GUI libraries",
	"tkinter":       "Tkinter requires native GUI libraries",
	"wx":            "wxPython requires native GUI libraries",
	"gtk":           "GTK requires native GUI libraries",
	"kivy":          "Kivy requires native GUI libraries",
}

type PyodideCompatibility struct {
	Compatible   bool
	Issues       []string
	Packages     []string
	Unsupported  []string
	MaybeSupport []string
}

func CheckPyodideCompatibility(definition *models.PluginDefinition, pluginDir string) PyodideCompatibility {
	result := PyodideCompatibility{
		Compatible:   true,
		Issues:       []string{},
		Packages:     []string{},
		Unsupported:  []string{},
		MaybeSupport: []string{},
	}

	if !definition.Runtime.HasEnvironment("python") {
		result.Issues = append(result.Issues, "Plugin does not support Python runtime")
		result.Compatible = false
		return result
	}

	if definition.Runtime.HasEnvironment("r") {
		result.Issues = append(result.Issues, "Plugin requires R runtime which is not supported by Pyodide")
		result.Compatible = false
	}

	packages := getRequiredPackages(definition, pluginDir)
	result.Packages = packages

	for _, pkg := range packages {
		pkgLower := strings.ToLower(pkg)
		pkgName := strings.Split(pkgLower, "==")[0]
		pkgName = strings.Split(pkgName, ">=")[0]
		pkgName = strings.Split(pkgName, "<=")[0]
		pkgName = strings.Split(pkgName, "<")[0]
		pkgName = strings.Split(pkgName, ">")[0]
		pkgName = strings.TrimSpace(pkgName)

		if reason, found := incompatiblePackages[pkgName]; found {
			result.Issues = append(result.Issues, reason)
			result.Unsupported = append(result.Unsupported, pkgName)
			result.Compatible = false
		} else if !pyodideBuiltinPackages[pkgName] {
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

func getRequiredPackages(definition *models.PluginDefinition, pluginDir string) []string {
	var packages []string

	if len(definition.Execution.Requirements.Packages) > 0 {
		packages = append(packages, definition.Execution.Requirements.Packages...)
	}

	if definition.Execution.Requirements.PythonRequirementsFile != "" {
		reqPath := definition.Execution.Requirements.PythonRequirementsFile
		if !filepath.IsAbs(reqPath) {
			reqPath = filepath.Join(pluginDir, reqPath)
		}

		pkgs, err := parseRequirementsFile(reqPath)
		if err == nil {
			packages = append(packages, pkgs...)
		}
	} else if len(packages) == 0 {
		defaultReqPath := filepath.Join(pluginDir, "requirements.txt")
		if pkgs, err := parseRequirementsFile(defaultReqPath); err == nil {
			packages = append(packages, pkgs...)
		}
	}

	return packages
}

func parseRequirementsFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		packages = append(packages, line)
	}

	return packages, scanner.Err()
}
