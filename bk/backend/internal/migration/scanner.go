package migration

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MigrationFile represents a single migration file
type MigrationFile struct {
	Version    string
	Name       string
	Direction  string // "up" or "down"
	Filename   string
	Content    string
}

// MigrationPair represents a pair of up/down migration files
type MigrationPair struct {
	Version string
	Name    string
	Up      *MigrationFile
	Down    *MigrationFile
}

// Scanner scans for migration files
type Scanner struct {
	fsys fs.FS
}

// NewScanner creates a new migration scanner
func NewScanner(fsys fs.FS) *Scanner {
	return &Scanner{fsys: fsys}
}

// NewEmbedScanner creates a scanner for embedded files
func NewEmbedScanner(embedFS embed.FS) *Scanner {
	return &Scanner{fsys: embedFS}
}

// Scan scans for all migration files
func (s *Scanner) Scan() ([]MigrationPair, error) {
	files := make(map[string]*MigrationPair)
	
	err := fs.WalkDir(s.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		
		filename := filepath.Base(path)
		version, name, err := ParseVersionFromFilename(filename)
		if err != nil {
			// Skip files that don't match the pattern
			return nil
		}
		
		content, err := fs.ReadFile(s.fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}
		
		migFile := &MigrationFile{
			Version:   version,
			Name:      name,
			Filename:  filename,
			Content:   string(content),
		}
		
		if strings.HasSuffix(filename, ".up.sql") {
			migFile.Direction = "up"
			if pair, ok := files[version]; ok {
				pair.Up = migFile
			} else {
				files[version] = &MigrationPair{
					Version: version,
					Name:    name,
					Up:      migFile,
				}
			}
		} else if strings.HasSuffix(filename, ".down.sql") {
			migFile.Direction = "down"
			if pair, ok := files[version]; ok {
				pair.Down = migFile
			} else {
				files[version] = &MigrationPair{
					Version: version,
					Name:    name,
					Down:    migFile,
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan migration files: %w", err)
	}
	
	// Convert map to sorted slice
	var pairs []MigrationPair
	for _, pair := range files {
		pairs = append(pairs, *pair)
	}
	
	// Sort by version
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Version < pairs[j].Version
	})
	
	return pairs, nil
}

// ScanPending scans for migrations that haven't been applied yet
func (s *Scanner) ScanPending(vm *VersionManager) ([]MigrationPair, error) {
	allMigrations, err := s.Scan()
	if err != nil {
		return nil, err
	}
	
	var pending []MigrationPair
	for _, migration := range allMigrations {
		applied, err := vm.IsApplied(migration.Version)
		if err != nil {
			return nil, err
		}
		if !applied {
			pending = append(pending, migration)
		}
	}
	
	return pending, nil
}