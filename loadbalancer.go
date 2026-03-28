package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
)

// LoadBalancerService handles communication with the load balancer related methods of the CubePath API.
type LoadBalancerService interface {
	// Load Balancers
	List(ctx context.Context) ([]LoadBalancer, error)
	Get(ctx context.Context, lbUUID string) (*LoadBalancer, error)
	Create(ctx context.Context, req *CreateLoadBalancerRequest) (*LoadBalancer, error)
	Update(ctx context.Context, lbUUID string, req *UpdateLoadBalancerRequest) (*LoadBalancer, error)
	Delete(ctx context.Context, lbUUID string) error
	Resize(ctx context.Context, lbUUID, planName string) error
	ListPlans(ctx context.Context) ([]LBLocationPlans, error)

	// Listeners
	CreateListener(ctx context.Context, lbUUID string, req *CreateListenerRequest) (*LBListener, error)
	UpdateListener(ctx context.Context, lbUUID, listenerUUID string, req *UpdateListenerRequest) (*LBListener, error)
	DeleteListener(ctx context.Context, lbUUID, listenerUUID string) error

	// Targets
	AddTarget(ctx context.Context, lbUUID, listenerUUID string, req *AddTargetRequest) (*LBTarget, error)
	UpdateTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string, req *UpdateTargetRequest) (*LBTarget, error)
	RemoveTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string) error
	DrainTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string) error

	// Health Checks
	ConfigureHealthCheck(ctx context.Context, lbUUID, listenerUUID string, req *HealthCheckConfig) error
	DeleteHealthCheck(ctx context.Context, lbUUID, listenerUUID string) error
}

// LoadBalancer represents a load balancer.
type LoadBalancer struct {
	UUID           string         `json:"uuid"`
	Name           string         `json:"name"`
	Label          string         `json:"label"`
	Status         string         `json:"status"`
	LocationName   string         `json:"location_name"`
	Plan           *LBPlan        `json:"plan,omitempty"`
	PlanName       string         `json:"plan_name"`
	FloatingIPs    []LBFloatingIP `json:"floating_ips"`
	Listeners      []LBListener   `json:"listeners"`
	ListenersCount int            `json:"listeners_count"`
	ProjectID      int            `json:"project_id"`
	CreatedAt      string         `json:"created_at"`
}

// LBPlan represents a load balancer plan.
type LBPlan struct {
	Name                 string      `json:"name"`
	PricePerHour         json.Number `json:"price_per_hour"`
	PricePerMonth        json.Number `json:"price_per_month"`
	MaxListeners         int         `json:"max_listeners"`
	MaxTargets           int         `json:"max_targets"`
	ConnectionsPerSecond int         `json:"connections_per_second"`
}

// LBFloatingIP represents a floating IP assigned to a load balancer.
type LBFloatingIP struct {
	Address string `json:"address"`
	Type    string `json:"type"`
}

// LBListener represents a load balancer listener.
type LBListener struct {
	UUID           string          `json:"uuid"`
	Name           string          `json:"name"`
	Protocol       string          `json:"protocol"`
	SourcePort     int             `json:"source_port"`
	TargetPort     int             `json:"target_port"`
	Algorithm      string          `json:"algorithm"`
	StickySessions bool            `json:"sticky_sessions"`
	Enabled        bool            `json:"enabled"`
	Targets        []LBTarget      `json:"targets"`
	TargetsCount   int             `json:"targets_count"`
	HealthCheck    json.RawMessage `json:"health_check,omitempty"`
}

// LBTarget represents a load balancer target.
type LBTarget struct {
	UUID         string `json:"uuid"`
	TargetType   string `json:"target_type"`
	TargetUUID   string `json:"target_uuid"`
	TargetName   string `json:"target_name"`
	TargetIP     string `json:"target_ip"`
	Port         int    `json:"port"`
	Weight       int    `json:"weight"`
	Enabled      bool   `json:"enabled"`
	HealthStatus string `json:"health_status"`
}

// LBLocationPlans represents plans available at a location.
type LBLocationPlans struct {
	LocationName string   `json:"location_name"`
	Description  string   `json:"location_description"`
	Plans        []LBPlan `json:"plans"`
}

