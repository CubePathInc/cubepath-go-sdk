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
	Templates(ctx context.Context) (*VPSTemplatesResponse, error)
	// Plans returns the orderable plans per location and cluster.
	Plans(ctx context.Context) (*VPSPlansResponse, error)
	// SetProtection enables or disables destruction protection: a protected VPS cannot be
	// destroyed or reinstalled.
	SetProtection(ctx context.Context, vpsID int, enabled bool) error
	MoveToProject(ctx context.Context, vpsID, projectID int) error
	// AddSSHKeys associates SSH keys with the VPS. Keys are only installed on the next
	// reinstall; the running system is not changed.
	AddSSHKeys(ctx context.Context, vpsID int, sshKeyIDs []int) error
	// RemoveSSHKey removes the association of an SSH key. The running system is not changed.
	RemoveSSHKey(ctx context.Context, vpsID, sshKeyID int) error
	// AttachNetwork attaches the VPS to a private network of the same location. The IP is
	// assigned automatically; restart the VPS to apply the change.
	AttachNetwork(ctx context.Context, vpsID, networkID int) error
	// DetachNetwork detaches the VPS from its private network; restart it to apply the change.
	DetachNetwork(ctx context.Context, vpsID int) error
	// VNCURL opens a console session (valid 5 minutes) on a running VPS. Connect a noVNC
	// client to WebSocketURL and use Ticket as the VNC password.
	VNCURL(ctx context.Context, vpsID int) (*VNCSession, error)
	Backups() VPSBackupService
	ISOs() VPSISOService
	AvailabilityGroups() VPSAvailabilityGroupService
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
	Protected   bool            `json:"protected"`
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
	// Status is only set by Plans: 2 orderable, 1 out of stock.
	Status int `json:"status,omitempty"`
}

// VPSPlansResponse represents the plans available per location.
type VPSPlansResponse struct {
	Locations []VPSPlanLocation `json:"locations"`
}

// VPSPlanLocation represents the plans of one location, grouped by cluster.
type VPSPlanLocation struct {
	LocationName string           `json:"location_name"`
	Description  string           `json:"description"`
	Clusters     []VPSPlanCluster `json:"clusters"`
}

// VPSPlanCluster represents a group of plans (for example "General Purpose").
type VPSPlanCluster struct {
	ClusterName string    `json:"cluster_name"`
	Type        string    `json:"type"`
	Plans       []VPSPlan `json:"plans"`
}

// VNCSession is a console session of a VPS.
type VNCSession struct {
	WebSocketURL string `json:"websocket_url"`
	SessionID    string `json:"session_id"`
	VNCInfo      struct {
		Ticket string `json:"ticket"`
	} `json:"vnc_info"`
}

// VPSTemplate represents a VPS template/operating system.
type VPSTemplate struct {
	ID           int    `json:"id"`
	TemplateName string `json:"template_name"`
	OSName       string `json:"os_name"`
	Version      string `json:"version"`
}

// VPSTemplatesResponse represents the response from the VPS templates endpoint.
type VPSTemplatesResponse struct {
	OperatingSystems []VPSTemplate    `json:"operating_systems"`
	Applications     []VPSAppTemplate `json:"applications"`
}

// VPSAppTemplate represents an application template.
type VPSAppTemplate struct {
	AppName         string `json:"app_name"`
	Version         string `json:"version"`
	RecommendedPlan string `json:"recommended_plan"`
	AppDocs         string `json:"app_docs"`
	AppWiki         string `json:"app_wiki"`
	LicenseType     string `json:"license_type"`
	Description     string `json:"description"`
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

// CreateVPSRequest represents a request to create a VPS. Label is always sent (the API
// requires the field; it may be empty).
type CreateVPSRequest struct {
	Name                  string  `json:"name"`
	PlanName              string  `json:"plan_name"`
	TemplateName          string  `json:"template_name"`
	LocationName          string  `json:"location_name"`
	Label                 string  `json:"label"`
	NetworkID             *int    `json:"network_id,omitempty"`
	SSHKeyIDs             []int   `json:"ssh_key_ids,omitempty"`
	User                  string  `json:"user,omitempty"`
	Password              string  `json:"password,omitempty"`
	IPv4                  *bool   `json:"ipv4,omitempty"`
	IPv6                  *bool   `json:"ipv6,omitempty"`
	EnableBackups         *bool   `json:"enable_backups,omitempty"`
	CustomCloudInit       *string `json:"custom_cloudinit,omitempty"`
	FirewallGroupIDs      []int   `json:"firewall_group_ids,omitempty"`
	AvailabilityGroupUUID *string `json:"availability_group_uuid,omitempty"`
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
	groups  *vpsAvailabilityGroupService
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

func (s *vpsService) Templates(ctx context.Context) (*VPSTemplatesResponse, error) {
	var templates VPSTemplatesResponse
	if err := s.client.get(ctx, "/vps/templates", &templates); err != nil {
		return nil, err
	}
	return &templates, nil
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

func (s *vpsService) Plans(ctx context.Context) (*VPSPlansResponse, error) {
	var plans VPSPlansResponse
	if err := s.client.get(ctx, "/vps/plans", &plans); err != nil {
		return nil, err
	}
	return &plans, nil
}

func (s *vpsService) SetProtection(ctx context.Context, vpsID int, enabled bool) error {
	body := map[string]interface{}{
		"enabled": enabled,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/protection", vpsID), body, nil)
}

func (s *vpsService) MoveToProject(ctx context.Context, vpsID, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/move-project", vpsID), body, nil)
}

func (s *vpsService) AddSSHKeys(ctx context.Context, vpsID int, sshKeyIDs []int) error {
	if len(sshKeyIDs) == 0 {
		return fmt.Errorf("at least one SSH key id is required")
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/ssh-keys", vpsID), sshKeyIDs, nil)
}

func (s *vpsService) RemoveSSHKey(ctx context.Context, vpsID, sshKeyID int) error {
	return s.client.del(ctx, fmt.Sprintf("/vps/%d/ssh-keys/%d", vpsID, sshKeyID))
}

func (s *vpsService) AttachNetwork(ctx context.Context, vpsID, networkID int) error {
	body := map[string]interface{}{
		"network_id": networkID,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/network", vpsID), body, nil)
}

func (s *vpsService) DetachNetwork(ctx context.Context, vpsID int) error {
	return s.client.del(ctx, fmt.Sprintf("/vps/%d/network", vpsID))
}

func (s *vpsService) VNCURL(ctx context.Context, vpsID int) (*VNCSession, error) {
	var session VNCSession
	if err := s.client.post(ctx, fmt.Sprintf("/vps/%d/vnc-url", vpsID), nil, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *vpsService) AvailabilityGroups() VPSAvailabilityGroupService {
	if s.groups == nil {
		s.groups = &vpsAvailabilityGroupService{client: s.client}
	}
	return s.groups
}
