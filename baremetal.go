package cubepath

import (
	"context"
	"fmt"
	"time"
)

// BaremetalService handles communication with the baremetal related methods of the CubePath API.
type BaremetalService interface {
	Deploy(ctx context.Context, projectID int, req *CreateBaremetalRequest) (*TaskResponse, error)
	List(ctx context.Context) ([]ProjectResponse, error)
	Get(ctx context.Context, baremetalID int) (*Baremetal, error)
	Update(ctx context.Context, baremetalID int, req *UpdateBaremetalRequest) error
	Power(ctx context.Context, baremetalID int, action string) error
	Rescue(ctx context.Context, baremetalID int) (*RescueResponse, error)
	ResetBMC(ctx context.Context, baremetalID int) error
	BMCSensors(ctx context.Context, baremetalID int) (*BMCSensors, error)
	IPMISession(ctx context.Context, baremetalID int) (*IPMISession, error)
	Reinstall(ctx context.Context, baremetalID int, req *ReinstallBaremetalRequest) error
	ReinstallStatus(ctx context.Context, baremetalID int) (*ReinstallStatus, error)
	CancelReinstall(ctx context.Context, baremetalID int) error
	MonitoringEnable(ctx context.Context, baremetalID int) error
	MonitoringDisable(ctx context.Context, baremetalID int) error
	// ListModels returns the server models for sale per location, with their stock.
	ListModels(ctx context.Context) ([]BaremetalModelLocation, error)
	// ListOS returns the operating systems a server can be reinstalled with, each with the
	// disk layouts compatible with its hardware.
	ListOS(ctx context.Context, baremetalID int) ([]BaremetalOS, error)
	// GetKVM returns the KVM console access of a server (the access is audit logged).
	GetKVM(ctx context.Context, baremetalID int) (*BaremetalKVM, error)
	// SetProtection enables or disables destruction protection.
	SetProtection(ctx context.Context, baremetalID int, enabled bool) error
	MoveToProject(ctx context.Context, baremetalID, projectID int) error
	// AddSSHKeys associates SSH keys with the server; they are installed on the next reinstall.
	AddSSHKeys(ctx context.Context, baremetalID int, sshKeyIDs []int) error
	RemoveSSHKey(ctx context.Context, baremetalID, sshKeyID int) error
	// AttachNetwork attaches the server to a private network of the same location; restart
	// it to apply the change.
	AttachNetwork(ctx context.Context, baremetalID, networkID int) error
	// DetachNetwork detaches the server from its private network; restart it to apply.
	DetachNetwork(ctx context.Context, baremetalID int) error
}

// Baremetal represents a baremetal server.
type Baremetal struct {
	ID               int            `json:"id"`
	Hostname         string         `json:"hostname"`
	Label            string         `json:"label"`
	ProjectID        int            `json:"project_id"`
	Status           string         `json:"status"`
	User             string         `json:"user"`
	OS               *OSInfo        `json:"os"`
	Location         Location       `json:"location"`
	BaremetalModel   BaremetalModel `json:"baremetal_model"`
	FloatingIPs      []FloatingIP   `json:"floating_ips"`
	MonitoringEnable bool           `json:"monitoring_enable"`
	SSHUsername      string         `json:"ssh_username"`
	SSHKey           *SSHKeyRef     `json:"ssh_key,omitempty"`
	Protected        bool           `json:"protected"`
	CreatedAt        time.Time      `json:"created_at"`
}

// SSHKeyRef represents a reference to an SSH key.
type SSHKeyRef struct {
	Name string `json:"name"`
}

// BaremetalModel represents a baremetal server model.
type BaremetalModel struct {
	ID          int     `json:"id"`
	ModelName   string  `json:"model_name"`
	CPU         string  `json:"cpu"`
	CPUSpecs    string  `json:"cpu_specs"`
	CPUBench    float64 `json:"cpu_bench"`
	RAM         int     `json:"ram"`
	RAMSize     int     `json:"ram_size"`
	RAMType     string  `json:"ram_type"`
	StorageType string  `json:"storage_type"`
	DiskCount   int     `json:"disk_count"`
	DiskSize    string  `json:"disk_size"`
	DiskType    string  `json:"disk_type"`
	Port        int     `json:"port"`
	KVM         string  `json:"kvm"`
	Price       float64 `json:"price"`
}

