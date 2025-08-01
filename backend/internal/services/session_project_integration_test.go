package services

import (
	"database/sql"
	"testing"

	_ "github.com/marcboeker/go-duckdb"
)

func setupIntegrationTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create all required tables
	queries := []string{
		// Projects table
		`CREATE TABLE projects (
			id VARCHAR PRIMARY KEY,
			name VARCHAR NOT NULL,
			path VARCHAR NOT NULL,
			description TEXT,
			repository_url VARCHAR,
			language VARCHAR,
			framework VARCHAR,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(name, path)
		)`,
		
		// Sessions table with project_id
		`CREATE TABLE sessions (
			id VARCHAR PRIMARY KEY,
			project_name VARCHAR NOT NULL,
			project_path VARCHAR NOT NULL,
			project_id VARCHAR,
			start_time TIMESTAMP NOT NULL,
			end_time TIMESTAMP,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			message_count INTEGER DEFAULT 0,
			total_cost DOUBLE DEFAULT 0.0,
			status VARCHAR DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Messages table
		`CREATE TABLE messages (
			id VARCHAR PRIMARY KEY,
			session_id VARCHAR NOT NULL,
			parent_uuid VARCHAR,
			is_sidechain BOOLEAN DEFAULT false,
			user_type VARCHAR,
			message_type VARCHAR,
			message_role VARCHAR,
			model VARCHAR,
			content TEXT,
			input_tokens INTEGER DEFAULT 0,
			cache_creation_input_tokens INTEGER DEFAULT 0,
			cache_read_input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			service_tier VARCHAR,
			request_id VARCHAR,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	for _, query := range queries {
		_, err = db.Exec(query)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return db
}

// TestBackwardCompatibility tests that existing functionality still works
func TestBackwardCompatibility(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	sessionService := NewSessionService(db)

	// Test 1: Legacy CreateOrUpdateSession should still work
	sessionID := "test-session-1"
	projectName := "test-project"
	projectPath := "/test/path"

	err := sessionService.CreateOrUpdateSession(sessionID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Legacy CreateOrUpdateSession failed: %v", err)
	}

	// Verify session was created without project_id
	var createdProjectID *string
	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", sessionID).Scan(&createdProjectID)
	if err != nil {
		t.Fatalf("Failed to verify session creation: %v", err)
	}

	if createdProjectID != nil {
		t.Error("Expected project_id to be NULL for legacy session creation")
	}

	// Test 2: GetAllSessions should work with mixed data
	sessions, err := sessionService.GetAllSessions()
	if err != nil {
		t.Fatalf("GetAllSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].ProjectID != nil {
		t.Error("Expected project_id to be nil for legacy session")
	}
}

// TestProjectIntegration tests the new Project integration features
func TestProjectIntegration(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	sessionService := NewSessionService(db)

	// Test 1: CreateOrUpdateSessionWithProject should create project and session
	sessionID := "test-session-with-project"
	projectName := "integrated-project"
	projectPath := "/integrated/path"

	err := sessionService.CreateOrUpdateSessionWithProject(sessionID, projectName, projectPath)
	if err != nil {
		t.Fatalf("CreateOrUpdateSessionWithProject failed: %v", err)
	}

	// Verify project was created
	projectService := NewProjectService(db)
	project, err := projectService.FindProjectByNameAndPath(projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to find created project: %v", err)
	}
	if project == nil {
		t.Fatal("Project was not created")
	}

	// Verify session has project_id
	var sessionProjectID *string
	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", sessionID).Scan(&sessionProjectID)
	if err != nil {
		t.Fatalf("Failed to get session project_id: %v", err)
	}

	if sessionProjectID == nil {
		t.Fatal("Session project_id should not be null")
	}

	if *sessionProjectID != project.ID {
		t.Errorf("Expected session project_id %s, got %s", project.ID, *sessionProjectID)
	}

	// Test 2: GetSessionsByProject should return the session
	sessions, err := sessionService.GetSessionsByProject(project.ID)
	if err != nil {
		t.Fatalf("GetSessionsByProject failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for project, got %d", len(sessions))
	}

	if sessions[0].ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, sessions[0].ID)
	}
}

// TestMigrationFromLegacyToProject tests migrating existing sessions
func TestMigrationFromLegacyToProject(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	sessionService := NewSessionService(db)

	// Create legacy session
	sessionID := "legacy-session"
	projectName := "legacy-project"
	projectPath := "/legacy/path"

	err := sessionService.CreateOrUpdateSession(sessionID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to create legacy session: %v", err)
	}

	// Verify it has no project_id
	sessions, err := sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		t.Fatalf("Failed to get sessions without project_id: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session without project_id, got %d", len(sessions))
	}

	// Migrate the session
	err = sessionService.MigrateSessionToProject(sessionID)
	if err != nil {
		t.Fatalf("Failed to migrate session: %v", err)
	}

	// Verify project was created and session was updated
	projectService := NewProjectService(db)
	project, err := projectService.FindProjectByNameAndPath(projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to find project after migration: %v", err)
	}
	if project == nil {
		t.Fatal("Project should exist after migration")
	}

	// Verify session now has project_id
	var sessionProjectID *string
	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", sessionID).Scan(&sessionProjectID)
	if err != nil {
		t.Fatalf("Failed to get session project_id after migration: %v", err)
	}

	if sessionProjectID == nil || *sessionProjectID != project.ID {
		t.Errorf("Session should have project_id %s after migration, got %v", project.ID, sessionProjectID)
	}

	// Verify no more sessions without project_id
	remainingSessions, err := sessionService.GetSessionsWithoutProjectID()
	if err != nil {
		t.Fatalf("Failed to get remaining sessions: %v", err)
	}

	if len(remainingSessions) != 0 {
		t.Errorf("Expected 0 sessions without project_id after migration, got %d", len(remainingSessions))
	}
}

