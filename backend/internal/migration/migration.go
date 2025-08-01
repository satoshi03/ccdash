package migration

import (
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version    string
	Name       string
	UpScript   string
	DownScript string
	Checksum   string
}

// MigrationHistory represents a migration execution record
type MigrationHistory struct {
	ID              int
	Version         string
	Name            string
	AppliedAt       time.Time
	ExecutionTimeMs int64
	Checksum        string
	UpScript        string
	DownScript      string
	Status          string
	ErrorMessage    string
}

// Direction represents the migration direction
type Direction int

const (
	Up Direction = iota
	Down
)

// String returns the string representation of the direction
func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	default:
		return "unknown"
	}
}