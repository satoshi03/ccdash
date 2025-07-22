package models

import (
	"time"
)

// FileProcessingState represents the state of file processing for differential sync
type FileProcessingState struct {
	FilePath          string    `json:"file_path" db:"file_path"`
	LastModified      time.Time `json:"last_modified" db:"last_modified"`
	FileSize          int64     `json:"file_size" db:"file_size"`
	LastProcessedLine int       `json:"last_processed_line" db:"last_processed_line"`
	ProcessedUntil    *time.Time `json:"processed_until" db:"processed_until"`
	Checksum          *string   `json:"checksum" db:"checksum"`
	SyncStatus        string    `json:"sync_status" db:"sync_status"` // pending, processing, completed, error
	LastSyncTime      time.Time `json:"last_sync_time" db:"last_sync_time"`
	ErrorMessage      *string   `json:"error_message" db:"error_message"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// FileInfo represents basic file information
type FileInfo struct {
	Path    string
	ModTime time.Time
	Size    int64
}

// SyncStats represents synchronization statistics
type SyncStats struct {
	TotalFiles       int           `json:"total_files"`
	ProcessedFiles   int           `json:"processed_files"`
	SkippedFiles     int           `json:"skipped_files"`
	NewLines         int           `json:"new_lines"`
	ProcessingTime   time.Duration `json:"processing_time"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
}