// TestDuplicateProjectHandling tests that duplicate projects are handled correctly
func TestDuplicateProjectHandling(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	sessionService := NewSessionService(db)

	projectName := "shared-project"
	projectPath := "/shared/path"

	// Create first session with project
	session1ID := "session-1"
	err := sessionService.CreateOrUpdateSessionWithProject(session1ID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to create first session: %v", err)
	}

	// Create second session with same project
	session2ID := "session-2"
	err = sessionService.CreateOrUpdateSessionWithProject(session2ID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}

	// Verify only one project was created
	projectService := NewProjectService(db)
	projects, err := projectService.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get all projects: %v", err)
	}

	if len(projects) != 1 {
		t.Errorf("Expected 1 project, got %d", len(projects))
	}

	// Verify both sessions have the same project_id
	var project1ID, project2ID *string
	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", session1ID).Scan(&project1ID)
	if err != nil {
		t.Fatalf("Failed to get session 1 project_id: %v", err)
	}

	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", session2ID).Scan(&project2ID)
	if err != nil {
		t.Fatalf("Failed to get session 2 project_id: %v", err)
	}

	if project1ID == nil || project2ID == nil {
		t.Fatal("Both sessions should have project_id")
	}

	if *project1ID != *project2ID {
		t.Errorf("Both sessions should have the same project_id, got %s and %s", *project1ID, *project2ID)
	}

	// Verify GetSessionsByProject returns both sessions
	sessions, err := sessionService.GetSessionsByProject(*project1ID)
	if err != nil {
		t.Fatalf("GetSessionsByProject failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions for project, got %d", len(sessions))
	}
}

// TestExistingSessionUpdateWithProject tests updating existing sessions
func TestExistingSessionUpdateWithProject(t *testing.T) {
	db := setupIntegrationTestDB(t)
	defer db.Close()

	sessionService := NewSessionService(db)

	sessionID := "existing-session"
	projectName := "update-project"
	projectPath := "/update/path"

	// Create legacy session first
	err := sessionService.CreateOrUpdateSession(sessionID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to create legacy session: %v", err)
	}

	// Now update with project integration
	err = sessionService.CreateOrUpdateSessionWithProject(sessionID, projectName, projectPath)
	if err != nil {
		t.Fatalf("Failed to update session with project: %v", err)
	}

	// Verify session now has project_id
	var sessionProjectID *string
	err = db.QueryRow("SELECT project_id FROM sessions WHERE id = ?", sessionID).Scan(&sessionProjectID)
	if err != nil {
		t.Fatalf("Failed to get session project_id: %v", err)
	}

	if sessionProjectID == nil {
		t.Error("Session should have project_id after update")
	}

	// Verify only one session exists (no duplicates)
	var sessionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", sessionID).Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}

	if sessionCount != 1 {
		t.Errorf("Expected 1 session, got %d", sessionCount)
	}
}