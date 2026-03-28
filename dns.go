package cubepath

import (
	"context"
	"fmt"
)

// DNSService handles communication with the DNS related methods of the CubePath API.
type DNSService interface {
	// Zones
	ListZones(ctx context.Context) ([]DNSZone, error)
	ListZonesByProject(ctx context.Context, projectID int) ([]DNSZone, error)
	GetZone(ctx context.Context, zoneUUID string) (*DNSZone, error)
	CreateZone(ctx context.Context, req *CreateDNSZoneRequest) (*DNSZone, error)
	DeleteZone(ctx context.Context, zoneUUID string) error
	VerifyZone(ctx context.Context, zoneUUID string) (*ZoneVerifyResponse, error)
	ScanZone(ctx context.Context, zoneUUID string, autoImport bool) (*ZoneScanResponse, error)

	// Records
	ListRecords(ctx context.Context, zoneUUID string) ([]DNSRecord, error)
	ListRecordsByType(ctx context.Context, zoneUUID, recordType string) ([]DNSRecord, error)
	CreateRecord(ctx context.Context, zoneUUID string, req *CreateDNSRecordRequest) (*DNSRecord, error)
	UpdateRecord(ctx context.Context, zoneUUID, recordUUID string, req *UpdateDNSRecordRequest) (*DNSRecord, error)
	DeleteRecord(ctx context.Context, zoneUUID, recordUUID string) error

	// SOA
	GetSOA(ctx context.Context, zoneUUID string) (*SOARecord, error)
	UpdateSOA(ctx context.Context, zoneUUID string, req *UpdateSOARequest) (*SOARecord, error)
}

// DNSZone represents a DNS zone.
type DNSZone struct {
	UUID         string   `json:"uuid"`
	Domain       string   `json:"domain"`
	Status       string   `json:"status"`
	RecordsCount int      `json:"records_count"`
	Nameservers  []string `json:"nameservers"`
	ProjectID    int      `json:"project_id"`
	CreatedAt    string   `json:"created_at"`
}

// DNSRecord represents a DNS record.
type DNSRecord struct {
	UUID       string `json:"uuid"`
	ZoneUUID   string `json:"zone_uuid"`
	Name       string `json:"name"`
	RecordType string `json:"record_type"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	TTL        int    `json:"ttl"`
	Priority   *int   `json:"priority,omitempty"`
	Weight     *int   `json:"weight,omitempty"`
	Port       *int   `json:"port,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

// SOARecord represents a DNS SOA record.
type SOARecord struct {
	PrimaryNS  string `json:"primary_ns"`
	Hostmaster string `json:"hostmaster"`
	Serial     int64  `json:"serial"`
	Refresh    int    `json:"refresh"`
	Retry      int    `json:"retry"`
	Expire     int    `json:"expire"`
	Minimum    int    `json:"minimum"`
}

// ZoneVerifyResponse represents the response from verifying a zone.
type ZoneVerifyResponse struct {
	Verified    bool   `json:"verified"`
	Message     string `json:"message"`
	NextCheckAt string `json:"next_check_at"`
}

// ZoneScanResponse represents the response from scanning a zone.
type ZoneScanResponse struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Records []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
	} `json:"records"`
}

// CreateDNSZoneRequest represents a request to create a DNS zone.
type CreateDNSZoneRequest struct {
	Domain    string `json:"domain"`
	ProjectID *int   `json:"project_id,omitempty"`
}

// CreateDNSRecordRequest represents a request to create a DNS record.
type CreateDNSRecordRequest struct {
	Name     string `json:"name"`
	Type     string `json:"record_type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
	Weight   *int   `json:"weight,omitempty"`
	Port     *int   `json:"port,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// UpdateDNSRecordRequest represents a request to update a DNS record.
type UpdateDNSRecordRequest struct {
	Name     *string `json:"name,omitempty"`
	Content  *string `json:"content,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
	Priority *int    `json:"priority,omitempty"`
}

// UpdateSOARequest represents a request to update a SOA record.
type UpdateSOARequest struct {
	Refresh    *int    `json:"refresh,omitempty"`
	Retry      *int    `json:"retry,omitempty"`
	Expire     *int    `json:"expire,omitempty"`
	Minimum    *int    `json:"minimum,omitempty"`
	Hostmaster *string `json:"hostmaster,omitempty"`
}

type dnsService struct {
	client *Client
}

func (s *dnsService) ListZones(ctx context.Context) ([]DNSZone, error) {
	var zones []DNSZone
	if err := s.client.get(ctx, "/dns/zones", &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (s *dnsService) ListZonesByProject(ctx context.Context, projectID int) ([]DNSZone, error) {
	var zones []DNSZone
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones?project_id=%d", projectID), &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (s *dnsService) GetZone(ctx context.Context, zoneUUID string) (*DNSZone, error) {
	var zone DNSZone
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones/%s", zoneUUID), &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

func (s *dnsService) CreateZone(ctx context.Context, req *CreateDNSZoneRequest) (*DNSZone, error) {
	var zone DNSZone
	if err := s.client.post(ctx, "/dns/zones", req, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

func (s *dnsService) DeleteZone(ctx context.Context, zoneUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/dns/zones/%s", zoneUUID))
}

func (s *dnsService) VerifyZone(ctx context.Context, zoneUUID string) (*ZoneVerifyResponse, error) {
	var result ZoneVerifyResponse
	if err := s.client.post(ctx, fmt.Sprintf("/dns/zones/%s/verify", zoneUUID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *dnsService) ScanZone(ctx context.Context, zoneUUID string, autoImport bool) (*ZoneScanResponse, error) {
	var result ZoneScanResponse
	path := fmt.Sprintf("/dns/zones/%s/scan?auto_import=%t", zoneUUID, autoImport)
	if err := s.client.post(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *dnsService) ListRecords(ctx context.Context, zoneUUID string) ([]DNSRecord, error) {
	var records []DNSRecord
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones/%s/records", zoneUUID), &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *dnsService) ListRecordsByType(ctx context.Context, zoneUUID, recordType string) ([]DNSRecord, error) {
	var records []DNSRecord
	path := fmt.Sprintf("/dns/zones/%s/records?record_type=%s", zoneUUID, recordType)
	if err := s.client.get(ctx, path, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *dnsService) CreateRecord(ctx context.Context, zoneUUID string, req *CreateDNSRecordRequest) (*DNSRecord, error) {
	var record DNSRecord
	if err := s.client.post(ctx, fmt.Sprintf("/dns/zones/%s/records", zoneUUID), req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *dnsService) UpdateRecord(ctx context.Context, zoneUUID, recordUUID string, req *UpdateDNSRecordRequest) (*DNSRecord, error) {
	var record DNSRecord
	if err := s.client.put(ctx, fmt.Sprintf("/dns/zones/%s/records/%s", zoneUUID, recordUUID), req, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *dnsService) DeleteRecord(ctx context.Context, zoneUUID, recordUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/dns/zones/%s/records/%s", zoneUUID, recordUUID))
}

func (s *dnsService) GetSOA(ctx context.Context, zoneUUID string) (*SOARecord, error) {
	var soa SOARecord
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones/%s/soa", zoneUUID), &soa); err != nil {
		return nil, err
	}
	return &soa, nil
}

func (s *dnsService) UpdateSOA(ctx context.Context, zoneUUID string, req *UpdateSOARequest) (*SOARecord, error) {
	var soa SOARecord
	if err := s.client.put(ctx, fmt.Sprintf("/dns/zones/%s/soa", zoneUUID), req, &soa); err != nil {
		return nil, err
	}
	return &soa, nil
}
