package services

import (
	"context"
	"database/sql"
	"github.com/noatgnu/cauldron-go/backend/models"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
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

func NewDatabaseService(ctx context.Context) (*DatabaseService, error) {
	userConfigDir, _ := os.UserConfigDir()
	dbDir := filepath.Join(userConfigDir, "cauldron")
	os.MkdirAll(dbDir, 0755)

	dbPath := filepath.Join(dbDir, "cauldron.db")

	log.Printf("[Database] Opening database at: %s\n", dbPath)

	// Add SQLite-specific settings to prevent hangs
	dsn := dbPath + "?_busy_timeout=5000&_journal_mode=WAL"

	// Open using modernc.org/sqlite driver explicitly
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Printf("[Database] ERROR: Failed to open SQL connection: %v\n", err)
		return nil, err
	}

	log.Println("[Database] SQL connection opened successfully")

	// Wrap with GORM
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("[Database] ERROR: Failed to initialize GORM: %v\n", err)
		sqlDB.Close()
		return nil, err
	}

	log.Println("[Database] GORM initialized successfully")

	// SQLite doesn't handle concurrent writes well, so limit to 1 connection
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	log.Println("[Database] Connection pool configured")

	service := &DatabaseService{
		ctx: ctx,
		db:  db,
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
	return d.db.AutoMigrate(
		&Setting{},
		&ImportedFile{},
		&VirtualEnvironment{},
		&PythonEnvironmentDB{},
		&REnvironmentDB{},
		&models.Job{},
	)
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
