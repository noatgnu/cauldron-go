package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

type TestAPI struct {
	app *App
}

func NewTestAPI(app *App) *TestAPI {
	return &TestAPI{app: app}
}

func (t *TestAPI) Start(port int) {
	if os.Getenv("CAULDRON_TEST_MODE") != "true" {
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/test/health", t.handleHealth)
	mux.HandleFunc("/test/settings", t.handleSettings)
	mux.HandleFunc("/test/python-environments", t.handlePythonEnvironments)
	mux.HandleFunc("/test/r-environments", t.handleREnvironments)
	mux.HandleFunc("/test/virtual-environments", t.handleVirtualEnvironments)
	mux.HandleFunc("/test/create-venv", t.handleCreateVenv)
	mux.HandleFunc("/test/delete-venv", t.handleDeleteVenv)
	mux.HandleFunc("/test/renv-environments", t.handleRenvEnvironments)
	mux.HandleFunc("/test/create-renv", t.handleCreateRenv)
	mux.HandleFunc("/test/delete-renv", t.handleDeleteRenv)
	mux.HandleFunc("/test/plugins", t.handlePlugins)
	mux.HandleFunc("/test/plugin-bindings", t.handlePluginBindings)
	mux.HandleFunc("/test/bind-plugin-environment", t.handleBindPluginEnvironment)
	mux.HandleFunc("/test/jobs", t.handleJobs)
	mux.HandleFunc("/test/imported-files", t.handleImportedFiles)
	mux.HandleFunc("/test/default-venv-path", t.handleDefaultVenvPath)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("[TestAPI] Starting test API server on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("[TestAPI] Server error: %v", err)
		}
	}()
}

func (t *TestAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	})
}

func (t *TestAPI) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "App not initialized yet",
		})
		return
	}

	settings := t.app.GetSettings()
	json.NewEncoder(w).Encode(settings)
}

func (t *TestAPI) handlePythonEnvironments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        "App not initialized yet",
		})
		return
	}

	envs, err := t.app.DetectPythonEnvironments()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environments": envs,
		"count":        len(envs),
	})
}

func (t *TestAPI) handleREnvironments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        "App not initialized yet",
		})
		return
	}

	envs, err := t.app.DetectREnvironments()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environments": envs,
		"count":        len(envs),
	})
}

func (t *TestAPI) handleVirtualEnvironments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        "App not initialized yet",
		})
		return
	}

	venvs, err := t.app.GetVirtualEnvironments()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environments": venvs,
		"count":        len(venvs),
	})
}

func (t *TestAPI) handleCreateVenv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case <-t.app.initialized:
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "App not initialized yet",
		})
		return
	}

	var req struct {
		BasePythonPath string `json:"basePythonPath"`
		VenvPath       string `json:"venvPath"`
		PluginID       string `json:"pluginId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TestAPI] Creating venv: basePython=%s, venvPath=%s, pluginId=%s", req.BasePythonPath, req.VenvPath, req.PluginID)

	err := t.app.CreatePythonVirtualEnv(req.BasePythonPath, req.VenvPath, req.PluginID)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[TestAPI] Create venv error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[TestAPI] Venv created successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (t *TestAPI) handleDeleteVenv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case <-t.app.initialized:
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "App not initialized yet",
		})
		return
	}

	var req struct {
		ID uint `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TestAPI] Deleting venv: id=%d", req.ID)

	err := t.app.DeleteVirtualEnvironment(req.ID)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[TestAPI] Delete venv error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[TestAPI] Venv deleted successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (t *TestAPI) handleRenvEnvironments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        "App not initialized yet",
		})
		return
	}

	renvs, err := t.app.GetRenvEnvironments()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environments": []interface{}{},
			"error":        err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"environments": renvs,
		"count":        len(renvs),
	})
}

func (t *TestAPI) handleCreateRenv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case <-t.app.initialized:
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "App not initialized yet",
		})
		return
	}

	var req struct {
		Name     string   `json:"name"`
		Packages []string `json:"packages"`
		PluginID string   `json:"pluginId"`
		UseCache bool     `json:"useCache"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TestAPI] Creating renv: name=%s, packages=%v, pluginId=%s, useCache=%v", req.Name, req.Packages, req.PluginID, req.UseCache)

	err := t.app.CreateRenvEnvironment(req.Name, req.Packages, req.PluginID, req.UseCache)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[TestAPI] Create renv error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[TestAPI] Renv created successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (t *TestAPI) handleDeleteRenv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case <-t.app.initialized:
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "App not initialized yet",
		})
		return
	}

	var req struct {
		ID uint `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TestAPI] Deleting renv: id=%d", req.ID)

	err := t.app.DeleteRenvEnvironment(req.ID)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[TestAPI] Delete renv error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[TestAPI] Renv deleted successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (t *TestAPI) handlePlugins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"plugins": []interface{}{},
			"error":   "App not initialized yet",
		})
		return
	}

	plugins := t.app.GetPluginsV2()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"plugins": plugins,
		"count":   len(plugins),
	})
}

func (t *TestAPI) handlePluginBindings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bindings": []interface{}{},
			"error":    "App not initialized yet",
		})
		return
	}

	bindings, err := t.app.GetAllPluginEnvironmentBindings()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"bindings": []interface{}{},
			"error":    err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"bindings": bindings,
		"count":    len(bindings),
	})
}

func (t *TestAPI) handleBindPluginEnvironment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	select {
	case <-t.app.initialized:
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "App not initialized yet",
		})
		return
	}

	var req struct {
		PluginID string `json:"pluginId"`
		EnvType  string `json:"envType"`
		EnvID    uint   `json:"envId"`
		EnvPath  string `json:"envPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[TestAPI] Binding plugin %s to %s environment %d", req.PluginID, req.EnvType, req.EnvID)

	err := t.app.BindPluginToEnvironment(req.PluginID, req.EnvType, req.EnvID, req.EnvPath)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[TestAPI] Bind plugin error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	log.Printf("[TestAPI] Plugin bound successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (t *TestAPI) handleJobs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jobs":  []interface{}{},
			"error": "App not initialized yet",
		})
		return
	}

	jobs := t.app.GetAllJobs()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func (t *TestAPI) handleImportedFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"files": []interface{}{},
			"error": "App not initialized yet",
		})
		return
	}

	files, err := t.app.GetImportedFiles()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"files": []interface{}{},
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

func (t *TestAPI) handleDefaultVenvPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	select {
	case <-t.app.initialized:
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":  "",
			"error": "App not initialized yet",
		})
		return
	}

	pluginID := r.URL.Query().Get("pluginId")
	if pluginID == "" {
		pluginID = "test-" + strconv.FormatInt(int64(os.Getpid()), 10)
	}

	path, err := t.app.GetDefaultVenvPath(pluginID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":  "",
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"path": path,
	})
}
