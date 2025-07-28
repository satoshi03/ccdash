package services

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"claudeee-backend/internal/config"
	"claudeee-backend/internal/models"
)

type DiffSyncService struct {
	db             *sql.DB
	tokenService   *TokenService
	sessionService *SessionService
	windowService  *SessionWindowService
	stateManager   *FileSyncStateManager
}

func NewDiffSyncService(db *sql.DB, tokenService *TokenService, sessionService *SessionService) *DiffSyncService {
	stateManager := NewFileSyncStateManager(db)
	windowService := NewSessionWindowService(db)
	return &DiffSyncService{
		db:             db,
		tokenService:   tokenService,
		sessionService: sessionService,
		windowService:  windowService,
		stateManager:   stateManager,
	}
}

// InitializeSchema initializes the database schema for differential sync
func (d *DiffSyncService) InitializeSchema() error {
	return d.stateManager.InitializeSchema()
}

// SyncAllLogs performs differential synchronization of all logs
func (d *DiffSyncService) SyncAllLogs() (*models.SyncStats, error) {
	stats := &models.SyncStats{
		StartTime: time.Now(),
	}

	// Initialize schema if needed
	if err := d.InitializeSchema(); err != nil {
		return stats, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Clean up old states for deleted files
	if err := d.stateManager.CleanupOldStates(); err != nil {
		log.Printf("Warning: failed to cleanup old states: %v", err)
	}

	// Discover all JSONL files
	files, err := d.discoverJSONLFiles()
	if err != nil {
		return stats, fmt.Errorf("failed to discover JSONL files: %w", err)
	}

	stats.TotalFiles = len(files)
	log.Printf("Found %d JSONL files to process", len(files))

	// Process each file
	for _, file := range files {
		needsSync, lastState, err := d.stateManager.NeedsProcessing(file.Path)
		if err != nil {
			log.Printf("Error checking file %s: %v", file.Path, err)
			continue
		}

		if needsSync {
			newLines, err := d.syncFile(file, lastState)
			if err != nil {
				log.Printf("Error syncing file %s: %v", file.Path, err)
				// Update state with error
				errorMsg := err.Error()
				errorState := &models.FileProcessingState{
					FilePath:     file.Path,
					LastModified: file.ModTime,
					FileSize:     file.Size,
					SyncStatus:   "error",
					ErrorMessage: &errorMsg,
				}
				d.stateManager.UpdateFileState(errorState)
				continue
			}
			stats.ProcessedFiles++
			stats.NewLines += newLines
		} else {
			stats.SkippedFiles++
		}
	}

	stats.EndTime = time.Now()
	stats.ProcessingTime = stats.EndTime.Sub(stats.StartTime)

	log.Printf("Sync completed: %d files processed, %d skipped, %d new lines, took %v",
		stats.ProcessedFiles, stats.SkippedFiles, stats.NewLines, stats.ProcessingTime)

	return stats, nil
}

// discoverJSONLFiles discovers all JSONL files in Claude projects directory
func (d *DiffSyncService) discoverJSONLFiles() ([]models.FileInfo, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	claudeDir := cfg.ClaudeProjectsDir
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("claude projects directory not found: %s", claudeDir)
	}

	var files []models.FileInfo

	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read claude projects directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := filepath.Join(claudeDir, entry.Name())
		jsonlFiles, err := filepath.Glob(filepath.Join(projectPath, "*.jsonl"))
		if err != nil {
			log.Printf("Warning: failed to glob files in %s: %v", projectPath, err)
			continue
		}

		for _, jsonlFile := range jsonlFiles {
			fileInfo, err := os.Stat(jsonlFile)
			if err != nil {
				log.Printf("Warning: failed to stat file %s: %v", jsonlFile, err)
				continue
			}
			files = append(files, models.FileInfo{
				Path:    jsonlFile,
				ModTime: fileInfo.ModTime(),
				Size:    fileInfo.Size(),
			})
		}
	}

	return files, nil
}

