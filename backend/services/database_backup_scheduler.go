package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	dbBackupFilePrefix = "cauldron-"
	dbBackupFileSuffix = ".db"
)

// BackupTo writes a consistent point-in-time copy of the database to path using
// SQLite's VACUUM INTO, which is safe against a live WAL-mode connection. A plain
// file copy of cauldron.db can capture a torn write or miss data still sitting in
// the -wal file; VACUUM INTO produces a single self-contained, defragmented file.
func (d *DatabaseService) BackupTo(path string) error {
	return d.db.Exec("VACUUM INTO ?", path).Error
}

// DatabaseBackupScheduler periodically snapshots the database into a rotating set
// of timestamped files, so a corrupted cauldron.db can be recovered from a recent
// backup instead of losing everything.
type DatabaseBackupScheduler struct {
	db       *DatabaseService
	dir      string
	interval time.Duration
	keep     int
	stop     chan struct{}
	done     chan struct{}
}

func NewDatabaseBackupScheduler(db *DatabaseService, interval time.Duration, keep int) *DatabaseBackupScheduler {
	return &DatabaseBackupScheduler{
		db:       db,
		dir:      filepath.Join(db.GetDataDir(), "backups"),
		interval: interval,
		keep:     keep,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start runs an immediate backup if the newest existing one is already older than
// the interval (or none exists yet), then keeps backing up on that interval until
// Shutdown is called. Desktop sessions rarely stay open for a full interval, so
// this startup due-check is what actually keeps backups from going stale.
func (s *DatabaseBackupScheduler) Start() {
	go func() {
		defer close(s.done)

		if s.dueNow() {
			s.runBackup()
		}

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runBackup()
			case <-s.stop:
				return
			}
		}
	}()
}

func (s *DatabaseBackupScheduler) Shutdown() {
	close(s.stop)
	<-s.done
}

func (s *DatabaseBackupScheduler) dueNow() bool {
	newest, ok := s.newestBackupTime()
	if !ok {
		return true
	}
	return time.Since(newest) >= s.interval
}

func (s *DatabaseBackupScheduler) newestBackupTime() (time.Time, bool) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return time.Time{}, false
	}
	var newest time.Time
	found := false
	for _, e := range entries {
		if e.IsDir() || !isDBBackupFile(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !found || info.ModTime().After(newest) {
			newest = info.ModTime()
			found = true
		}
	}
	return newest, found
}

func (s *DatabaseBackupScheduler) runBackup() {
	if err := s.BackupNow(); err != nil {
		log.Printf("[DatabaseBackup] backup failed: %v", err)
	}
}

// BackupNow writes one timestamped snapshot immediately and rotates out anything
// past the retention count. Exposed separately from the scheduler loop so a
// manual "back up now" trigger can reuse the exact same path.
func (s *DatabaseBackupScheduler) BackupNow() error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	filename := dbBackupFilePrefix + time.Now().Format("20060102-150405") + dbBackupFileSuffix
	path := filepath.Join(s.dir, filename)

	if err := s.db.BackupTo(path); err != nil {
		return fmt.Errorf("failed to back up database: %w", err)
	}
	log.Printf("[DatabaseBackup] wrote %s", path)

	return s.rotate()
}

func (s *DatabaseBackupScheduler) rotate() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && isDBBackupFile(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	if len(names) <= s.keep {
		return nil
	}
	for _, name := range names[:len(names)-s.keep] {
		if err := os.Remove(filepath.Join(s.dir, name)); err != nil {
			log.Printf("[DatabaseBackup] failed to remove old backup %s: %v", name, err)
		}
	}
	return nil
}

func isDBBackupFile(name string) bool {
	return strings.HasPrefix(name, dbBackupFilePrefix) && strings.HasSuffix(name, dbBackupFileSuffix)
}