// OSInfo represents operating system information.
type OSInfo struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CreateBaremetalRequest represents a request to deploy a baremetal server.
type CreateBaremetalRequest struct {
	ModelName      string `json:"model_name"`
	LocationName   string `json:"location_name"`
	Hostname       string `json:"hostname"`
	Label          string `json:"label,omitempty"`
	User           string `json:"user,omitempty"`
	Password       string `json:"password"`
	SSHKeyIDs      []int  `json:"ssh_key_ids,omitempty"`
	OSName         string `json:"os_name,omitempty"`
	DiskLayoutName string `json:"disk_layout_name,omitempty"`
}

// UpdateBaremetalRequest represents a request to update a baremetal server.
type UpdateBaremetalRequest struct {
	Hostname *string `json:"hostname,omitempty"`
	Label    *string `json:"label,omitempty"`
	Tags     *string `json:"tags,omitempty"`
}

// ReinstallBaremetalRequest represents a request to reinstall a baremetal OS.
type ReinstallBaremetalRequest struct {
	OSName         string `json:"os_name"`
	DiskLayoutName string `json:"disk_layout_name,omitempty"`
	User           string `json:"user,omitempty"`
	Password       string `json:"password"`
	Hostname       string `json:"hostname,omitempty"`
	SSHKeyIDs      []int  `json:"ssh_key_ids,omitempty"`
}

// RescueResponse represents the response from activating rescue mode.
type RescueResponse struct {
	Detail   string `json:"detail"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// BMCSensors represents BMC sensor data.
type BMCSensors struct {
	// Deprecated: no longer returned by the API; always empty.
	Node string `json:"node"`
	// IPMIAvailable and PowerOn are false when the BMC has not been polled recently
	// (see LastSeen).
	IPMIAvailable bool `json:"ipmi_available"`
	PowerOn       bool `json:"power_on"`
	// LastSeen is the Unix time of the last BMC poll, 0 when never polled.
	LastSeen int64 `json:"last_seen"`
	Sensors  struct {
		Temperatures []SensorReading `json:"temperatures"`
		Fans         []SensorReading `json:"fans"`
	} `json:"sensors"`
}

// SensorReading represents a single sensor reading.
type SensorReading struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	// Unit is CELSIUS for temperatures and RPM for fans.
	Unit string `json:"unit"`
}

// IPMISession represents an IPMI proxy session.
type IPMISession struct {
	ProxyURL    string `json:"proxy_url"`
	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"credentials"`
}

// ReinstallStatus represents the status of a baremetal reinstallation.
type ReinstallStatus struct {
	IsReinstalling bool   `json:"is_reinstalling"`
	Status         string `json:"status"`
	// Deprecated: no longer available; always empty.
	OSName string `json:"os_name"`
}

// BaremetalModelLocation represents the models for sale at a location.
type BaremetalModelLocation struct {
	LocationName string                `json:"location_name"`
	Description  string                `json:"description"`
	Models       []BaremetalModelOffer `json:"models"`
}

// BaremetalModelOffer is a server model for sale. Price is the monthly price before the
// discount (DiscountValue, of DiscountType "fixed" or "percent"); Port is in Gbps.
type BaremetalModelOffer struct {
	ModelName      string   `json:"model_name"`
	Price          float64  `json:"price"`
	DiscountValue  float64  `json:"discount_value"`
	DiscountType   string   `json:"discount_type"`
	CPU            string   `json:"cpu"`
	CPUSpecs       string   `json:"cpu_specs"`
	CPUBench       *float64 `json:"cpu_bench"`
	RAMSize        int      `json:"ram_size"`
	RAMType        string   `json:"ram_type"`
	DiskSize       string   `json:"disk_size"`
	DiskType       string   `json:"disk_type"`
	Port           int      `json:"port"`
	Setup          float64  `json:"setup"`
	KVM            string   `json:"kvm"`
	StockAvailable int      `json:"stock_available"`
}