// syncFile syncs a single file, processing only new lines
func (d *DiffSyncService) syncFile(file models.FileInfo, lastState *models.FileProcessingState) (int, error) {
	// Update state to processing
	processingState := &models.FileProcessingState{
		FilePath:     file.Path,
		LastModified: file.ModTime,
		FileSize:     file.Size,
		SyncStatus:   "processing",
	}

	// Keep the previous processed line count if available
	if lastState != nil {
		processingState.LastProcessedLine = lastState.LastProcessedLine
	}

	err := d.stateManager.UpdateFileState(processingState)
	if err != nil {
		return 0, fmt.Errorf("failed to update processing state: %w", err)
	}

	// Process file from the last processed line
	startLine := 0
	if lastState != nil {
		startLine = lastState.LastProcessedLine
	}

	newLines, totalLines, err := d.processFileFromLine(file.Path, startLine)
	if err != nil {
		return 0, fmt.Errorf("failed to process file: %w", err)
	}

	// Update state to completed
	now := time.Now()
	completedState := &models.FileProcessingState{
		FilePath:          file.Path,
		LastModified:      file.ModTime,
		FileSize:          file.Size,
		LastProcessedLine: totalLines,
		ProcessedUntil:    &now,
		SyncStatus:        "completed",
	}

	err = d.stateManager.UpdateFileState(completedState)
	if err != nil {
		return newLines, fmt.Errorf("failed to update completed state: %w", err)
	}

	return newLines, nil
}

// processFileFromLine processes a file starting from a specific line
func (d *DiffSyncService) processFileFromLine(filePath string, startLine int) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	
	// Increase buffer size to handle very long lines (up to 10MB)
	const maxCapacity = 10 * 1024 * 1024 // 10MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
	lineCount := 0
	processedCount := 0

	// Skip already processed lines
	for lineCount < startLine && scanner.Scan() {
		lineCount++
	}

	// Process new lines
	for scanner.Scan() {
		lineCount++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// First, try to parse as a basic JSON to check if it has required fields
		var basicCheck map[string]interface{}
		if err := json.Unmarshal([]byte(line), &basicCheck); err != nil {
			continue
		}

		// Check if this looks like a LogEntry (has sessionId and timestamp)
		sessionId, hasSessionId := basicCheck["sessionId"]
		timestamp, hasTimestamp := basicCheck["timestamp"]
		if !hasSessionId || !hasTimestamp || sessionId == nil || timestamp == nil {
			// Skip non-LogEntry entries (like summary entries)
			continue
		}

		var entry models.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Extract project name from file path
		projectName := d.extractProjectNameFromPath(filePath)
		if err := d.processLogEntry(&entry, projectName); err != nil {
			log.Printf("Error processing log entry at line %d: %v", lineCount, err)
			continue
		}
		processedCount++
	}

	if err := scanner.Err(); err != nil {
		return processedCount, lineCount, fmt.Errorf("scanner error: %w", err)
	}

	return processedCount, lineCount, nil
}

// extractProjectNameFromPath extracts project name from file path
func (d *DiffSyncService) extractProjectNameFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	return filepath.Base(dir)
}

// processLogEntry processes a single log entry (similar to existing logic)
func (d *DiffSyncService) processLogEntry(entry *models.LogEntry, projectName string) error {
	// Use cwd from log entry if available, otherwise fall back to project name conversion
	var actualProjectPath, actualProjectName string
	if entry.Cwd != "" {
		actualProjectPath = entry.Cwd
		actualProjectName = d.extractProjectNameFromCwd(entry.Cwd)
	} else {
		actualProjectPath = d.convertProjectNameToPath(projectName)
		actualProjectName = projectName
	}

	if err := d.sessionService.CreateOrUpdateSession(entry.SessionID, actualProjectName, actualProjectPath, entry.Timestamp); err != nil {
		return fmt.Errorf("failed to create/update session: %w", err)
	}

	message := &models.Message{
		ID:          entry.UUID,
		SessionID:   entry.SessionID,
		ParentUUID:  entry.ParentUUID,
		IsSidechain: entry.IsSidechain,
		UserType:    &entry.UserType,
		MessageType: entry.Message.Type,
		MessageRole: &entry.Message.Role,
		Model:       entry.Message.Model,
		Timestamp:   entry.Timestamp,
		RequestID:   entry.RequestID,
	}

	if entry.Message.Content != nil {
		contentStr := d.convertContentToString(entry.Message.Content)
		message.Content = &contentStr
	}

	if entry.Message.Usage != nil {
		message.InputTokens = entry.Message.Usage.InputTokens
		message.CacheCreationInputTokens = entry.Message.Usage.CacheCreationInputTokens
		message.CacheReadInputTokens = entry.Message.Usage.CacheReadInputTokens
		message.OutputTokens = entry.Message.Usage.OutputTokens
		message.ServiceTier = &entry.Message.Usage.ServiceTier
	}

	// Get or create appropriate session window for this message
	window, err := d.windowService.GetOrCreateWindowForMessage(entry.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to get/create session window: %w", err)
	}
	message.SessionWindowID = &window.ID

	if err := d.insertMessage(message); err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Update window statistics after message insertion
	if err := d.windowService.UpdateWindowStats(window.ID); err != nil {
		return fmt.Errorf("failed to update window stats: %w", err)
	}

	if err := d.tokenService.UpdateSessionTokens(entry.SessionID); err != nil {
		return fmt.Errorf("failed to update session tokens: %w", err)
	}

	return nil
}