// HealthCheckConfig represents a health check configuration.
type HealthCheckConfig struct {
	Protocol           string `json:"protocol"`
	Path               string `json:"path"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	HealthyThreshold   int    `json:"healthy_threshold"`
	UnhealthyThreshold int    `json:"unhealthy_threshold"`
	ExpectedCodes      string `json:"expected_codes"`
}

// CreateLoadBalancerRequest represents a request to create a load balancer.
type CreateLoadBalancerRequest struct {
	Name         string `json:"name"`
	PlanName     string `json:"plan_name"`
	LocationName string `json:"location_name"`
	ProjectID    *int   `json:"project_id,omitempty"`
	Label        string `json:"label,omitempty"`
}

// UpdateLoadBalancerRequest represents a request to update a load balancer.
type UpdateLoadBalancerRequest struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

// CreateListenerRequest represents a request to create a listener.
type CreateListenerRequest struct {
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	SourcePort     int    `json:"source_port"`
	TargetPort     int    `json:"target_port"`
	Algorithm      string `json:"algorithm"`
	StickySessions bool   `json:"sticky_sessions"`
}

// UpdateListenerRequest represents a request to update a listener.
type UpdateListenerRequest struct {
	Name       *string `json:"name,omitempty"`
	TargetPort *int    `json:"target_port,omitempty"`
	Algorithm  *string `json:"algorithm,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

// AddTargetRequest represents a request to add a target.
type AddTargetRequest struct {
	TargetType string `json:"target_type"`
	TargetUUID string `json:"target_uuid"`
	Port       *int   `json:"port,omitempty"`
	Weight     int    `json:"weight"`
}

// UpdateTargetRequest represents a request to update a target.
type UpdateTargetRequest struct {
	Port    *int  `json:"port,omitempty"`
	Weight  *int  `json:"weight,omitempty"`
	Enabled *bool `json:"enabled,omitempty"`
}

type loadBalancerService struct {
	client *Client
}

func (s *loadBalancerService) List(ctx context.Context) ([]LoadBalancer, error) {
	var lbs []LoadBalancer
	if err := s.client.get(ctx, "/loadbalancer/", &lbs); err != nil {
		return nil, err
	}
	return lbs, nil
}

func (s *loadBalancerService) Get(ctx context.Context, lbUUID string) (*LoadBalancer, error) {
	var lb LoadBalancer
	if err := s.client.get(ctx, fmt.Sprintf("/loadbalancer/%s", lbUUID), &lb); err != nil {
		return nil, err
	}
	return &lb, nil
}

func (s *loadBalancerService) Create(ctx context.Context, req *CreateLoadBalancerRequest) (*LoadBalancer, error) {
	var lb LoadBalancer
	if err := s.client.post(ctx, "/loadbalancer/", req, &lb); err != nil {
		return nil, err
	}
	return &lb, nil
}

func (s *loadBalancerService) Update(ctx context.Context, lbUUID string, req *UpdateLoadBalancerRequest) (*LoadBalancer, error) {
	var lb LoadBalancer
	if err := s.client.patch(ctx, fmt.Sprintf("/loadbalancer/%s", lbUUID), req, &lb); err != nil {
		return nil, err
	}
	return &lb, nil
}

func (s *loadBalancerService) Delete(ctx context.Context, lbUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/loadbalancer/%s", lbUUID))
}

func (s *loadBalancerService) Resize(ctx context.Context, lbUUID, planName string) error {
	body := map[string]interface{}{
		"plan_name": planName,
	}
	return s.client.post(ctx, fmt.Sprintf("/loadbalancer/%s/resize", lbUUID), body, nil)
}

func (s *loadBalancerService) ListPlans(ctx context.Context) ([]LBLocationPlans, error) {
	var plans []LBLocationPlans
	if err := s.client.get(ctx, "/loadbalancer/plans", &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *loadBalancerService) CreateListener(ctx context.Context, lbUUID string, req *CreateListenerRequest) (*LBListener, error) {
	var listener LBListener
	if err := s.client.post(ctx, fmt.Sprintf("/loadbalancer/%s/listeners", lbUUID), req, &listener); err != nil {
		return nil, err
	}
	return &listener, nil
}

func (s *loadBalancerService) UpdateListener(ctx context.Context, lbUUID, listenerUUID string, req *UpdateListenerRequest) (*LBListener, error) {
	var listener LBListener
	if err := s.client.patch(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s", lbUUID, listenerUUID), req, &listener); err != nil {
		return nil, err
	}
	return &listener, nil
}

func (s *loadBalancerService) DeleteListener(ctx context.Context, lbUUID, listenerUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s", lbUUID, listenerUUID))
}

func (s *loadBalancerService) AddTarget(ctx context.Context, lbUUID, listenerUUID string, req *AddTargetRequest) (*LBTarget, error) {
	var target LBTarget
	if err := s.client.post(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/targets", lbUUID, listenerUUID), req, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

func (s *loadBalancerService) UpdateTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string, req *UpdateTargetRequest) (*LBTarget, error) {
	var target LBTarget
	if err := s.client.patch(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/targets/%s", lbUUID, listenerUUID, targetUUID), req, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

func (s *loadBalancerService) RemoveTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/targets/%s", lbUUID, listenerUUID, targetUUID))
}

func (s *loadBalancerService) DrainTarget(ctx context.Context, lbUUID, listenerUUID, targetUUID string) error {
	return s.client.post(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/targets/%s/drain", lbUUID, listenerUUID, targetUUID), nil, nil)
}

func (s *loadBalancerService) ConfigureHealthCheck(ctx context.Context, lbUUID, listenerUUID string, req *HealthCheckConfig) error {
	return s.client.put(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/health-check", lbUUID, listenerUUID), req, nil)
}

func (s *loadBalancerService) DeleteHealthCheck(ctx context.Context, lbUUID, listenerUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/loadbalancer/%s/listeners/%s/health-check", lbUUID, listenerUUID))
}
