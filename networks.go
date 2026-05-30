package cubepath

import (
	"context"
	"fmt"
	"time"
)

// NetworkService handles communication with the network related methods of the CubePath API.
type NetworkService interface {
	Create(ctx context.Context, req *CreateNetworkRequest) (*Network, error)
	List(ctx context.Context) ([]ProjectResponse, error)
	Update(ctx context.Context, networkID int, req *UpdateNetworkRequest) error
	Delete(ctx context.Context, networkID int) error
	ListRoutes(ctx context.Context, networkID int) ([]NetworkRoute, error)
	CreateRoute(ctx context.Context, networkID int, req *CreateNetworkRouteRequest) (*NetworkRoute, error)
	DeleteRoute(ctx context.Context, networkID int, routeID string) error
}

// Network represents a private network.
type Network struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Label        string    `json:"label"`
	ProjectID    int       `json:"project_id"`
	LocationName string    `json:"location_name"`
	IPRange      string    `json:"ip_range"`
	Prefix       int       `json:"prefix"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateNetworkRequest represents a request to create a network.
type CreateNetworkRequest struct {
	Name         string `json:"name"`
	LocationName string `json:"location_name"`
	IPRange      string `json:"ip_range"`
	Prefix       int    `json:"prefix"`
	ProjectID    int    `json:"project_id"`
	Label        string `json:"label,omitempty"`
}

// UpdateNetworkRequest represents a request to update a network.
type UpdateNetworkRequest struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

// NetworkRoute represents a route in a private network.
type NetworkRoute struct {
	ID                string `json:"id"`
	NetworkID         int    `json:"network_id"`
	Destination       string `json:"destination"`
	NextHopType       string `json:"next_hop_type"`
	NextHopTarget     string `json:"next_hop_target"`
	ResolvedNextHopIP string `json:"resolved_next_hop_ip"`
	Description       string `json:"description"`
	CreatedAt         string `json:"created_at"`
	NATGatewayUUID    string `json:"nat_gateway_uuid"`
	NATGatewayName    string `json:"nat_gateway_name"`
}

// CreateNetworkRouteRequest represents a request to create a network route.
type CreateNetworkRouteRequest struct {
	Destination   string `json:"destination"`
	NextHopType   string `json:"next_hop_type"`
	NextHopTarget string `json:"next_hop_target"`
	Description   string `json:"description,omitempty"`
}

type networkService struct {
	client *Client
}

func (s *networkService) Create(ctx context.Context, req *CreateNetworkRequest) (*Network, error) {
	var network Network
	if err := s.client.post(ctx, "/networks/create_network", req, &network); err != nil {
		return nil, err
	}
	return &network, nil
}

func (s *networkService) List(ctx context.Context) ([]ProjectResponse, error) {
	var projects []ProjectResponse
	if err := s.client.get(ctx, "/projects/", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *networkService) Update(ctx context.Context, networkID int, req *UpdateNetworkRequest) error {
	return s.client.put(ctx, fmt.Sprintf("/networks/%d", networkID), req, nil)
}

func (s *networkService) Delete(ctx context.Context, networkID int) error {
	return s.client.del(ctx, fmt.Sprintf("/networks/%d", networkID))
}

func (s *networkService) ListRoutes(ctx context.Context, networkID int) ([]NetworkRoute, error) {
	var routes []NetworkRoute
	if err := s.client.get(ctx, fmt.Sprintf("/networks/%d/routes", networkID), &routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (s *networkService) CreateRoute(ctx context.Context, networkID int, req *CreateNetworkRouteRequest) (*NetworkRoute, error) {
	var route NetworkRoute
	if err := s.client.post(ctx, fmt.Sprintf("/networks/%d/routes", networkID), req, &route); err != nil {
		return nil, err
	}
	return &route, nil
}

func (s *networkService) DeleteRoute(ctx context.Context, networkID int, routeID string) error {
	return s.client.del(ctx, fmt.Sprintf("/networks/%d/routes/%s", networkID, routeID))
}
