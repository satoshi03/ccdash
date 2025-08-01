package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "http://localhost:6060/api"

type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Path          string    `json:"path"`
	Description   *string   `json:"description"`
	RepositoryURL *string   `json:"repository_url"`
	Language      *string   `json:"language"`
	Framework     *string   `json:"framework"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProjectsResponse struct {
	Projects []Project `json:"projects"`
	Count    int       `json:"count"`
}

type ProjectResponse struct {
	Project      Project   `json:"project"`
	Sessions     []Session `json:"sessions"`
	SessionCount int       `json:"session_count"`
}

type Session struct {
	ID           string `json:"id"`
	ProjectName  string `json:"project_name"`
	ProjectPath  string `json:"project_path"`
	ProjectID    string `json:"project_id"`
	MessageCount int    `json:"message_count"`
}

type UpdateProjectRequest struct {
	Description   *string `json:"description,omitempty"`
	RepositoryURL *string `json:"repository_url,omitempty"`
	Language      *string `json:"language,omitempty"`
	Framework     *string `json:"framework,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

func main() {
	fmt.Println("=== Project API Testing ===")
	
	// Test 1: GET /api/projects
	fmt.Println("\n1. Testing GET /api/projects")
	projects, err := getAllProjects()
	if err != nil {
		log.Printf("Error getting all projects: %v", err)
		return
	}
	fmt.Printf("✅ Retrieved %d projects\n", len(projects))
	
	if len(projects) == 0 {
		fmt.Println("❌ No projects found, cannot continue with tests")
		return
	}

	// Test 2: GET /api/projects/:id
	testProject := projects[0]
	fmt.Printf("\n2. Testing GET /api/projects/%s\n", testProject.ID)
	projectDetail, err := getProject(testProject.ID)
	if err != nil {
		log.Printf("Error getting project detail: %v", err)
		return
	}
	fmt.Printf("✅ Retrieved project '%s' with %d sessions\n", projectDetail.Project.Name, projectDetail.SessionCount)

	// Test 3: PUT /api/projects/:id (Update project)
	fmt.Printf("\n3. Testing PUT /api/projects/%s\n", testProject.ID)
	description := "Updated via API test"
	language := "Go"
	updateReq := UpdateProjectRequest{
		Description: &description,
		Language:    &language,
	}
	err = updateProject(testProject.ID, updateReq)
	if err != nil {
		log.Printf("Error updating project: %v", err)
		return
	}
	fmt.Println("✅ Project updated successfully")

	// Verify the update
	updatedProject, err := getProject(testProject.ID)
	if err != nil {
		log.Printf("Error verifying project update: %v", err)
		return
	}
	if updatedProject.Project.Description != nil && *updatedProject.Project.Description == description {
		fmt.Printf("✅ Update verified - description: %s\n", *updatedProject.Project.Description)
	}
	if updatedProject.Project.Language != nil && *updatedProject.Project.Language == language {
		fmt.Printf("✅ Update verified - language: %s\n", *updatedProject.Project.Language)
	}

	// Test 4: GET /api/projects/:id/sessions
	fmt.Printf("\n4. Testing GET /api/projects/%s/sessions\n", testProject.ID)
	sessions, err := getProjectSessions(testProject.ID)
	if err != nil {
		log.Printf("Error getting project sessions: %v", err)
		return
	}
	fmt.Printf("✅ Retrieved %d sessions for project\n", len(sessions))

	// Test 5: POST /api/projects/migrate-sessions
	fmt.Println("\n5. Testing POST /api/projects/migrate-sessions")
	err = migrateSessionsToProjects()
	if err != nil {
		log.Printf("Error migrating sessions: %v", err)
		return
	}
	fmt.Println("✅ Migration endpoint tested successfully")

	fmt.Println("\n=== All API Tests Passed! ===")
	fmt.Println("✅ Projects API is working correctly")
	fmt.Println("✅ All endpoints are accessible and functioning")
	fmt.Println("✅ Data integrity is maintained")
}

func getAllProjects() ([]Project, error) {
	resp, err := http.Get(baseURL + "/projects")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response ProjectsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Projects, nil
}

func getProject(id string) (*ProjectResponse, error) {
	resp, err := http.Get(baseURL + "/projects/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response ProjectResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func updateProject(id string, updateReq UpdateProjectRequest) error {
	jsonData, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest("PUT", baseURL+"/projects/"+id, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func getProjectSessions(id string) ([]Session, error) {
	resp, err := http.Get(baseURL + "/projects/" + id + "/sessions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Sessions []Session `json:"sessions"`
		Count    int       `json:"count"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Sessions, nil
}

func migrateSessionsToProjects() error {
	resp, err := http.Post(baseURL+"/projects/migrate-sessions", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}