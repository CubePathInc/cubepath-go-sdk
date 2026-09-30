package cubepath

import (
	"context"
	"fmt"
	"net/url"
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
	MoveToProject(ctx context.Context, networkID, projectID int) error

	// BGP peers (Dynamic Routes): eBGP sessions between the network gateway (ASN 64512) and a
	// BGP speaker inside the network.
	ListBGPPeers(ctx context.Context, networkID int) ([]BGPPeer, error)
	// CreateBGPPeer adds a peer. The network needs at least one attached server; at most 8
	// peers per network.
	CreateBGPPeer(ctx context.Context, networkID int, req *CreateBGPPeerRequest) (*BGPPeerCreated, error)
	// UpdateBGPPeer changes the mutable fields of a peer; nil fields are left unchanged. To
	// change the type, target or ASN, delete the peer and create it again.
	UpdateBGPPeer(ctx context.Context, networkID int, peerID string, req *UpdateBGPPeerRequest) error
	DeleteBGPPeer(ctx context.Context, networkID int, peerID string) error
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

// BGP peer types.
const (
	BGPPeerTypeIP        = "ip"
	BGPPeerTypeVPS       = "vps"
	BGPPeerTypeBaremetal = "baremetal"
)

// BGPPeer represents a BGP session of a private network. LastState, PrefixesReceived,
// LastStateAt and ReceivedPrefixes are observed by the platform and empty until then.
type BGPPeer struct {
	ID               string   `json:"id"`
	NetworkID        int      `json:"network_id"`
	PeerType         string   `json:"peer_type"`
	PeerTarget       string   `json:"peer_target"`
	RemoteASN        int64    `json:"remote_asn"`
	MaxPrefix        int      `json:"max_prefix"`
	Description      *string  `json:"description"`
	Enabled          bool     `json:"enabled"`
	CreatedAt        string   `json:"created_at"`
	ResolvedPeerIP   *string  `json:"resolved_peer_ip"`
	LastState        *string  `json:"last_state"`
	PrefixesReceived *int     `json:"prefixes_received"`
	LastStateAt      *string  `json:"last_state_at"`
	ReceivedPrefixes []string `json:"received_prefixes"`
}

// CreateBGPPeerRequest represents a request to create a BGP peer. PeerType is "ip" (PeerTarget
// is an IP inside the network) or "vps"/"baremetal" (PeerTarget is the server id). RemoteASN
// must not be 64512. MaxPrefix defaults to 100 (1-1000).
type CreateBGPPeerRequest struct {
	PeerType    string `json:"peer_type"`
	PeerTarget  string `json:"peer_target"`
	RemoteASN   int64  `json:"remote_asn"`
	MaxPrefix   int    `json:"max_prefix,omitempty"`
	Description string `json:"description,omitempty"`
}

// BGPPeerCreated is the response of a BGP peer creation.
type BGPPeerCreated struct {
	Detail     string `json:"detail"`
	PeerID     string `json:"peer_id"`
	PeerType   string `json:"peer_type"`
	PeerTarget string `json:"peer_target"`
	RemoteASN  int64  `json:"remote_asn"`
}

// UpdateBGPPeerRequest represents a request to update a BGP peer.
type UpdateBGPPeerRequest struct {
	MaxPrefix   *int    `json:"max_prefix,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
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

func (s *networkService) MoveToProject(ctx context.Context, networkID, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/networks/%d/move-project", networkID), body, nil)
}

func (s *networkService) ListBGPPeers(ctx context.Context, networkID int) ([]BGPPeer, error) {
	var peers []BGPPeer
	if err := s.client.get(ctx, fmt.Sprintf("/networks/%d/bgp-peers", networkID), &peers); err != nil {
		return nil, err
	}
	return peers, nil
}

func (s *networkService) CreateBGPPeer(ctx context.Context, networkID int, req *CreateBGPPeerRequest) (*BGPPeerCreated, error) {
	var peer BGPPeerCreated
	if err := s.client.post(ctx, fmt.Sprintf("/networks/%d/bgp-peers", networkID), req, &peer); err != nil {
		return nil, err
	}
	return &peer, nil
}

func (s *networkService) UpdateBGPPeer(ctx context.Context, networkID int, peerID string, req *UpdateBGPPeerRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/networks/%d/bgp-peers/%s", networkID, url.PathEscape(peerID)), req, nil)
}

func (s *networkService) DeleteBGPPeer(ctx context.Context, networkID int, peerID string) error {
	return s.client.del(ctx, fmt.Sprintf("/networks/%d/bgp-peers/%s", networkID, url.PathEscape(peerID)))
}
