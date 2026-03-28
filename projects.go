package cubepath

import (
	"context"
	"fmt"
	"time"
)

// ProjectService handles communication with the project related methods of the CubePath API.
type ProjectService interface {
	Create(ctx context.Context, req *CreateProjectRequest) (*Project, error)
	List(ctx context.Context) ([]ProjectResponse, error)
	Get(ctx context.Context, projectID int) (*ProjectResponse, error)
	Delete(ctx context.Context, projectID int) error
}

// Project represents a CubePath project.
type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProjectResponse represents a project with its associated resources.
type ProjectResponse struct {
	Project    Project     `json:"project"`
	VPS        []VPS       `json:"vps"`
	Networks   []Network   `json:"networks"`
	Baremetals []Baremetal `json:"baremetals"`
}

// CreateProjectRequest represents a request to create a project.
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type projectService struct {
	client *Client
}

func (s *projectService) Create(ctx context.Context, req *CreateProjectRequest) (*Project, error) {
	var project Project
	if err := s.client.post(ctx, "/projects/", req, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *projectService) List(ctx context.Context) ([]ProjectResponse, error) {
	var projects []ProjectResponse
	if err := s.client.get(ctx, "/projects/", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *projectService) Get(ctx context.Context, projectID int) (*ProjectResponse, error) {
	projects, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].Project.ID == projectID {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %d not found", projectID)
}

func (s *projectService) Delete(ctx context.Context, projectID int) error {
	return s.client.del(ctx, fmt.Sprintf("/projects/%d", projectID))
}
