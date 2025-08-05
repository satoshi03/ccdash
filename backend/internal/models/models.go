package models

import (
	"time"
)

type Session struct {
	ID               string    `json:"id" db:"id"`
	ProjectName      string    `json:"project_name" db:"project_name"`
	ProjectPath      string    `json:"project_path" db:"project_path"`
	ProjectID        *string   `json:"project_id" db:"project_id"` // Phase 2: nullable for backward compatibility
	StartTime        time.Time `json:"start_time" db:"start_time"`
	EndTime          *time.Time `json:"end_time" db:"end_time"`
	TotalInputTokens int       `json:"total_input_tokens" db:"total_input_tokens"`
	TotalOutputTokens int      `json:"total_output_tokens" db:"total_output_tokens"`
	TotalTokens      int       `json:"total_tokens" db:"total_tokens"`
	MessageCount     int       `json:"message_count" db:"message_count"`
	Status           string    `json:"status" db:"status"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	TotalCost        float64   `json:"total_cost" db:"total_cost"`
}

type Message struct {
	ID                        string    `json:"id" db:"id"`
	SessionID                 string    `json:"session_id" db:"session_id"`
	ParentUUID                *string   `json:"parent_uuid" db:"parent_uuid"`
	IsSidechain              bool      `json:"is_sidechain" db:"is_sidechain"`
	UserType                 *string   `json:"user_type" db:"user_type"`
	MessageType              *string   `json:"message_type" db:"message_type"`
	MessageRole              *string   `json:"message_role" db:"message_role"`
	Model                    *string   `json:"model" db:"model"`
	Content                  *string   `json:"content" db:"content"`
	InputTokens              int       `json:"input_tokens" db:"input_tokens"`
	CacheCreationInputTokens int       `json:"cache_creation_input_tokens" db:"cache_creation_input_tokens"`
	CacheReadInputTokens     int       `json:"cache_read_input_tokens" db:"cache_read_input_tokens"`
	OutputTokens             int       `json:"output_tokens" db:"output_tokens"`
	ServiceTier              *string   `json:"service_tier" db:"service_tier"`
	RequestID                *string   `json:"request_id" db:"request_id"`
	Timestamp                time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
}

type SessionWindowMessage struct {
	ID               string    `json:"id" db:"id"`
	SessionWindowID  string    `json:"session_window_id" db:"session_window_id"`
	MessageID        string    `json:"message_id" db:"message_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type TokenUsage struct {
	TotalTokens      int     `json:"total_tokens"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	UsageLimit       int     `json:"usage_limit"`
	UsageRate        float64 `json:"usage_rate"`
	WindowStart      time.Time `json:"window_start"`
	WindowEnd        time.Time `json:"window_end"`
	ActiveSessions   int     `json:"active_sessions"`
	TotalCost        float64 `json:"total_cost"`
	TotalMessages    int     `json:"total_messages"`
}

type SessionSummary struct {
	Session
	Duration        *time.Duration `json:"duration"`
	IsActive        bool          `json:"is_active"`
	LastActivity    time.Time     `json:"last_activity"`
	GeneratedCode   []string      `json:"generated_code"`
}

type LogEntry struct {
	ParentUUID   *string                `json:"parentUuid"`
	IsSidechain  bool                  `json:"isSidechain"`
	UserType     string                `json:"userType"`
	Cwd          string                `json:"cwd"`
	SessionID    string                `json:"sessionId"`
	Version      string                `json:"version"`
	Type         string                `json:"type"`
	Message      LogMessage            `json:"message"`
	RequestID    *string               `json:"requestId"`
	UUID         string                `json:"uuid"`
	Timestamp    time.Time             `json:"timestamp"`
}

type LogMessage struct {
	ID      *string    `json:"id"`
	Type    *string    `json:"type"`
	Role    string     `json:"role"`
	Model   *string    `json:"model"`
	Content interface{} `json:"content"`
	Usage   *Usage     `json:"usage"`
}

type Usage struct {
	InputTokens              int    `json:"input_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	ServiceTier              string `json:"service_tier"`
}

type BurnRatePoint struct {
	Timestamp     time.Time `json:"timestamp"`
	TokensPerHour int       `json:"tokens_per_hour"`
}

