package cubepath

import "context"

// PricingService handles communication with the pricing related methods of the CubePath API.
type PricingService interface {
	Get(ctx context.Context) (*PricingResponse, error)
}

// PricingResponse represents the response from the pricing endpoint.
type PricingResponse struct {
	VPS       VPSPricing       `json:"vps"`
	Baremetal BaremetalPricing `json:"baremetal,omitempty"`
}

// VPSPricing represents VPS pricing information.
type VPSPricing struct {
	Locations []LocationPricing `json:"locations"`
	Templates []VPSTemplate     `json:"templates"`
}

// LocationPricing represents pricing for a specific location.
type LocationPricing struct {
	LocationName string    `json:"location_name"`
	Description  string    `json:"description"`
	Clusters     []Cluster `json:"clusters"`
}

// Cluster represents a cluster in a location with its available plans.
type Cluster struct {
	ClusterName string    `json:"cluster_name"`
	Plans       []VPSPlan `json:"plans"`
}

// BaremetalPricing represents baremetal pricing information.
type BaremetalPricing struct {
	Locations []BaremetalLocationPricing `json:"locations"`
}

// BaremetalLocationPricing represents baremetal pricing for a specific location.
type BaremetalLocationPricing struct {
	LocationName    string                `json:"location_name"`
	Description     string                `json:"description"`
	BaremetalModels []BaremetalModelPrice `json:"baremetal_models"`
}

// BaremetalModelPrice represents pricing for a baremetal model.
type BaremetalModelPrice struct {
	ModelName      string  `json:"model_name"`
	CPU            string  `json:"cpu"`
	CPUSpecs       string  `json:"cpu_specs"`
	RAMSize        int     `json:"ram_size"`
	RAMType        string  `json:"ram_type"`
	DiskSize       string  `json:"disk_size"`
	DiskType       string  `json:"disk_type"`
	Port           int     `json:"port"`
	Price          float64 `json:"price"`
	Setup          float64 `json:"setup"`
	StockAvailable bool    `json:"stock_available"`
}

type pricingService struct {
	client *Client
}

func (s *pricingService) Get(ctx context.Context) (*PricingResponse, error) {
	var pricing PricingResponse
	if err := s.client.get(ctx, "/pricing", &pricing); err != nil {
		return nil, err
	}
	return &pricing, nil
}
