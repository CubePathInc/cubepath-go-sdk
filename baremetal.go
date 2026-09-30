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
