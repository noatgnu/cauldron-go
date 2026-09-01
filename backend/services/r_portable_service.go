package services

import (
	"fmt"
	"path/filepath"

	"github.com/noatgnu/cookeR/rversion"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// RPortableService manages portable R installations in-process via cookeR's rversion package.
type RPortableService struct {
	manager          *rversion.Manager
	progressNotifier *ProgressNotifier
}

// NewRPortableServiceV3 constructs an RPortableService storing installed R versions under the app's data folder.
func NewRPortableServiceV3(wailsApp *application.App) (*RPortableService, error) {
	appFolder, err := getAppDataFolder()
	if err != nil {
		return nil, err
	}
	installDir := filepath.Join(appFolder, "bin", "r-portable")
	return &RPortableService{
		manager:          rversion.New(installDir),
		progressNotifier: NewProgressNotifierV3(wailsApp),
	}, nil
}

// ListAvailableRVersions returns R versions installable for the current platform.
func (s *RPortableService) ListAvailableRVersions() ([]rversion.Release, error) {
	return s.manager.ListAvailableRVersions()
}

// ListInstalledRVersions returns R versions already installed.
func (s *RPortableService) ListInstalledRVersions() ([]string, error) {
	return s.manager.ListInstalledRVersions()
}

// InstallRVersion downloads, verifies, and installs the given R version, reporting progress.
func (s *RPortableService) InstallRVersion(version string) error {
	releases, err := s.manager.ListAvailableRVersions()
	if err != nil {
		return fmt.Errorf("failed to list available R versions: %w", err)
	}

	var target *rversion.Release
	for i := range releases {
		if releases[i].Version == version {
			target = &releases[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("R version %q is not available for this platform", version)
	}

	id := "r-portable-" + version
	s.progressNotifier.EmitStart(ProgressTypeInstall, id, fmt.Sprintf("Installing R %s", version))
	if err := s.manager.InstallRVersion(*target, func(msg string) {
		s.progressNotifier.EmitProgress(ProgressTypeInstall, id, msg, 0)
	}); err != nil {
		s.progressNotifier.EmitError(ProgressTypeInstall, id, "R installation failed", err.Error())
		return err
	}
	s.progressNotifier.EmitComplete(ProgressTypeInstall, id, fmt.Sprintf("R %s installed", version))
	return nil
}

// UninstallRVersion removes an installed R version.
func (s *RPortableService) UninstallRVersion(version string) error {
	return s.manager.UninstallRVersion(version)
}

// GetRPath returns the Rscript path for an installed R version.
func (s *RPortableService) GetRPath(version string) (string, error) {
	return s.manager.GetRPath(version)
}
