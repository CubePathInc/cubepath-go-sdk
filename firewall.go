package cubepath

import (
	"context"
	"fmt"
)

// FirewallService handles communication with the firewall related methods of the CubePath API.
type FirewallService interface {
	Create(ctx context.Context, req *CreateFirewallGroupRequest) (*FirewallGroup, error)
	List(ctx context.Context) ([]FirewallGroup, error)
	Get(ctx context.Context, groupID int) (*FirewallGroup, error)
	Update(ctx context.Context, groupID int, req *UpdateFirewallGroupRequest) (*FirewallGroup, error)
	Delete(ctx context.Context, groupID int) error
	AssignToVPS(ctx context.Context, vpsID int, req *VPSFirewallGroupsRequest) (*VPSFirewallGroupsResponse, error)
}

// FirewallGroup represents a firewall group.
type FirewallGroup struct {
	ID        int            `json:"id"`
	ProjectID int            `json:"project_id"`
	Name      string         `json:"name"`
	Rules     []FirewallRule `json:"rules"`
	Enabled   bool           `json:"enabled"`
	VPSCount  int            `json:"vps_count,omitempty"`
}

// FirewallRule represents a single firewall rule.
type FirewallRule struct {
	Direction string  `json:"direction"`
	Protocol  string  `json:"protocol"`
	Port      *string `json:"port,omitempty"`
	Source    *string `json:"source,omitempty"`
	Comment   *string `json:"comment,omitempty"`
}

// CreateFirewallGroupRequest represents a request to create a firewall group.
type CreateFirewallGroupRequest struct {
	// ProjectID is the project the group belongs to (required; sent as a query parameter).
	ProjectID int            `json:"-"`
	Name      string         `json:"name"`
	Rules     []FirewallRule `json:"rules"`
	Enabled   bool           `json:"enabled"`
}

// UpdateFirewallGroupRequest represents a request to update a firewall group.
type UpdateFirewallGroupRequest struct {
	Name    *string         `json:"name,omitempty"`
	Rules   *[]FirewallRule `json:"rules,omitempty"`
	Enabled *bool           `json:"enabled,omitempty"`
}

// VPSFirewallGroupsRequest represents a request to update VPS firewall groups.
type VPSFirewallGroupsRequest struct {
	FirewallGroupIDs []int `json:"firewall_group_ids"`
}

// VPSFirewallGroupsResponse represents the response from updating VPS firewall groups.
type VPSFirewallGroupsResponse struct {
	Detail string `json:"detail"`
	// Deprecated: the API returns Detail; Message is always empty.
	Message         string `json:"message"`
	VPSID           int    `json:"vps_id"`
	FirewallGroups  []int  `json:"firewall_groups"`
	SyncTaskCreated bool   `json:"sync_task_created"`
}

type firewallService struct {
	client *Client
}

func (s *firewallService) Create(ctx context.Context, req *CreateFirewallGroupRequest) (*FirewallGroup, error) {
	var group FirewallGroup
	if req == nil || req.ProjectID == 0 {
		return nil, fmt.Errorf("CreateFirewallGroupRequest.ProjectID is required")
	}
	if err := s.client.post(ctx, fmt.Sprintf("/firewall/groups?project_id=%d", req.ProjectID), req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *firewallService) List(ctx context.Context) ([]FirewallGroup, error) {
	var groups []FirewallGroup
	if err := s.client.get(ctx, "/firewall/groups", &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// Get returns one firewall group. The API has no single-group endpoint, so it is looked up
// in the list.
func (s *firewallService) Get(ctx context.Context, groupID int) (*FirewallGroup, error) {
	groups, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].ID == groupID {
			return &groups[i], nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "Not Found", Detail: fmt.Sprintf("firewall group %d not found", groupID)}
}

// Update changes the name, rules or enabled flag of a group; nil fields are left unchanged.
func (s *firewallService) Update(ctx context.Context, groupID int, req *UpdateFirewallGroupRequest) (*FirewallGroup, error) {
	var group FirewallGroup
	if err := s.client.put(ctx, fmt.Sprintf("/firewall/groups/%d", groupID), req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *firewallService) Delete(ctx context.Context, groupID int) error {
	return s.client.del(ctx, fmt.Sprintf("/firewall/groups/%d", groupID))
}

// AssignToVPS replaces the firewall groups of a VPS (at most 10, in priority order). An
// empty list removes them all. The new rules are applied in the background.
func (s *firewallService) AssignToVPS(ctx context.Context, vpsID int, req *VPSFirewallGroupsRequest) (*VPSFirewallGroupsResponse, error) {
	var result VPSFirewallGroupsResponse
	if err := s.client.put(ctx, fmt.Sprintf("/firewall/vps/%d/groups", vpsID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
