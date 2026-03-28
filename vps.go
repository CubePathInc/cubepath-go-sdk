package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// VPSService handles communication with the VPS related methods of the CubePath API.
type VPSService interface {
	Create(ctx context.Context, projectID int, req *CreateVPSRequest) (*TaskResponse, error)
	List(ctx context.Context) ([]ProjectResponse, error)
	Get(ctx context.Context, vpsID int) (*VPS, error)
	Destroy(ctx context.Context, vpsID int, releaseIPs bool) error
	Update(ctx context.Context, vpsID int, req *UpdateVPSRequest) error
	Resize(ctx context.Context, vpsID int, planName string) error
	ChangePassword(ctx context.Context, vpsID int, password string) error
	Reinstall(ctx context.Context, vpsID int, templateName string) error
	Power(ctx context.Context, vpsID int, action string) error
	Backups() VPSBackupService
	ISOs() VPSISOService
}

// VPS represents a VPS instance.
type VPS struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Label       string          `json:"label"`
	ProjectID   int             `json:"project_id"`
	Status      string          `json:"status"`
	User        string          `json:"user"`
	Plan        VPSPlan         `json:"plan"`
	Template    VPSTemplate     `json:"template"`
	Location    Location        `json:"location"`
	FloatingIPs json.RawMessage `json:"floating_ips"`
	IPv4        string          `json:"ipv4"`
	IPv6        string          `json:"ipv6"`
	Network     *NetworkInfo    `json:"network,omitempty"`
	SSHKeys     []SSHKey        `json:"ssh_keys"`
	CreatedAt   time.Time       `json:"created_at"`
}

// VPSPlan represents a VPS plan.
type VPSPlan struct {
	ID           int    `json:"id"`
	PlanName     string `json:"plan_name"`
	CPU          int    `json:"cpu"`
	RAM          int    `json:"ram"`
	Storage      int    `json:"storage"`
	Bandwidth    int    `json:"bandwidth"`
	PricePerHour string `json:"price_per_hour"`
}

// VPSTemplate represents a VPS template/operating system.
type VPSTemplate struct {
	ID           int    `json:"id"`
	TemplateName string `json:"template_name"`
	OSName       string `json:"os_name"`
	Version      string `json:"version"`
}

// Location represents a datacenter location.
type Location struct {
	ID           int    `json:"id"`
	LocationName string `json:"location_name"`
	Description  string `json:"description"`
}

// NetworkInfo represents network information for a VPS.
type NetworkInfo struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	AssignedIP string `json:"assigned_ip"`
}

// TaskResponse represents a response with a task ID.
type TaskResponse struct {
	TaskID  string `json:"task_id,omitempty"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// CreateVPSRequest represents a request to create a VPS.
type CreateVPSRequest struct {
	Name                  string   `json:"name"`
	PlanName              string   `json:"plan_name"`
	TemplateName          string   `json:"template_name"`
	LocationName          string   `json:"location_name"`
	Label                 string   `json:"label,omitempty"`
	NetworkID             *int     `json:"network_id,omitempty"`
	SSHKeyNames           []string `json:"ssh_key_names,omitempty"`
	User                  string   `json:"user,omitempty"`
	Password              string   `json:"password,omitempty"`
	IPv4                  *bool    `json:"ipv4,omitempty"`
	EnableBackups         *bool    `json:"enable_backups,omitempty"`
	CustomCloudInit       *string  `json:"custom_cloudinit,omitempty"`
	FirewallGroupIDs      []int    `json:"firewall_group_ids,omitempty"`
	AvailabilityGroupUUID *string  `json:"availability_group_uuid,omitempty"`
}

// UpdateVPSRequest represents a request to update a VPS.
type UpdateVPSRequest struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

type vpsService struct {
	client  *Client
	backups *vpsBackupService
	isos    *vpsISOService
}

func (s *vpsService) Create(ctx context.Context, projectID int, req *CreateVPSRequest) (*TaskResponse, error) {
	var result TaskResponse
	if err := s.client.post(ctx, fmt.Sprintf("/vps/create/%d", projectID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *vpsService) List(ctx context.Context) ([]ProjectResponse, error) {
	var projects []ProjectResponse
	if err := s.client.get(ctx, "/projects/", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *vpsService) Get(ctx context.Context, vpsID int) (*VPS, error) {
	projects, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		for i := range p.VPS {
			if p.VPS[i].ID == vpsID {
				return &p.VPS[i], nil
			}
		}
	}
	return nil, fmt.Errorf("VPS %d not found", vpsID)
}

func (s *vpsService) Destroy(ctx context.Context, vpsID int, releaseIPs bool) error {
	body := map[string]interface{}{
		"release_ips": releaseIPs,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/destroy/%d", vpsID), body, nil)
}

func (s *vpsService) Update(ctx context.Context, vpsID int, req *UpdateVPSRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/vps/update/%d", vpsID), req, nil)
}

func (s *vpsService) Resize(ctx context.Context, vpsID int, planName string) error {
	return s.client.post(ctx, fmt.Sprintf("/vps/resize/vps_id/%d/resize_plan/%s", vpsID, planName), nil, nil)
}

func (s *vpsService) ChangePassword(ctx context.Context, vpsID int, password string) error {
	body := map[string]interface{}{
		"password": password,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/change-password", vpsID), body, nil)
}

func (s *vpsService) Reinstall(ctx context.Context, vpsID int, templateName string) error {
	body := map[string]interface{}{
		"template_name": templateName,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/reinstall/%d", vpsID), body, nil)
}

func (s *vpsService) Power(ctx context.Context, vpsID int, action string) error {
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/power/%s", vpsID, action), nil, nil)
}

func (s *vpsService) Backups() VPSBackupService {
	if s.backups == nil {
		s.backups = &vpsBackupService{client: s.client}
	}
	return s.backups
}

func (s *vpsService) ISOs() VPSISOService {
	if s.isos == nil {
		s.isos = &vpsISOService{client: s.client}
	}
	return s.isos
}