// Helper methods (copied from existing JSONLParser)
func (d *DiffSyncService) extractProjectNameFromCwd(cwd string) string {
	if cwd == "" {
		return "unknown"
	}

	cleanPath := filepath.Clean(cwd)
	parts := strings.Split(cleanPath, "/")
	if len(parts) == 0 {
		return "unknown"
	}

	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part != "" && part != "." && part != ".." {
			if i > 0 && (part == "frontend" || part == "backend" || part == "src" || part == "lib") {
				parentPart := parts[i-1]
				if parentPart != "" && parentPart != "." && parentPart != ".." {
					return parentPart
				}
			}
			return part
		}
	}

	return "unknown"
}

func (d *DiffSyncService) convertProjectNameToPath(projectName string) string {
	if strings.HasPrefix(projectName, "-") {
		return "/" + strings.ReplaceAll(projectName[1:], "-", "/")
	}
	return projectName
}

func (d *DiffSyncService) convertContentToString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case map[string]interface{}:
		data, _ := json.Marshal(v)
		return string(data)
	case []interface{}:
		data, _ := json.Marshal(v)
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (d *DiffSyncService) insertMessage(message *models.Message) error {
	// Use INSERT OR REPLACE to handle both insert and update atomically
	upsertQuery := `
		INSERT OR REPLACE INTO messages (
			id, session_id, session_window_id, parent_uuid, is_sidechain, user_type, message_type,
			message_role, model, content, input_tokens, cache_creation_input_tokens,
			cache_read_input_tokens, output_tokens, service_tier, request_id,
			timestamp, created_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			COALESCE((SELECT created_at FROM messages WHERE id = ?), ?)
		)
	`
	
	now := time.Now()
	_, err := d.db.Exec(upsertQuery,
		message.ID,
		message.SessionID,
		message.SessionWindowID,
		message.ParentUUID,
		message.IsSidechain,
		message.UserType,
		message.MessageType,
		message.MessageRole,
		message.Model,
		message.Content,
		message.InputTokens,
		message.CacheCreationInputTokens,
		message.CacheReadInputTokens,
		message.OutputTokens,
		message.ServiceTier,
		message.RequestID,
		message.Timestamp,
		message.ID, // for COALESCE subquery
		now,        // created_at for new records
	)
	if err != nil {
		return fmt.Errorf("failed to upsert message: %w", err)
	}

	return nil
}

// GetSyncStats returns current synchronization statistics
func (d *DiffSyncService) GetSyncStats() (*models.SyncStats, error) {
	states, err := d.stateManager.GetAllFileStates()
	if err != nil {
		return nil, fmt.Errorf("failed to get sync stats: %w", err)
	}

	stats := &models.SyncStats{
		TotalFiles: len(states),
	}

	for _, state := range states {
		if state.SyncStatus == "completed" {
			stats.ProcessedFiles++
		}
	}

	stats.SkippedFiles = stats.TotalFiles - stats.ProcessedFiles

	return stats, nil
}