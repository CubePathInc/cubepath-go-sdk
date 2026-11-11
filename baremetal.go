package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BaremetalService handles communication with the baremetal related methods of the CubePath API.
//
// Every method that takes baremetalID takes a server reference: the number (as text, "5") or
// the uuid, in any case. Anything else is rejected before a request is made. Use Baremetal.Ref
// to get the number of a server you already hold. The echo fields of other resources (route
// NextHopTarget, BGP PeerTarget, load balancer TargetUUID, alert TargetID) are different: they
// still take only the server number.
type BaremetalService interface {
	Deploy(ctx context.Context, projectID int, req *CreateBaremetalRequest) (*TaskResponse, error)
	List(ctx context.Context) ([]ProjectResponse, error)
	// Get finds the server in the project listing by number or uuid. When it is not there it
	// returns an *APIError with status 404 (see IsNotFound).
	Get(ctx context.Context, baremetalID string) (*Baremetal, error)
	Update(ctx context.Context, baremetalID string, req *UpdateBaremetalRequest) error
	Power(ctx context.Context, baremetalID string, action string) error
	Rescue(ctx context.Context, baremetalID string) (*RescueResponse, error)
	ResetBMC(ctx context.Context, baremetalID string) error
	BMCSensors(ctx context.Context, baremetalID string) (*BMCSensors, error)
	IPMISession(ctx context.Context, baremetalID string) (*IPMISession, error)
	Reinstall(ctx context.Context, baremetalID string, req *ReinstallBaremetalRequest) error
	ReinstallStatus(ctx context.Context, baremetalID string) (*ReinstallStatus, error)
	CancelReinstall(ctx context.Context, baremetalID string) error
	MonitoringEnable(ctx context.Context, baremetalID string) error
	MonitoringDisable(ctx context.Context, baremetalID string) error
	// ListModels returns the server models for sale per location, with their stock.
	ListModels(ctx context.Context) ([]BaremetalModelLocation, error)
	// ListOS returns the operating systems a server can be reinstalled with, each with the
	// disk layouts compatible with its hardware.
	ListOS(ctx context.Context, baremetalID string) ([]BaremetalOS, error)
	// GetKVM returns the KVM console access of a server (the access is audit logged).
	GetKVM(ctx context.Context, baremetalID string) (*BaremetalKVM, error)
	// SetProtection enables or disables destruction protection.
	SetProtection(ctx context.Context, baremetalID string, enabled bool) error
	MoveToProject(ctx context.Context, baremetalID string, projectID int) error
	// AddSSHKeys associates SSH keys with the server; they are installed on the next reinstall.
	AddSSHKeys(ctx context.Context, baremetalID string, sshKeyIDs []int) error
	RemoveSSHKey(ctx context.Context, baremetalID string, sshKeyID int) error
	// AttachNetwork attaches the server to a private network of the same location; restart
	// it to apply the change.
	AttachNetwork(ctx context.Context, baremetalID string, networkID int) error
	// DetachNetwork detaches the server from its private network; restart it to apply.
	DetachNetwork(ctx context.Context, baremetalID string) error
}

