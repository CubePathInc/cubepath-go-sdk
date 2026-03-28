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
	Name    string         `json:"name"`
	Rules   []FirewallRule `json:"rules"`
	Enabled bool           `json:"enabled"`
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
	if err := s.client.post(ctx, "/firewall/groups", req, &group); err != nil {
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

func (s *firewallService) Get(ctx context.Context, groupID int) (*FirewallGroup, error) {
	var group FirewallGroup
	if err := s.client.get(ctx, fmt.Sprintf("/firewall/groups/%d", groupID), &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *firewallService) Update(ctx context.Context, groupID int, req *UpdateFirewallGroupRequest) (*FirewallGroup, error) {
	var group FirewallGroup
	if err := s.client.patch(ctx, fmt.Sprintf("/firewall/groups/%d", groupID), req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

func (s *firewallService) Delete(ctx context.Context, groupID int) error {
	return s.client.del(ctx, fmt.Sprintf("/firewall/groups/%d", groupID))
}

func (s *firewallService) AssignToVPS(ctx context.Context, vpsID int, req *VPSFirewallGroupsRequest) (*VPSFirewallGroupsResponse, error) {
	var result VPSFirewallGroupsResponse
	if err := s.client.post(ctx, fmt.Sprintf("/vps/%d/firewall-groups", vpsID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
