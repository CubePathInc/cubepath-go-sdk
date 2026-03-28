package cubepath

import (
	"context"
	"fmt"
)

// VPSISOService handles communication with the VPS ISO related methods of the CubePath API.
type VPSISOService interface {
	List(ctx context.Context, vpsID int) (*ISOListResponse, error)
	Mount(ctx context.Context, vpsID int, isoID string) error
	Unmount(ctx context.Context, vpsID int) error
}

// ISO represents an ISO image.
type ISO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FileSize  int    `json:"file_size"`
	IsMounted bool   `json:"is_mounted"`
}

// ISOListResponse represents the response from listing ISOs.
type ISOListResponse struct {
	Items        []ISO  `json:"items"`
	MountedISOID string `json:"mounted_iso_id"`
}

type vpsISOService struct {
	client *Client
}

func (s *vpsISOService) List(ctx context.Context, vpsID int) (*ISOListResponse, error) {
	var result ISOListResponse
	if err := s.client.get(ctx, fmt.Sprintf("/vps/%d/isos", vpsID), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *vpsISOService) Mount(ctx context.Context, vpsID int, isoID string) error {
	body := map[string]interface{}{
		"iso_id": isoID,
	}
	return s.client.post(ctx, fmt.Sprintf("/vps/%d/iso", vpsID), body, nil)
}

func (s *vpsISOService) Unmount(ctx context.Context, vpsID int) error {
	return s.client.del(ctx, fmt.Sprintf("/vps/%d/iso", vpsID))
}
