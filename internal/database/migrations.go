package database

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type MigrationRunner struct {
	db           *gorm.DB
	migrationsDir string
}

func NewMigrationRunner(db *gorm.DB, migrationsDir string) *MigrationRunner {
	return &MigrationRunner{db: db, migrationsDir: migrationsDir}
}

func (r *MigrationRunner) Up() error {
	files, err := os.ReadDir(r.migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		path := filepath.Join(r.migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if err := r.db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("executing %s: %w", name, err)
		}
	}

	return nil
}

func (r *MigrationRunner) Down() error {
	files, err := os.ReadDir(r.migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	var downFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".down.sql") {
			downFiles = append(downFiles, f.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(downFiles)))

	for _, name := range downFiles {
		path := filepath.Join(r.migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}
		if err := r.db.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("executing %s: %w", name, err)
		}
	}

	return nil
}