// Project represents a project entity
type Project struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Path          string    `json:"path" db:"path"`
	Description   *string   `json:"description" db:"description"`
	RepositoryURL *string   `json:"repository_url" db:"repository_url"`
	Language      *string   `json:"language" db:"language"`
	Framework     *string   `json:"framework" db:"framework"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// Job represents a task execution job
type Job struct {
	ID                  string     `json:"id" db:"id"`
	ProjectID           string     `json:"project_id" db:"project_id"`
	Command             string     `json:"command" db:"command"`
	ExecutionDirectory  string     `json:"execution_directory" db:"execution_directory"`
	YoloMode           bool       `json:"yolo_mode" db:"yolo_mode"`
	Status             string     `json:"status" db:"status"`
	Priority           int        `json:"priority" db:"priority"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	StartedAt          *time.Time `json:"started_at" db:"started_at"`
	CompletedAt        *time.Time `json:"completed_at" db:"completed_at"`
	OutputLog          *string    `json:"output_log" db:"output_log"`
	ErrorLog           *string    `json:"error_log" db:"error_log"`
	ExitCode           *int       `json:"exit_code" db:"exit_code"`
	PID                *int       `json:"pid" db:"pid"`
	ScheduledAt        *time.Time `json:"scheduled_at" db:"scheduled_at"`
	ScheduleType       *string    `json:"schedule_type" db:"schedule_type"`
	ScheduleParams     *string    `json:"schedule_params" db:"schedule_params"`
	
	// リレーション情報（JOIN時に使用）
	Project            *Project   `json:"project,omitempty"`
}

// JobStatus constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running" 
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
	JobStatusCancelled = "cancelled"
)

// ScheduleType constants
const (
	ScheduleTypeImmediate  = "immediate"
	ScheduleTypeAfterReset = "after_reset"
	ScheduleTypeDelayed    = "delayed"    // N時間後実行
	ScheduleTypeScheduled  = "scheduled"  // 時刻指定（customを廃止）
)

// ScheduleParams stores additional scheduling parameters
type ScheduleParams struct {
	DelayHours    *int       `json:"delay_hours,omitempty"`    // For delayed execution
	ScheduledTime *time.Time `json:"scheduled_time,omitempty"` // For scheduled execution
}

// JobFilters for queries
type JobFilters struct {
	ProjectID *string
	Status    *string
	Limit     int
	Offset    int
}

// CreateJobRequest represents job creation request
type CreateJobRequest struct {
	ProjectID      string          `json:"project_id" binding:"required"`
	Command        string          `json:"command" binding:"required"`
	YoloMode       bool            `json:"yolo_mode"`
	ScheduleType   string          `json:"schedule_type"`
	ScheduleParams *ScheduleParams `json:"schedule_params,omitempty"`
}

// Authentication models

// User represents a user in the system
type User struct {
	ID                   string     `json:"id" db:"id"`
	Email                string     `json:"email" db:"email"`
	PasswordHash         string     `json:"-" db:"password_hash"` // Never expose password hash in JSON
	Roles                []string   `json:"roles" db:"roles"`     // Will be serialized as JSON array
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin            *time.Time `json:"last_login" db:"last_login"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	FailedLoginAttempts  int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil          *time.Time `json:"locked_until" db:"locked_until"`
}

// UserRegistrationRequest represents user registration request
type UserRegistrationRequest struct {
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles,omitempty"`
}

// UserLoginRequest represents user login request
type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken represents a refresh token in the database
type RefreshToken struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never expose token hash
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at" db:"revoked_at"`
	IsRevoked bool       `json:"is_revoked" db:"is_revoked"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string     `json:"id" db:"id"`
	UserID    *string    `json:"user_id" db:"user_id"`
	UserEmail *string    `json:"user_email" db:"user_email"`
	Action    string     `json:"action" db:"action"`
	Resource  string     `json:"resource" db:"resource"`
	Details   *string    `json:"details" db:"details"` // JSON string
	IPAddress *string    `json:"ip_address" db:"ip_address"`
	UserAgent *string    `json:"user_agent" db:"user_agent"`
	Success   bool       `json:"success" db:"success"`
	Timestamp time.Time  `json:"timestamp" db:"timestamp"`
}

// Permission represents a permission in the system
type Permission string

// Permission constants
const (
	PermissionViewDashboard   Permission = "dashboard:view"
	PermissionSyncLogs        Permission = "logs:sync"
	PermissionExecuteTasks    Permission = "tasks:execute"
	PermissionManageSystem    Permission = "system:manage"
	PermissionManageUsers     Permission = "users:manage"
	PermissionViewAuditLogs   Permission = "audit:view"
)

// RolePermissions maps roles to their permissions
type RolePermissions map[string][]Permission

// DefaultRoles defines the default role-permission mappings
var DefaultRoles = RolePermissions{
	"viewer": {PermissionViewDashboard},
	"user":   {PermissionViewDashboard, PermissionSyncLogs},
	"admin":  {PermissionViewDashboard, PermissionSyncLogs, PermissionExecuteTasks, PermissionManageSystem, PermissionManageUsers, PermissionViewAuditLogs},
}