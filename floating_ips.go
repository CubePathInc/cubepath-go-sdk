package cubepath

import (
	"context"
	"fmt"
)

// FloatingIPService handles communication with the floating IP related methods of the CubePath API.
type FloatingIPService interface {
	List(ctx context.Context) (*FloatingIPsResponse, error)
	Acquire(ctx context.Context, ipType, locationName string) (*FloatingIP, error)
	Release(ctx context.Context, address string) error
	Assign(ctx context.Context, resourceType string, resourceID int, address string) error
	Unassign(ctx context.Context, address string) error
	ConfigureReverseDNS(ctx context.Context, ip, reverseDNS string) error
}

// FloatingIP represents a floating IP address.
type FloatingIP struct {
	ID             int    `json:"id"`
	Address        string `json:"address"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	IsPrimary      bool   `json:"is_primary"`
	LocationName   string `json:"location_name"`
	ProtectionType string `json:"protection_type"`
	VPSName        string `json:"vps_name"`
	BaremetalName  string `json:"baremetal_name"`
}

// FloatingIPsResponse represents the response from listing floating IPs.
type FloatingIPsResponse struct {
	SingleIPs []FloatingIP `json:"single_ips"`
	Subnets   []Subnet     `json:"subnets"`
}

// Subnet represents a subnet of floating IPs.
type Subnet struct {
	Prefix         int          `json:"prefix"`
	ProtectionType string       `json:"protection_type"`
	IPAddresses    []FloatingIP `json:"ip_addresses"`
}

type floatingIPService struct {
	client *Client
}

func (s *floatingIPService) List(ctx context.Context) (*FloatingIPsResponse, error) {
	var result FloatingIPsResponse
	if err := s.client.get(ctx, "/floating_ips/organization", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *floatingIPService) Acquire(ctx context.Context, ipType, locationName string) (*FloatingIP, error) {
	var result FloatingIP
	path := fmt.Sprintf("/floating_ips/acquire?ip_type=%s&location_name=%s", ipType, locationName)
	if err := s.client.post(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *floatingIPService) Release(ctx context.Context, address string) error {
	return s.client.post(ctx, fmt.Sprintf("/floating_ips/release/%s", address), nil, nil)
}

func (s *floatingIPService) Assign(ctx context.Context, resourceType string, resourceID int, address string) error {
	path := fmt.Sprintf("/floating_ips/assign/%s/%d?address=%s", resourceType, resourceID, address)
	return s.client.post(ctx, path, nil, nil)
}

func (s *floatingIPService) Unassign(ctx context.Context, address string) error {
	return s.client.post(ctx, fmt.Sprintf("/floating_ips/unassign/%s", address), nil, nil)
}

func (s *floatingIPService) ConfigureReverseDNS(ctx context.Context, ip, reverseDNS string) error {
	path := fmt.Sprintf("/floating_ips/reverse_dns/configure?ip=%s&reverse_dns=%s", ip, reverseDNS)
	return s.client.post(ctx, path, nil, nil)
}
