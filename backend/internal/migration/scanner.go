package migration

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FileScanner scans for migration files
type FileScanner struct {
	fsys fs.FS
}

// NewFileScanner creates a new file scanner
func NewFileScanner(fsys fs.FS) *FileScanner {
	return &FileScanner{fsys: fsys}
}

// migrationFileRegex matches migration files like: 20250801120000_description.up.sql
var migrationFileRegex = regexp.MustCompile(`^(\d{14})_([^.]+)\.(up|down)\.sql$`)

// ScanMigrations scans for all migration files
func (s *FileScanner) ScanMigrations() ([]Migration, error) {
	migrationMap := make(map[string]*Migration)
	
	err := fs.WalkDir(s.fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		filename := filepath.Base(path)
		matches := migrationFileRegex.FindStringSubmatch(filename)
		if matches == nil {
			return nil
		}
		
		version := matches[1]
		name := matches[2]
		direction := matches[3]
		
		content, err := fs.ReadFile(s.fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}
		
		if _, exists := migrationMap[version]; !exists {
			migrationMap[version] = &Migration{
				Version: version,
				Name:    name,
			}
		}
		
		migration := migrationMap[version]
		script := string(content)
		
		if direction == "up" {
			migration.UpScript = script
		} else {
			migration.DownScript = script
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan migrations: %w", err)
	}
	
	// Convert map to slice and sort by version
	var migrations []Migration
	for _, m := range migrationMap {
		// Calculate checksum
		m.Checksum = calculateChecksum(m.UpScript + m.DownScript)
		migrations = append(migrations, *m)
	}
	
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	
	return migrations, nil
}

// calculateChecksum calculates MD5 checksum of the script
func calculateChecksum(script string) string {
	h := md5.New()
	h.Write([]byte(script))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ParseVersion extracts the timestamp from a version string
func ParseVersion(version string) (string, error) {
	if len(version) != 14 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}
	
	// Validate it's all digits
	for _, ch := range version {
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("version must contain only digits: %s", version)
		}
	}
	
	return version, nil
}

// FormatName formats the migration name for display
func FormatName(name string) string {
	// Replace underscores with spaces and capitalize words
	words := strings.Split(name, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}