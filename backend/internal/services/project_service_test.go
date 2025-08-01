package services

import (
	"database/sql"
	"testing"
	"time"

	"ccdash-backend/internal/models"
	_ "github.com/marcboeker/go-duckdb"
)

func setupProjectTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create projects table
	createTableQuery := `
		CREATE TABLE projects (
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
		)
	`
	
	_, err = db.Exec(createTableQuery)
	if err != nil {
		t.Fatalf("Failed to create projects table: %v", err)
	}

	// Create sessions table for migration test
	createSessionsQuery := `
		CREATE TABLE sessions (
			id VARCHAR PRIMARY KEY,
			project_name VARCHAR NOT NULL,
			project_path VARCHAR NOT NULL,
			start_time TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	
	_, err = db.Exec(createSessionsQuery)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return db
}

func TestCreateProject(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	// Test creating a new project
	name := "test-project"
	path := "/test/path"

	project, err := projectService.CreateProject(name, path)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	if project.Name != name {
		t.Errorf("Expected project name %s, got %s", name, project.Name)
	}

	if project.Path != path {
		t.Errorf("Expected project path %s, got %s", path, project.Path)
	}

	if !project.IsActive {
		t.Error("Expected project to be active")
	}

	// Verify project was inserted in database
	var dbName, dbPath string
	var isActive bool
	err = db.QueryRow("SELECT name, path, is_active FROM projects WHERE id = ?", project.ID).
		Scan(&dbName, &dbPath, &isActive)
	if err != nil {
		t.Fatalf("Failed to verify project in database: %v", err)
	}

	if dbName != name || dbPath != path || !isActive {
		t.Errorf("Project data mismatch in database")
	}
}

func TestGetOrCreateProject(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	name := "test-project"
	path := "/test/path"

	// First call should create the project
	project1, err := projectService.GetOrCreateProject(name, path)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Second call should return the existing project
	project2, err := projectService.GetOrCreateProject(name, path)
	if err != nil {
		t.Fatalf("Failed to get existing project: %v", err)
	}

	if project1.ID != project2.ID {
		t.Errorf("Expected same project ID, got %s and %s", project1.ID, project2.ID)
	}

	// Verify only one project exists in database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM projects WHERE name = ? AND path = ?", name, path).
		Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count projects: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 project, found %d", count)
	}
}

func TestFindProjectByNameAndPath(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	name := "test-project"
	path := "/test/path"

	// Project should not exist initially
	project, err := projectService.FindProjectByNameAndPath(name, path)
	if err != nil {
		t.Fatalf("Failed to find project: %v", err)
	}
	if project != nil {
		t.Error("Expected nil project, but found one")
	}

	// Create project
	createdProject, err := projectService.CreateProject(name, path)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Now project should be found
	foundProject, err := projectService.FindProjectByNameAndPath(name, path)
	if err != nil {
		t.Fatalf("Failed to find project: %v", err)
	}
	if foundProject == nil {
		t.Fatal("Expected to find project, but got nil")
	}

	if foundProject.ID != createdProject.ID {
		t.Errorf("Expected project ID %s, got %s", createdProject.ID, foundProject.ID)
	}
}

func TestGetAllProjects(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	// Initially should have no projects
	projects, err := projectService.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get all projects: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}

	// Create test projects
	testProjects := []struct {
		name string
		path string
	}{
		{"project-a", "/path/a"},
		{"project-b", "/path/b"},
		{"project-c", "/path/c"},
	}

	for _, tp := range testProjects {
		_, err := projectService.CreateProject(tp.name, tp.path)
		if err != nil {
			t.Fatalf("Failed to create test project %s: %v", tp.name, err)
		}
	}

	// Get all projects
	projects, err = projectService.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get all projects: %v", err)
	}

	if len(projects) != len(testProjects) {
		t.Errorf("Expected %d projects, got %d", len(testProjects), len(projects))
	}

	// Verify projects are sorted by name
	for i := 1; i < len(projects); i++ {
		if projects[i-1].Name > projects[i].Name {
			t.Error("Projects are not sorted by name")
			break
		}
	}
}

func TestUpdateProject(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	// Create initial project
	project, err := projectService.CreateProject("test-project", "/test/path")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	originalUpdatedAt := project.UpdatedAt

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	// Update project (only update fields that don't affect uniqueness)
	description := "Updated description"
	language := "Go"
	project.Description = &description
	project.Language = &language

	// Debug: Check existing records before update
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count projects: %v", err)
	}
	t.Logf("Projects in DB before update: %d", count)
	
	// Debug: Show current project data
	t.Logf("Updating project ID: %s, Name: %s, Path: %s", project.ID, project.Name, project.Path)
	
	err = projectService.UpdateProject(project)
	if err != nil {
		t.Fatalf("Failed to update project: %v", err)
	}

	// Verify update in database
	updatedProject, err := projectService.GetProjectByID(project.ID)
	if err != nil {
		t.Fatalf("Failed to get updated project: %v", err)
	}

	if updatedProject.Description == nil || *updatedProject.Description != description {
		t.Errorf("Expected description %s, got %v", description, updatedProject.Description)
	}

	if updatedProject.Language == nil || *updatedProject.Language != language {
		t.Errorf("Expected language %s, got %v", language, updatedProject.Language)
	}

	if !updatedProject.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected updated_at to be updated")
	}
}

func TestDeleteProject(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	// Create project
	project, err := projectService.CreateProject("test-project", "/test/path")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Delete project
	err = projectService.DeleteProject(project.ID)
	if err != nil {
		t.Fatalf("Failed to delete project: %v", err)
	}

	// Verify project is soft deleted (is_active = false)
	var isActive bool
	err = db.QueryRow("SELECT is_active FROM projects WHERE id = ?", project.ID).
		Scan(&isActive)
	if err != nil {
		t.Fatalf("Failed to check project status: %v", err)
	}

	if isActive {
		t.Error("Expected project to be inactive after deletion")
	}

	// Verify GetAllProjects doesn't return deleted project
	projects, err := projectService.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get all projects: %v", err)
	}

	for _, p := range projects {
		if p.ID == project.ID {
			t.Error("Deleted project should not appear in GetAllProjects")
		}
	}
}

func TestMigrateExistingSessionsToProjects(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	projectService := NewProjectService(db)

	// Insert test sessions
	testSessions := []struct {
		id          string
		projectName string
		projectPath string
		createdAt   time.Time
	}{
		{"session1", "project-a", "/path/a", time.Now().Add(-2 * time.Hour)},
		{"session2", "project-a", "/path/a", time.Now().Add(-1 * time.Hour)}, // Duplicate project
		{"session3", "project-b", "/path/b", time.Now().Add(-30 * time.Minute)},
	}

	for _, session := range testSessions {
		_, err := db.Exec(`
			INSERT INTO sessions (id, project_name, project_path, start_time, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, session.id, session.projectName, session.projectPath, session.createdAt, session.createdAt)
		if err != nil {
			t.Fatalf("Failed to insert test session: %v", err)
		}
	}

	// Run migration
	err := projectService.MigrateExistingSessionsToProjects()
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify projects were created
	projects, err := projectService.GetAllProjects()
	if err != nil {
		t.Fatalf("Failed to get projects after migration: %v", err)
	}

	// Should have 2 unique projects (project-a and project-b)
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects after migration, got %d", len(projects))
	}

	// Verify project details
	projectMap := make(map[string]*models.Project)
	for i := range projects {
		projectMap[projects[i].Name] = &projects[i]
	}

	if projectA, exists := projectMap["project-a"]; exists {
		if projectA.Path != "/path/a" {
			t.Errorf("Expected project-a path /path/a, got %s", projectA.Path)
		}
		// Should have the earliest creation time
		expectedTime := testSessions[0].createdAt.Truncate(time.Second)
		actualTime := projectA.CreatedAt.Truncate(time.Second)
		if !actualTime.Equal(expectedTime) {
			t.Errorf("Expected project-a created_at %v, got %v", expectedTime, actualTime)
		}
	} else {
		t.Error("project-a not found after migration")
	}

	if projectB, exists := projectMap["project-b"]; exists {
		if projectB.Path != "/path/b" {
			t.Errorf("Expected project-b path /path/b, got %s", projectB.Path)
		}
	} else {
		t.Error("project-b not found after migration")
	}
}