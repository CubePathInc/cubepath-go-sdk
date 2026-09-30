package cubepath

import (
	"context"
	"fmt"
	"net/url"
)

// VPSAvailabilityGroupService handles communication with the VPS availability group related
// methods of the CubePath API. The VPS of a group are spread across different physical nodes.
type VPSAvailabilityGroupService interface {
	ListByProject(ctx context.Context, projectID int) ([]AvailabilityGroup, error)
	Get(ctx context.Context, groupUUID string) (*AvailabilityGroup, error)
	Create(ctx context.Context, req *CreateAvailabilityGroupRequest) (*AvailabilityGroup, error)
	// Delete deletes an empty group; remove its VPS first.
	Delete(ctx context.Context, groupUUID string) error
	// AddVPS adds a VPS of the same project and location to the group.
	AddVPS(ctx context.Context, groupUUID string, vpsID int) error
	RemoveVPS(ctx context.Context, groupUUID string, vpsID int) error
	MoveToProject(ctx context.Context, groupUUID string, projectID int) error
}

// AvailabilityGroupVPS is a VPS member of an availability group.
type AvailabilityGroupVPS struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Label  string `json:"label"`
	Status string `json:"status"`
}

// AvailabilityGroup represents a VPS availability group.
type AvailabilityGroup struct {
	UUID         string                 `json:"uuid"`
	ProjectID    int                    `json:"project_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Strategy     string                 `json:"strategy"`
	LocationName string                 `json:"location_name"`
	MaxServers   int                    `json:"max_servers"`
	VPSCount     int                    `json:"vps_count"`
	VPSList      []AvailabilityGroupVPS `json:"vps_list"`
	CreatedAt    string                 `json:"created_at,omitempty"`
}

// CreateAvailabilityGroupRequest represents a request to create an availability group.
type CreateAvailabilityGroupRequest struct {
	ProjectID    int    `json:"project_id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	LocationName string `json:"location_name"`
}

type vpsAvailabilityGroupService struct {
	client *Client
}

func (s *vpsAvailabilityGroupService) ListByProject(ctx context.Context, projectID int) ([]AvailabilityGroup, error) {
	var result struct {
		Groups []AvailabilityGroup `json:"groups"`
	}
	if err := s.client.get(ctx, fmt.Sprintf("/vps/availability-groups/project/%d", projectID), &result); err != nil {
		return nil, err
	}
	return result.Groups, nil
}

func (s *vpsAvailabilityGroupService) Get(ctx context.Context, groupUUID string) (*AvailabilityGroup, error) {
	var group AvailabilityGroup
	if err := s.client.get(ctx, "/vps/availability-groups/"+url.PathEscape(groupUUID), &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *vpsAvailabilityGroupService) Create(ctx context.Context, req *CreateAvailabilityGroupRequest) (*AvailabilityGroup, error) {
	var group AvailabilityGroup
	if err := s.client.post(ctx, "/vps/availability-groups/", req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *vpsAvailabilityGroupService) Delete(ctx context.Context, groupUUID string) error {
	return s.client.del(ctx, "/vps/availability-groups/"+url.PathEscape(groupUUID))
}

func (s *vpsAvailabilityGroupService) AddVPS(ctx context.Context, groupUUID string, vpsID int) error {
	return s.client.post(ctx, fmt.Sprintf("/vps/availability-groups/%s/vps/%d", url.PathEscape(groupUUID), vpsID), nil, nil)
}

func (s *vpsAvailabilityGroupService) RemoveVPS(ctx context.Context, groupUUID string, vpsID int) error {
	return s.client.del(ctx, fmt.Sprintf("/vps/availability-groups/%s/vps/%d", url.PathEscape(groupUUID), vpsID))
}

func (s *vpsAvailabilityGroupService) MoveToProject(ctx context.Context, groupUUID string, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/availability-groups/%s/move-project", url.PathEscape(groupUUID)), body, nil)
}
