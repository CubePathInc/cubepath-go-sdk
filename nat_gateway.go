package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
)

// NATGatewayService handles communication with the NAT gateway related methods of the CubePath API.
type NATGatewayService interface {
	ListPlans(ctx context.Context) ([]NATGatewayLocationPlans, error)
	List(ctx context.Context) ([]NATGateway, error)
	Get(ctx context.Context, uuid string) (*NATGateway, error)
	Create(ctx context.Context, req *CreateNATGatewayRequest) (*NATGateway, error)
	Update(ctx context.Context, uuid string, req *UpdateNATGatewayRequest) (*NATGateway, error)
	Delete(ctx context.Context, uuid string) error
	Resize(ctx context.Context, uuid, planName string) error
	MoveToProject(ctx context.Context, uuid string, projectID int) error
	SetProtection(ctx context.Context, uuid string, enabled bool) error
	GetMetrics(ctx context.Context, uuid string) (json.RawMessage, error)
	GetBandwidthUsage(ctx context.Context, uuid string) (json.RawMessage, error)
}

// NATGatewayPlan represents a NAT gateway plan.
type NATGatewayPlan struct {
	Name                 string  `json:"name"`
	Description          string  `json:"description"`
	PricePerHour         float64 `json:"price_per_hour"`
	BandwidthMbps        int     `json:"bandwidth_mbps"`
	ConnectionsPerSecond int     `json:"connections_per_second"`
}

// NATGatewayLocationPlans represents plans available at a location.
type NATGatewayLocationPlans struct {
	LocationName        string           `json:"location_name"`
	LocationDescription string           `json:"location_description"`
	Plans               []NATGatewayPlan `json:"plans"`
}

// NATGatewayFloatingIP represents a floating IP associated with a NAT gateway.
type NATGatewayFloatingIP struct {
	Address string `json:"address"`
	Netmask string `json:"netmask"`
	Type    string `json:"type"`
	RDNS    string `json:"rdns"`
}

// NATGateway represents a NAT gateway.
type NATGateway struct {
	UUID                string                 `json:"uuid"`
	Name                string                 `json:"name"`
	Label               string                 `json:"label"`
	Status              string                 `json:"status"`
	PlanName            string                 `json:"plan_name"`
	LocationName        string                 `json:"location_name"`
	LocationDescription string                 `json:"location_description"`
	ProjectID           int                    `json:"project_id"`
	ProjectName         string                 `json:"project_name"`
	NetworkID           int                    `json:"network_id"`
	NetworkName         string                 `json:"network_name"`
	NetworkCIDR         string                 `json:"network_cidr"`
	PrivateIP           string                 `json:"private_ip"`
	MonthlyCharges      float64                `json:"monthly_charges"`
	Protected           bool                   `json:"protected"`
	FloatingIP          *NATGatewayFloatingIP  `json:"floating_ip,omitempty"`
	FloatingIPs         []NATGatewayFloatingIP `json:"floating_ips,omitempty"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
}

// CreateNATGatewayRequest represents a request to create a NAT gateway.
type CreateNATGatewayRequest struct {
	Name      string `json:"name"`
	Label     string `json:"label,omitempty"`
	PlanName  string `json:"plan_name"`
	NetworkID int    `json:"network_id"`
	ProjectID *int   `json:"project_id,omitempty"`
}

// UpdateNATGatewayRequest represents a request to update a NAT gateway.
type UpdateNATGatewayRequest struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

type natGatewayService struct {
	client *Client
}

func (s *natGatewayService) ListPlans(ctx context.Context) ([]NATGatewayLocationPlans, error) {
	var plans []NATGatewayLocationPlans
	if err := s.client.get(ctx, "/nat-gateway/plans", &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *natGatewayService) List(ctx context.Context) ([]NATGateway, error) {
	var gateways []NATGateway
	if err := s.client.get(ctx, "/nat-gateway/", &gateways); err != nil {
		return nil, err
	}
	return gateways, nil
}

func (s *natGatewayService) Get(ctx context.Context, uuid string) (*NATGateway, error) {
	var gateway NATGateway
	if err := s.client.get(ctx, fmt.Sprintf("/nat-gateway/%s", uuid), &gateway); err != nil {
		return nil, err
	}
	return &gateway, nil
}

func (s *natGatewayService) Create(ctx context.Context, req *CreateNATGatewayRequest) (*NATGateway, error) {
	var gateway NATGateway
	if err := s.client.post(ctx, "/nat-gateway/", req, &gateway); err != nil {
		return nil, err
	}
	return &gateway, nil
}

func (s *natGatewayService) Update(ctx context.Context, uuid string, req *UpdateNATGatewayRequest) (*NATGateway, error) {
	var gateway NATGateway
	if err := s.client.patch(ctx, fmt.Sprintf("/nat-gateway/%s", uuid), req, &gateway); err != nil {
		return nil, err
	}
	return &gateway, nil
}

func (s *natGatewayService) Delete(ctx context.Context, uuid string) error {
	return s.client.del(ctx, fmt.Sprintf("/nat-gateway/%s", uuid))
}

func (s *natGatewayService) Resize(ctx context.Context, uuid, planName string) error {
	body := map[string]interface{}{
		"plan_name": planName,
	}
	return s.client.post(ctx, fmt.Sprintf("/nat-gateway/%s/resize", uuid), body, nil)
}

func (s *natGatewayService) MoveToProject(ctx context.Context, uuid string, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/nat-gateway/%s/move-to-project", uuid), body, nil)
}

func (s *natGatewayService) SetProtection(ctx context.Context, uuid string, enabled bool) error {
	body := map[string]interface{}{
		"enabled": enabled,
	}
	return s.client.post(ctx, fmt.Sprintf("/nat-gateway/%s/protection", uuid), body, nil)
}

func (s *natGatewayService) GetMetrics(ctx context.Context, uuid string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := s.client.get(ctx, fmt.Sprintf("/nat-gateway/%s/metrics", uuid), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *natGatewayService) GetBandwidthUsage(ctx context.Context, uuid string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := s.client.get(ctx, fmt.Sprintf("/nat-gateway/%s/bandwidth-usage", uuid), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
