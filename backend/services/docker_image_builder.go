package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type DockerImageBuilder struct {
	db              *DatabaseService
	settingsService *SettingsService
}

func NewDockerImageBuilder(db *DatabaseService, settingsService *SettingsService) *DockerImageBuilder {
	return &DockerImageBuilder{db: db, settingsService: settingsService}
}

// resolveEngine returns the container engine binary name ("docker" or
// "podman") from the live global setting, defaulting to "docker" when no
// settings service is wired (e.g. in tests that construct this directly).
func (d *DockerImageBuilder) resolveEngine() string {
	if d.settingsService == nil {
		return "docker"
	}
	return d.settingsService.GetContainerEngine()
}

func (d *DockerImageBuilder) CheckDockerAvailable() error {
	engine := d.resolveEngine()
	cmd := exec.Command(engine, "version")
	hideConsoleWindow(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s daemon not available: %w", engine, err)
	}
	return nil
}

func (d *DockerImageBuilder) BuildImage(
	pluginDir string,
	dockerfile string,
	imageName string,
	platform string,
	buildArgs map[string]string,
) error {
	args := []string{"build"}

	if platform != "" {
		args = append(args, "--platform", platform)
	}

	for key, value := range buildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	args = append(args, "-t", imageName)
	args = append(args, "-f", dockerfile)
	args = append(args, ".")

	engine := d.resolveEngine()
	cmd := exec.Command(engine, args...)
	hideConsoleWindow(cmd)
	cmd.Dir = pluginDir

	log.Printf("[DockerImageBuilder] Building image: %s %s", engine, strings.Join(args, " "))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s build failed: %w\nOutput: %s", engine, err, string(output))
	}

	log.Printf("[DockerImageBuilder] Successfully built image: %s", imageName)
	return nil
}

func (d *DockerImageBuilder) PullImage(imageName string) error {
	engine := d.resolveEngine()
	cmd := exec.Command(engine, "pull", imageName)
	hideConsoleWindow(cmd)

	log.Printf("[DockerImageBuilder] Pulling image: %s", imageName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s pull failed: %w\nOutput: %s", engine, err, string(output))
	}

	log.Printf("[DockerImageBuilder] Successfully pulled image: %s", imageName)
	return nil
}

func (d *DockerImageBuilder) ImageExists(imageName string) bool {
	cmd := exec.Command(d.resolveEngine(), "image", "inspect", imageName)
	hideConsoleWindow(cmd)
	err := cmd.Run()
	return err == nil
}

func (d *DockerImageBuilder) GetDockerfileHash(dockerfilePath string) (string, error) {
	data, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

func (d *DockerImageBuilder) RecordBuiltImage(
	pluginID string,
	imageName string,
	dockerfileHash string,
	platform string,
	buildArgs map[string]string,
) error {
	cmd := exec.Command(d.resolveEngine(), "image", "inspect", "--format={{.Id}}", imageName)
	hideConsoleWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get image ID: %w", err)
	}
	imageID := strings.TrimSpace(string(output))

	buildArgsJSON, _ := json.Marshal(buildArgs)

	image := &PluginDockerImage{
		PluginID:       pluginID,
		ImageName:      imageName,
		ImageID:        imageID,
		Built:          dockerfileHash != "",
		DockerfileHash: dockerfileHash,
		Platform:       platform,
		BuildArgs:      string(buildArgsJSON),
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}

	return d.db.SavePluginDockerImage(image)
}

func (d *DockerImageBuilder) ShouldRebuildImage(pluginID string, dockerfilePath string) (bool, error) {
	existingImage, err := d.db.GetPluginDockerImage(pluginID)
	if err != nil {
		return true, nil
	}

	if !existingImage.Built {
		return false, nil
	}

	currentHash, err := d.GetDockerfileHash(dockerfilePath)
	if err != nil {
		return true, err
	}

	if currentHash != existingImage.DockerfileHash {
		log.Printf("[DockerImageBuilder] Dockerfile changed for plugin %s, rebuild required", pluginID)
		return true, nil
	}

	if !d.ImageExists(existingImage.ImageName) {
		log.Printf("[DockerImageBuilder] Image %s not found locally, rebuild required", existingImage.ImageName)
		return true, nil
	}

	return false, nil
}
