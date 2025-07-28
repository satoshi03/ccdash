package services

import (
	"sync"
	"time"
)

type InitializationStatus string

const (
	StatusInitializing InitializationStatus = "initializing"
	StatusCompleted    InitializationStatus = "completed"
	StatusFailed       InitializationStatus = "failed"
)

type InitializationState struct {
	Status    InitializationStatus `json:"status"`
	Message   string              `json:"message"`
	Progress  *ProgressInfo       `json:"progress,omitempty"`
	StartTime time.Time           `json:"start_time"`
	EndTime   *time.Time          `json:"end_time,omitempty"`
	Error     *string             `json:"error,omitempty"`
}

type ProgressInfo struct {
	ProcessedFiles int `json:"processed_files"`
	TotalFiles     int `json:"total_files"`
	NewLines       int `json:"new_lines"`
}

type InitializationService struct {
	mu    sync.RWMutex
	state InitializationState
}

var globalInitService *InitializationService

func init() {
	globalInitService = &InitializationService{
		state: InitializationState{
			Status:    StatusCompleted,
			Message:   "System ready",
			StartTime: time.Now(),
		},
	}
}

func GetGlobalInitializationService() *InitializationService {
	return globalInitService
}

func (s *InitializationService) StartInitialization() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = InitializationState{
		Status:    StatusInitializing,
		Message:   "Initializing database and syncing logs...",
		StartTime: time.Now(),
	}
}

func (s *InitializationService) UpdateProgress(processedFiles, totalFiles, newLines int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Status == StatusInitializing {
		s.state.Progress = &ProgressInfo{
			ProcessedFiles: processedFiles,
			TotalFiles:     totalFiles,
			NewLines:       newLines,
		}
		s.state.Message = "Syncing logs..."
	}
}

func (s *InitializationService) CompleteInitialization(processedFiles, newLines int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.state = InitializationState{
		Status:    StatusCompleted,
		Message:   "Database initialization completed successfully",
		StartTime: s.state.StartTime,
		EndTime:   &now,
		Progress: &ProgressInfo{
			ProcessedFiles: processedFiles,
			NewLines:       newLines,
		},
	}
}

func (s *InitializationService) FailInitialization(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	errorMsg := err.Error()
	s.state = InitializationState{
		Status:    StatusFailed,
		Message:   "Database initialization failed",
		StartTime: s.state.StartTime,
		EndTime:   &now,
		Error:     &errorMsg,
	}
}

func (s *InitializationService) GetState() InitializationState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *InitializationService) IsInitializing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Status == StatusInitializing
}