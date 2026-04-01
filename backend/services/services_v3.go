package services

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func NewDatabaseServiceV3(userDataPath string) (*DatabaseService, error) {
	return newDatabaseServiceFromPath(userDataPath)
}

func NewSettingsServiceV3(db *DatabaseService) *SettingsService {
	return newSettingsServiceInternal(db)
}

func NewFileServiceV3(wailsApp *application.App) *FileService {
	return &FileService{
		wailsApp: wailsApp,
	}
}

func NewEnvironmentServiceV3(db *DatabaseService, settingsService *SettingsService, progressNotifier *ProgressNotifier) *EnvironmentService {
	return &EnvironmentService{
		db:               db,
		settingsService:  settingsService,
		progressNotifier: progressNotifier,
		packageCache:     make(map[string]*packageCacheEntry),
		cacheTTL:         5 * time.Minute,
	}
}

func NewPortableEnvServiceV3(fileService *FileService) *PortableEnvService {
	return &PortableEnvService{
		fileService: fileService,
	}
}

func NewJobQueueServiceV3(db *DatabaseService, wailsApp *application.App) *JobQueueService {
	service := newJobQueueServiceInternal(db, context.Background())
	service.wailsApp = wailsApp
	return service
}

type ProgressNotifierV3 struct {
	wailsApp *application.App
}

func NewProgressNotifierV3(wailsApp *application.App) *ProgressNotifier {
	return &ProgressNotifier{
		wailsApp: wailsApp,
	}
}

func NewPluginInstallerV3(pluginsDir string, db *DatabaseService, pluginLoader *PluginLoaderV2, gitAuthService *GitAuthService, wailsApp *application.App) *PluginInstaller {
	return &PluginInstaller{
		pluginsDir:   pluginsDir,
		db:           db,
		pluginLoader: pluginLoader,
		gitAuth:      gitAuthService,
		wailsApp:     wailsApp,
	}
}

func NewPluginRegistryServiceV3(configService *SettingsService, gitAuth *GitAuthService) *PluginRegistryService {
	return &PluginRegistryService{
		configService: configService,
		gitAuth:       gitAuth,
	}
}

func NewProtocolHandlerV3(installer *PluginInstaller, wailsApp *application.App) *ProtocolHandler {
	return &ProtocolHandler{
		installer: installer,
		wailsApp:  wailsApp,
	}
}
