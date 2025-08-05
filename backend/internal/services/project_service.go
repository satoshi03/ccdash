package services

import (
	"database/sql"
	"fmt"
	"time"

	"ccdash-backend/internal/models"
	"github.com/google/uuid"
)

type ProjectService struct {
	db *sql.DB
}

func NewProjectService(db *sql.DB) *ProjectService {
	return &ProjectService{db: db}
}

// GetOrCreateProject gets an existing project or creates a new one
func (p *ProjectService) GetOrCreateProject(name, path string) (*models.Project, error) {
	// Try to find existing project first
	project, err := p.FindProjectByNameAndPath(name, path)
	if err != nil {
		return nil, fmt.Errorf("failed to find project: %w", err)
	}
	
	if project != nil {
		return project, nil
	}
	
	// Create new project if not found
	return p.CreateProject(name, path)
}

// FindProjectByNameAndPath finds a project by name and path
func (p *ProjectService) FindProjectByNameAndPath(name, path string) (*models.Project, error) {
	query := `
		SELECT id, name, path, description, repository_url, language, framework,
			   is_active, created_at, updated_at
		FROM projects
		WHERE name = ? AND path = ?
	`
	
	var project models.Project
	err := p.db.QueryRow(query, name, path).Scan(
		&project.ID,
		&project.Name,
		&project.Path,
		&project.Description,
		&project.RepositoryURL,
		&project.Language,
		&project.Framework,
		&project.IsActive,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Project not found
		}
		return nil, fmt.Errorf("failed to query project: %w", err)
	}
	
	return &project, nil
}

// CreateProject creates a new project
func (p *ProjectService) CreateProject(name, path string) (*models.Project, error) {
	// Generate UUID for project ID
	id := uuid.New().String()
	now := time.Now()
	
	project := &models.Project{
		ID:        id,
		Name:      name,
		Path:      path,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	query := `
		INSERT INTO projects (id, name, path, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err := p.db.Exec(query,
		project.ID,
		project.Name,
		project.Path,
		project.IsActive,
		project.CreatedAt,
		project.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	
	return project, nil
}

// GetProjectByID gets a project by ID
func (p *ProjectService) GetProjectByID(id string) (*models.Project, error) {
	query := `
		SELECT id, name, path, description, repository_url, language, framework,
			   is_active, created_at, updated_at
		FROM projects
		WHERE id = ?
	`
	
	var project models.Project
	err := p.db.QueryRow(query, id).Scan(
		&project.ID,
		&project.Name,
		&project.Path,
		&project.Description,
		&project.RepositoryURL,
		&project.Language,
		&project.Framework,
		&project.IsActive,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query project by ID: %w", err)
	}
	
	return &project, nil
}

// GetAllProjects gets all projects that have sessions
func (p *ProjectService) GetAllProjects() ([]models.Project, error) {
	// Only return projects that have sessions associated with them
	query := `
		SELECT DISTINCT p.id, p.name, p.path, p.description, p.repository_url, 
		       p.language, p.framework, p.is_active, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN sessions s ON p.id = s.project_id
		WHERE p.is_active = true
		ORDER BY p.name ASC
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all projects: %w", err)
	}
	defer rows.Close()
	
	var projects []models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Path,
			&project.Description,
			&project.RepositoryURL,
			&project.Language,
			&project.Framework,
			&project.IsActive,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}
	
	return projects, nil
}

// UpdateProject updates an existing project
func (p *ProjectService) UpdateProject(project *models.Project) error {
	project.UpdatedAt = time.Now()
	
	// Use simple UPDATE query for DuckDB compatibility
	query := `
		UPDATE projects
		SET description = ?, repository_url = ?, language = ?, framework = ?, updated_at = ?
		WHERE id = ?
	`
	
	_, err := p.db.Exec(query,
		project.Description,
		project.RepositoryURL,
		project.Language,
		project.Framework,
		project.UpdatedAt,
		project.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}
	
	return nil
}

// DeleteProject soft deletes a project (sets is_active to false)
func (p *ProjectService) DeleteProject(id string) error {
	query := `
		UPDATE projects
		SET is_active = false, updated_at = ?
		WHERE id = ?
	`
	
	_, err := p.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	
	return nil
}

// generateProjectUUID generates a new UUID for project ID
func (p *ProjectService) generateProjectUUID() string {
	return uuid.New().String()
}

// MigrateExistingSessionsToProjects migrates existing sessions to create projects
func (p *ProjectService) MigrateExistingSessionsToProjects() error {
	// Get all unique project_name and project_path combinations from sessions
	query := `
		SELECT DISTINCT project_name, project_path, MIN(created_at) as first_created
		FROM sessions
		WHERE project_name IS NOT NULL AND project_path IS NOT NULL
		GROUP BY project_name, project_path
		ORDER BY first_created ASC
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query existing sessions: %w", err)
	}
	defer rows.Close()
	
	projectCount := 0
	for rows.Next() {
		var name, path string
		var firstCreated time.Time
		
		err := rows.Scan(&name, &path, &firstCreated)
		if err != nil {
			return fmt.Errorf("failed to scan session data: %w", err)
		}
		
		// Check if project already exists
		existing, err := p.FindProjectByNameAndPath(name, path)
		if err != nil {
			return fmt.Errorf("failed to check existing project: %w", err)
		}
		
		if existing == nil {
			// Create project with original creation time using UUID
			id := uuid.New().String()
			createQuery := `
				INSERT INTO projects (id, name, path, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?)
			`
			
			_, err = p.db.Exec(createQuery, id, name, path, true, firstCreated, firstCreated)
			if err != nil {
				return fmt.Errorf("failed to create project during migration: %w", err)
			}
			
			projectCount++
		}
	}
	
	fmt.Printf("Migration completed: created %d projects\n", projectCount)
	return nil
}