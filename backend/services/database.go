package services

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/backend/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

type DatabaseService struct {
	ctx context.Context
	db  *gorm.DB
}

type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string `gorm:"not null"`
}

type ImportedFile struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"not null"`
	Path       string `gorm:"not null"`
	Size       int64  `gorm:"not null"`
	ImportedAt int64  `gorm:"not null"`
	FileType   string
	Preview    string
}

type VirtualEnvironment struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"not null"`
	Path           string `gorm:"not null;unique"`
	BasePythonPath string `gorm:"not null"`
	CreatedAt      int64  `gorm:"not null"`
	Source         string `gorm:"default:'venv'"`
}

type PythonEnvironmentDB struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Path        string `gorm:"not null;unique"`
	Type        string `gorm:"not null"`
	Version     string `gorm:"not null"`
	IsVirtual   bool   `gorm:"not null"`
	IsActive    bool   `gorm:"not null;default:false"`
	HasPackages bool   `gorm:"not null;default:false"`
	CreatedAt   int64  `gorm:"autoCreateTime"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"`
}

type REnvironmentDB struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Path        string `gorm:"not null;unique"`
	Type        string `gorm:"not null"`
	Version     string `gorm:"not null"`
	IsActive    bool   `gorm:"not null;default:false"`
	HasPackages bool   `gorm:"not null;default:false"`
	CreatedAt   int64  `gorm:"autoCreateTime"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"`
}

type RenvEnvironment struct {
	ID             uint   `gorm:"primaryKey"`
	Name           string `gorm:"not null"`
	Path           string `gorm:"not null;unique"`
	ProjectPath    string `gorm:"not null"`
	BaseRPath      string `gorm:"not null"`
	RenvVersion    string
	UseGlobalCache bool  `gorm:"not null;default:false"`
	CreatedAt      int64 `gorm:"not null"`
}

type PluginEnvironmentBinding struct {
	ID              uint   `gorm:"primaryKey"`
	PluginID        string `gorm:"not null;index"`
	EnvironmentType string `gorm:"not null"`
	EnvironmentID   uint   `gorm:"not null"`
	EnvironmentPath string `gorm:"not null"`
	CreatedAt       int64  `gorm:"autoCreateTime"`
	UpdatedAt       int64  `gorm:"autoUpdateTime"`
}