// BaremetalDiskLayout is a disk layout usable when installing an operating system.
type BaremetalDiskLayout struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	DiskLayoutName string `json:"disk_layout_name"`
	DiskType       string `json:"disk_type"`
	RAIDType       string `json:"raid_type"`
	DiskCount      int    `json:"disk_count"`
}

// BaremetalOS is an operating system with its compatible disk layouts. OSName is the value to
// pass as ReinstallBaremetalRequest.OSName.
type BaremetalOS struct {
	ID              int                   `json:"id"`
	OSName          string                `json:"os_name"`
	OperatingSystem string                `json:"operating_system"`
	DiskLayouts     []BaremetalDiskLayout `json:"disk_layouts"`
}

// BaremetalKVM is the KVM console access of a server.
type BaremetalKVM struct {
	URL       string  `json:"url"`
	Username  string  `json:"username"`
	Password  *string `json:"password"`
	UpdatedAt string  `json:"updated_at"`
}

type baremetalService struct {
	client *Client
}

func (s *baremetalService) Deploy(ctx context.Context, projectID int, req *CreateBaremetalRequest) (*TaskResponse, error) {
	var result TaskResponse
	if err := s.client.post(ctx, fmt.Sprintf("/baremetal/deploy/%d", projectID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *baremetalService) List(ctx context.Context) ([]ProjectResponse, error) {
	var projects []ProjectResponse
	if err := s.client.get(ctx, "/projects/", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *baremetalService) Get(ctx context.Context, baremetalID int) (*Baremetal, error) {
	projects, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		for i := range p.Baremetals {
			if p.Baremetals[i].ID == baremetalID {
				return &p.Baremetals[i], nil
			}
		}
	}
	return nil, fmt.Errorf("baremetal server %d not found", baremetalID)
}

func (s *baremetalService) Update(ctx context.Context, baremetalID int, req *UpdateBaremetalRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/baremetal/update/%d", baremetalID), req, nil)
}

func (s *baremetalService) Power(ctx context.Context, baremetalID int, action string) error {
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/power/%s", baremetalID, action), nil, nil)
}

func (s *baremetalService) Rescue(ctx context.Context, baremetalID int) (*RescueResponse, error) {
	var result RescueResponse
	if err := s.client.post(ctx, fmt.Sprintf("/baremetal/%d/rescue", baremetalID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *baremetalService) ResetBMC(ctx context.Context, baremetalID int) error {
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/reset-bmc", baremetalID), nil, nil)
}

// BMCSensors returns the temperatures and fan speeds the BMC reported on its last poll.
// Sensors are served through GraphQL; the old REST /bmc-sensors endpoint no longer exists.
func (s *baremetalService) BMCSensors(ctx context.Context, baremetalID int) (*BMCSensors, error) {
	type sensor struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
		Unit  string  `json:"unit"`
	}
	var data struct {
		Baremetal *struct {
			Sensors struct {
				IPMIAvailable *bool    `json:"ipmiAvailable"`
				PowerOn       *bool    `json:"powerOn"`
				LastSeen      *int64   `json:"lastSeen"`
				Temperatures  []sensor `json:"temperatures"`
				Fans          []sensor `json:"fans"`
			} `json:"sensors"`
		} `json:"baremetal"`
	}
	query := `query($id: ID!) { baremetal(id: $id) { sensors { ipmiAvailable powerOn lastSeen temperatures { name value unit } fans { name value unit } } } }`
	if err := s.client.graphQL(ctx, query, map[string]interface{}{"id": fmt.Sprint(baremetalID)}, &data); err != nil {
		return nil, err
	}
	if data.Baremetal == nil {
		return nil, &APIError{StatusCode: 404, Message: "Not Found", Detail: "Baremetal not found"}
	}
	in := data.Baremetal.Sensors
	result := &BMCSensors{}
	if in.IPMIAvailable != nil {
		result.IPMIAvailable = *in.IPMIAvailable
	}
	if in.PowerOn != nil {
		result.PowerOn = *in.PowerOn
	}
	if in.LastSeen != nil {
		result.LastSeen = *in.LastSeen
	}
	result.Sensors.Temperatures = make([]SensorReading, 0, len(in.Temperatures))
	for _, t := range in.Temperatures {
		result.Sensors.Temperatures = append(result.Sensors.Temperatures, SensorReading(t))
	}
	result.Sensors.Fans = make([]SensorReading, 0, len(in.Fans))
	for _, f := range in.Fans {
		result.Sensors.Fans = append(result.Sensors.Fans, SensorReading(f))
	}
	return result, nil
}

func (s *baremetalService) IPMISession(ctx context.Context, baremetalID int) (*IPMISession, error) {
	var result IPMISession
	if err := s.client.post(ctx, fmt.Sprintf("/ipmi-proxy/create-session/%d", baremetalID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *baremetalService) Reinstall(ctx context.Context, baremetalID int, req *ReinstallBaremetalRequest) error {
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/reinstall", baremetalID), req, nil)
}

// ReinstallStatus reports whether an OS reinstallation is running. The API no longer has a
// dedicated status endpoint: a server is reinstalling while its status is "deploying".
func (s *baremetalService) ReinstallStatus(ctx context.Context, baremetalID int) (*ReinstallStatus, error) {
	bm, err := s.Get(ctx, baremetalID)
	if err != nil {
		return nil, err
	}
	return &ReinstallStatus{IsReinstalling: bm.Status == "deploying", Status: bm.Status}, nil
}

// CancelReinstall cancels a pending or running OS reinstallation.
func (s *baremetalService) CancelReinstall(ctx context.Context, baremetalID int) error {
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%d/reinstall", baremetalID))
}

func (s *baremetalService) MonitoringEnable(ctx context.Context, baremetalID int) error {
	return s.client.put(ctx, fmt.Sprintf("/baremetal/%d/monitoring?enable=true", baremetalID), nil, nil)
}

func (s *baremetalService) MonitoringDisable(ctx context.Context, baremetalID int) error {
	return s.client.put(ctx, fmt.Sprintf("/baremetal/%d/monitoring?enable=false", baremetalID), nil, nil)
}

func (s *baremetalService) ListModels(ctx context.Context) ([]BaremetalModelLocation, error) {
	var result struct {
		Locations []BaremetalModelLocation `json:"locations"`
	}
	if err := s.client.get(ctx, "/baremetal/models", &result); err != nil {
		return nil, err
	}
	return result.Locations, nil
}

func (s *baremetalService) ListOS(ctx context.Context, baremetalID int) ([]BaremetalOS, error) {
	var systems []BaremetalOS
	if err := s.client.get(ctx, fmt.Sprintf("/baremetal/os/%d", baremetalID), &systems); err != nil {
		return nil, err
	}
	return systems, nil
}

func (s *baremetalService) GetKVM(ctx context.Context, baremetalID int) (*BaremetalKVM, error) {
	var kvm BaremetalKVM
	if err := s.client.get(ctx, fmt.Sprintf("/baremetal/%d/kvm", baremetalID), &kvm); err != nil {
		return nil, err
	}
	return &kvm, nil
}

func (s *baremetalService) SetProtection(ctx context.Context, baremetalID int, enabled bool) error {
	body := map[string]interface{}{
		"enabled": enabled,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/protection", baremetalID), body, nil)
}

func (s *baremetalService) MoveToProject(ctx context.Context, baremetalID, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/move-project", baremetalID), body, nil)
}

func (s *baremetalService) AddSSHKeys(ctx context.Context, baremetalID int, sshKeyIDs []int) error {
	if len(sshKeyIDs) == 0 {
		return fmt.Errorf("at least one SSH key id is required")
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/ssh-keys", baremetalID), sshKeyIDs, nil)
}

func (s *baremetalService) RemoveSSHKey(ctx context.Context, baremetalID, sshKeyID int) error {
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%d/ssh-keys/%d", baremetalID, sshKeyID))
}

func (s *baremetalService) AttachNetwork(ctx context.Context, baremetalID, networkID int) error {
	body := map[string]interface{}{
		"network_id": networkID,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%d/network", baremetalID), body, nil)
}

func (s *baremetalService) DetachNetwork(ctx context.Context, baremetalID int) error {
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%d/network", baremetalID))
}