// Baremetal represents a baremetal server.
type Baremetal struct {
	// ID is the server number. It is the same number before and after the API moved to uuids.
	ID int `json:"id"`
	// UUID is the server uuid, empty when the API does not send it. It is accepted wherever a
	// baremetal reference is (see Ref).
	UUID             string         `json:"uuid"`
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

// Ref returns the number of the server as text, the form every Baremetal method takes as
// its reference (UUID works too, when set).
func (b *Baremetal) Ref() string {
	return strconv.Itoa(b.ID)
}

// UnmarshalJSON reads "id" as a number or as a numeric string. A uuid sent as "id" is
// kept in UUID, so a listing never fails to decode because of it.
func (b *Baremetal) UnmarshalJSON(data []byte) error {
	type plain Baremetal
	aux := struct {
		*plain
		ID json.RawMessage `json:"id"`
	}{plain: (*plain)(b)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	raw := strings.TrimSpace(string(aux.ID))
	if raw == "" || raw == "null" {
		return nil
	}
	text := strings.Trim(raw, `"`)
	if n, err := strconv.Atoi(text); err == nil {
		b.ID = n
		return nil
	}
	if raw[0] == '"' && isUUID(text) {
		if b.UUID == "" {
			b.UUID = strings.ToLower(text)
		}
		return nil
	}
	return fmt.Errorf("baremetal id %s is neither a number nor a uuid", raw)
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

func (s *baremetalService) Get(ctx context.Context, baremetalID string) (*Baremetal, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	projects, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		for i := range p.Baremetals {
			if sameBaremetalRef(baremetalID, p.Baremetals[i].ID, p.Baremetals[i].UUID) {
				return &p.Baremetals[i], nil
			}
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "Not Found", Detail: fmt.Sprintf("baremetal server %s not found", baremetalID)}
}

func (s *baremetalService) Update(ctx context.Context, baremetalID string, req *UpdateBaremetalRequest) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.patch(ctx, fmt.Sprintf("/baremetal/update/%s", baremetalID), req, nil)
}

func (s *baremetalService) Power(ctx context.Context, baremetalID string, action string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/power/%s", baremetalID, action), nil, nil)
}

func (s *baremetalService) Rescue(ctx context.Context, baremetalID string) (*RescueResponse, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	var result RescueResponse
	if err := s.client.post(ctx, fmt.Sprintf("/baremetal/%s/rescue", baremetalID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *baremetalService) ResetBMC(ctx context.Context, baremetalID string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/reset-bmc", baremetalID), nil, nil)
}

// BMCSensors returns the temperatures and fan speeds the BMC reported on its last poll.
// Sensors are served through GraphQL; the old REST /bmc-sensors endpoint no longer exists.
func (s *baremetalService) BMCSensors(ctx context.Context, baremetalID string) (*BMCSensors, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
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
	if err := s.client.graphQL(ctx, query, map[string]interface{}{"id": baremetalID}, &data); err != nil {
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

func (s *baremetalService) IPMISession(ctx context.Context, baremetalID string) (*IPMISession, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	var result IPMISession
	if err := s.client.post(ctx, fmt.Sprintf("/ipmi-proxy/create-session/%s", baremetalID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *baremetalService) Reinstall(ctx context.Context, baremetalID string, req *ReinstallBaremetalRequest) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/reinstall", baremetalID), req, nil)
}

// ReinstallStatus reports whether an OS reinstallation is running. The API no longer has a
// dedicated status endpoint: a server is reinstalling while its status is "deploying".
func (s *baremetalService) ReinstallStatus(ctx context.Context, baremetalID string) (*ReinstallStatus, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	bm, err := s.Get(ctx, baremetalID)
	if err != nil {
		return nil, err
	}
	return &ReinstallStatus{IsReinstalling: bm.Status == "deploying", Status: bm.Status}, nil
}

// CancelReinstall cancels a pending or running OS reinstallation.
func (s *baremetalService) CancelReinstall(ctx context.Context, baremetalID string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%s/reinstall", baremetalID))
}

func (s *baremetalService) MonitoringEnable(ctx context.Context, baremetalID string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.put(ctx, fmt.Sprintf("/baremetal/%s/monitoring?enable=true", baremetalID), nil, nil)
}

func (s *baremetalService) MonitoringDisable(ctx context.Context, baremetalID string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.put(ctx, fmt.Sprintf("/baremetal/%s/monitoring?enable=false", baremetalID), nil, nil)
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

func (s *baremetalService) ListOS(ctx context.Context, baremetalID string) ([]BaremetalOS, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	var systems []BaremetalOS
	if err := s.client.get(ctx, fmt.Sprintf("/baremetal/os/%s", baremetalID), &systems); err != nil {
		return nil, err
	}
	return systems, nil
}

func (s *baremetalService) GetKVM(ctx context.Context, baremetalID string) (*BaremetalKVM, error) {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return nil, err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	var kvm BaremetalKVM
	if err := s.client.get(ctx, fmt.Sprintf("/baremetal/%s/kvm", baremetalID), &kvm); err != nil {
		return nil, err
	}
	return &kvm, nil
}

func (s *baremetalService) SetProtection(ctx context.Context, baremetalID string, enabled bool) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	body := map[string]interface{}{
		"enabled": enabled,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/protection", baremetalID), body, nil)
}

func (s *baremetalService) MoveToProject(ctx context.Context, baremetalID string, projectID int) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/move-project", baremetalID), body, nil)
}

func (s *baremetalService) AddSSHKeys(ctx context.Context, baremetalID string, sshKeyIDs []int) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	if len(sshKeyIDs) == 0 {
		return fmt.Errorf("at least one SSH key id is required")
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/ssh-keys", baremetalID), sshKeyIDs, nil)
}

func (s *baremetalService) RemoveSSHKey(ctx context.Context, baremetalID string, sshKeyID int) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%s/ssh-keys/%d", baremetalID, sshKeyID))
}

func (s *baremetalService) AttachNetwork(ctx context.Context, baremetalID string, networkID int) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	body := map[string]interface{}{
		"network_id": networkID,
	}
	return s.client.post(ctx, fmt.Sprintf("/baremetal/%s/network", baremetalID), body, nil)
}

func (s *baremetalService) DetachNetwork(ctx context.Context, baremetalID string) error {
	if err := validateBaremetalRef(baremetalID); err != nil {
		return err
	}
	baremetalID = normalizeBaremetalRef(baremetalID)
	return s.client.del(ctx, fmt.Sprintf("/baremetal/%s/network", baremetalID))
}