type CustomEnvVar struct {
	ID        uint   `gorm:"primaryKey"`
	PluginID  uint   `gorm:"index"` // 0 if Global
	Key       string `gorm:"not null"`
	Value     string `gorm:"not null"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

type GitAuthConfig struct {
	ID               uint   `gorm:"primaryKey"`
	RepositoryURL    string `gorm:"not null;uniqueIndex"`
	SSHKeyPath       string `gorm:"not null"`
	SSHKeyPassphrase string `gorm:""`
	CreatedAt        int64  `gorm:"autoCreateTime"`
	UpdatedAt        int64  `gorm:"autoUpdateTime"`
}

type PluginDockerImage struct {
	ID             uint   `gorm:"primaryKey"`
	PluginID       string `gorm:"not null;index;uniqueIndex:idx_plugin_docker"`
	ImageName      string `gorm:"not null"`
	ImageID        string `gorm:"not null"`
	Built          bool   `gorm:"not null;default:false"`
	DockerfileHash string `gorm:""`
	Platform       string `gorm:""`
	BuildArgs      string `gorm:""`
	CreatedAt      int64  `gorm:"autoCreateTime"`
	UpdatedAt      int64  `gorm:"autoUpdateTime"`
}

func NewDatabaseService(ctx context.Context) (*DatabaseService, error) {
	userConfigDir, _ := os.UserConfigDir()
	dbDir := filepath.Join(userConfigDir, "cauldron")
	return newDatabaseServiceFromPath(dbDir)
}

func newDatabaseServiceFromPath(dbDir string) (*DatabaseService, error) {
	os.MkdirAll(dbDir, 0755)

	dbPath := filepath.Join(dbDir, "cauldron.db")

	log.Printf("[Database] Opening database at: %s\n", dbPath)

	dsn := dbPath + "?_busy_timeout=5000&_journal_mode=WAL"

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Printf("[Database] ERROR: Failed to open SQL connection: %v\n", err)
		return nil, err
	}

	log.Println("[Database] SQL connection opened successfully")

	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("[Database] ERROR: Failed to initialize GORM: %v\n", err)
		sqlDB.Close()
		return nil, err
	}

	log.Println("[Database] GORM initialized successfully")

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	log.Println("[Database] Connection pool configured")

	service := &DatabaseService{
		db: db,
	}

	log.Println("[Database] Running auto-migration...")
	if err := service.autoMigrate(); err != nil {
		log.Printf("[Database] ERROR: Auto-migration failed: %v\n", err)
		return nil, err
	}

	log.Println("[Database] Database service initialized successfully")
	return service, nil
}

func (d *DatabaseService) autoMigrate() error {
	if err := d.db.AutoMigrate(
		&Setting{},
		&ImportedFile{},
		&VirtualEnvironment{},
		&RenvEnvironment{},
		&PluginEnvironmentBinding{},
		&PythonEnvironmentDB{},
		&REnvironmentDB{},
		&CustomEnvVar{},
		&GitAuthConfig{},
		&PluginDockerImage{},
		&models.Job{},
		&models.PluginRegistry{},
	); err != nil {
		return err
	}

	if err := d.db.Model(&models.PluginRegistry{}).Where("enabled = ?", false).Update("enabled", true).Error; err != nil {
		log.Printf("[Database] Warning: Failed to update plugin enabled defaults: %v", err)
	}

	return nil
}

func (d *DatabaseService) GetDB() *gorm.DB {
	return d.db
}

func (d *DatabaseService) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *DatabaseService) SaveSetting(key, value string) error {
	return d.db.Where("key = ?", key).
		Assign(Setting{Key: key, Value: value}).
		FirstOrCreate(&Setting{}).Error
}

func (d *DatabaseService) GetSetting(key string) (string, error) {
	var setting Setting
	err := d.db.Where("key = ?", key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return setting.Value, err
}

func (d *DatabaseService) GetAllSettings() (map[string]string, error) {
	var settings []Setting
	if err := d.db.Find(&settings).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}

	return result, nil
}

func (d *DatabaseService) SaveImportedFile(file *ImportedFile) error {
	return d.db.Create(file).Error
}

func (d *DatabaseService) GetImportedFiles() ([]ImportedFile, error) {
	var files []ImportedFile
	err := d.db.Order("imported_at DESC").Find(&files).Error
	return files, err
}

func (d *DatabaseService) DeleteImportedFile(id uint) error {
	return d.db.Delete(&ImportedFile{}, id).Error
}

func (d *DatabaseService) SavePythonEnvironment(env PythonEnvironment) error {
	dbEnv := PythonEnvironmentDB{
		Name:        env.Name,
		Path:        env.Path,
		Type:        env.Type,
		Version:     env.Version,
		IsVirtual:   env.IsVirtual,
		HasPackages: env.HasPackages,
	}

	result := d.db.Where("path = ?", env.Path).FirstOrCreate(&dbEnv)
	if result.Error != nil {
		return result.Error
	}

	return d.db.Model(&dbEnv).Updates(map[string]interface{}{
		"name":         env.Name,
		"type":         env.Type,
		"version":      env.Version,
		"is_virtual":   env.IsVirtual,
		"has_packages": env.HasPackages,
	}).Error
}

func (d *DatabaseService) GetPythonEnvironments() ([]PythonEnvironment, error) {
	var dbEnvs []PythonEnvironmentDB
	err := d.db.Order("is_active DESC, created_at DESC").Find(&dbEnvs).Error
	if err != nil {
		return nil, err
	}

	envs := make([]PythonEnvironment, len(dbEnvs))
	for i, dbEnv := range dbEnvs {
		envs[i] = PythonEnvironment{
			Name:        dbEnv.Name,
			Path:        dbEnv.Path,
			Type:        dbEnv.Type,
			Version:     dbEnv.Version,
			IsVirtual:   dbEnv.IsVirtual,
			HasPackages: dbEnv.HasPackages,
		}
	}
	return envs, nil
}

func (d *DatabaseService) SetActivePythonEnvironment(path string) error {
	tx := d.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&PythonEnvironmentDB{}).Where("1=1").Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&PythonEnvironmentDB{}).Where("path = ?", path).Update("is_active", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (d *DatabaseService) GetActivePythonEnvironment() (*PythonEnvironment, error) {
	var dbEnv PythonEnvironmentDB
	err := d.db.Where("is_active = ?", true).First(&dbEnv).Error
	if err != nil {
		return nil, err
	}

	return &PythonEnvironment{
		Name:        dbEnv.Name,
		Path:        dbEnv.Path,
		Type:        dbEnv.Type,
		Version:     dbEnv.Version,
		IsVirtual:   dbEnv.IsVirtual,
		HasPackages: dbEnv.HasPackages,
	}, nil
}

func (d *DatabaseService) SaveREnvironment(env REnvironment) error {
	dbEnv := REnvironmentDB{
		Name:        env.Name,
		Path:        env.Path,
		Type:        env.Type,
		Version:     env.Version,
		HasPackages: env.HasPackages,
	}

	result := d.db.Where("path = ?", env.Path).FirstOrCreate(&dbEnv)
	if result.Error != nil {
		return result.Error
	}

	return d.db.Model(&dbEnv).Updates(map[string]interface{}{
		"name":         env.Name,
		"type":         env.Type,
		"version":      env.Version,
		"has_packages": env.HasPackages,
	}).Error
}

func (d *DatabaseService) GetREnvironments() ([]REnvironment, error) {
	var dbEnvs []REnvironmentDB
	err := d.db.Order("is_active DESC, created_at DESC").Find(&dbEnvs).Error
	if err != nil {
		return nil, err
	}

	envs := make([]REnvironment, len(dbEnvs))
	for i, dbEnv := range dbEnvs {
		envs[i] = REnvironment{
			Name:        dbEnv.Name,
			Path:        dbEnv.Path,
			Type:        dbEnv.Type,
			Version:     dbEnv.Version,
			HasPackages: dbEnv.HasPackages,
		}
	}
	return envs, nil
}

func (d *DatabaseService) SetActiveREnvironment(path string) error {
	tx := d.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&REnvironmentDB{}).Where("1=1").Update("is_active", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&REnvironmentDB{}).Where("path = ?", path).Update("is_active", true).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (d *DatabaseService) GetActiveREnvironment() (*REnvironment, error) {
	var dbEnv REnvironmentDB
	err := d.db.Where("is_active = ?", true).First(&dbEnv).Error
	if err != nil {
		return nil, err
	}

	return &REnvironment{
		Name:        dbEnv.Name,
		Path:        dbEnv.Path,
		Type:        dbEnv.Type,
		Version:     dbEnv.Version,
		HasPackages: dbEnv.HasPackages,
	}, nil
}

func (d *DatabaseService) SaveRenvEnvironment(env RenvEnvironment) error {
	var existing RenvEnvironment
	result := d.db.Where("path = ?", env.Path).First(&existing)

	if result.Error == nil {
		return d.db.Model(&existing).Updates(env).Error
	}

	return d.db.Create(&env).Error
}

func (d *DatabaseService) GetRenvEnvironments() ([]RenvEnvironment, error) {
	var envs []RenvEnvironment
	err := d.db.Order("created_at DESC").Find(&envs).Error
	return envs, err
}

func (d *DatabaseService) DeleteRenvEnvironment(id uint) error {
	return d.db.Delete(&RenvEnvironment{}, id).Error
}

func (d *DatabaseService) GetRenvEnvironmentByID(id uint) (*RenvEnvironment, error) {
	var env RenvEnvironment
	err := d.db.First(&env, id).Error
	return &env, err
}

func (d *DatabaseService) SavePluginEnvironmentBinding(binding PluginEnvironmentBinding) error {
	var existing PluginEnvironmentBinding
	result := d.db.Where("plugin_id = ? AND environment_type = ?", binding.PluginID, binding.EnvironmentType).First(&existing)

	if result.Error == nil {
		return d.db.Model(&existing).Updates(binding).Error
	}

	return d.db.Create(&binding).Error
}

func (d *DatabaseService) GetPluginEnvironmentBinding(pluginID string, envType string) (*PluginEnvironmentBinding, error) {
	var binding PluginEnvironmentBinding
	err := d.db.Where("plugin_id = ? AND environment_type = ?", pluginID, envType).First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (d *DatabaseService) DeletePluginEnvironmentBinding(pluginID string, envType string) error {
	return d.db.Where("plugin_id = ? AND environment_type = ?", pluginID, envType).Delete(&PluginEnvironmentBinding{}).Error
}

func (d *DatabaseService) GetAllPluginEnvironmentBindings() ([]PluginEnvironmentBinding, error) {
	var bindings []PluginEnvironmentBinding
	err := d.db.Order("created_at DESC").Find(&bindings).Error
	return bindings, err
}

func (d *DatabaseService) SaveCustomEnvVar(envVar CustomEnvVar) error {
	var existing CustomEnvVar
	result := d.db.Where("plugin_id = ? AND key = ?", envVar.PluginID, envVar.Key).First(&existing)

	if result.Error == nil {
		return d.db.Model(&existing).Updates(envVar).Error
	}

	return d.db.Create(&envVar).Error
}

func (d *DatabaseService) GetCustomEnvVars(pluginID uint) ([]CustomEnvVar, error) {
	var envVars []CustomEnvVar
	err := d.db.Where("plugin_id = ?", pluginID).Find(&envVars).Error
	return envVars, err
}

func (d *DatabaseService) GetGlobalCustomEnvVars() ([]CustomEnvVar, error) {
	return d.GetCustomEnvVars(0)
}

func (d *DatabaseService) DeleteCustomEnvVar(id uint) error {
	return d.db.Delete(&CustomEnvVar{}, id).Error
}

func (d *DatabaseService) DeleteCustomEnvVarByKey(pluginID uint, key string) error {
	return d.db.Where("plugin_id = ? AND key = ?", pluginID, key).Delete(&CustomEnvVar{}).Error
}

func (d *DatabaseService) SavePluginDockerImage(image *PluginDockerImage) error {
	var existing PluginDockerImage
	result := d.db.Where("plugin_id = ?", image.PluginID).First(&existing)

	if result.Error == nil {
		image.ID = existing.ID
		return d.db.Save(image).Error
	}

	return d.db.Create(image).Error
}

func (d *DatabaseService) GetPluginDockerImage(pluginID string) (*PluginDockerImage, error) {
	var image PluginDockerImage
	if err := d.db.Where("plugin_id = ?", pluginID).First(&image).Error; err != nil {
		return nil, err
	}
	return &image, nil
}

func (d *DatabaseService) DeletePluginDockerImage(pluginID string) error {
	return d.db.Where("plugin_id = ?", pluginID).Delete(&PluginDockerImage{}).Error
}
