package cubepath

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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
	// ScanZone looks up the domain's current records in public DNS. With autoImport they are
	// added to the zone; without it they are only returned.
	ScanZone(ctx context.Context, zoneUUID string, autoImport bool) (*ZoneScanResponse, error)
	// CreateZoneFromScan creates a zone and imports the records found in public DNS.
	CreateZoneFromScan(ctx context.Context, domain string, projectID int) (*DNSImportResult, error)
	// CreateZoneFromFile creates a zone and imports the records of a BIND zone file (max 1 MB).
	// The zone is created even when the file cannot be parsed.
	CreateZoneFromFile(ctx context.Context, domain string, projectID int, zoneFile []byte) (*DNSImportResult, error)
	// ImportZoneFile imports the records of a BIND zone file into an existing zone. NS and SOA
	// records are skipped; problems with single records are listed in Errors.
	ImportZoneFile(ctx context.Context, zoneUUID string, zoneFile []byte) (*DNSImportResult, error)
	MoveZoneToProject(ctx context.Context, zoneUUID string, projectID int) error
	// ListRegions returns the GeoDNS regions a record can be limited to.
	ListRegions(ctx context.Context) ([]DNSRegion, error)

	// Records
	ListRecords(ctx context.Context, zoneUUID string) ([]DNSRecord, error)
	ListRecordsByType(ctx context.Context, zoneUUID, recordType string) ([]DNSRecord, error)
	CreateRecord(ctx context.Context, zoneUUID string, req *CreateDNSRecordRequest) (*DNSRecord, error)
	UpdateRecord(ctx context.Context, zoneUUID, recordUUID string, req *UpdateDNSRecordRequest) (*DNSRecord, error)
	DeleteRecord(ctx context.Context, zoneUUID, recordUUID string) error

	// SOA
	GetSOA(ctx context.Context, zoneUUID string) (*SOARecord, error)
	UpdateSOA(ctx context.Context, zoneUUID string, req *UpdateSOARequest) (*SOARecord, error)

	// Health checks (Pro and Business DNS plans; billed while enabled). When the target of an
	// A or AAAA record stops answering, the record is left out of DNS answers until it recovers.
	ListHealthChecks(ctx context.Context, zoneUUID string) ([]DNSHealthCheck, error)
	GetHealthCheck(ctx context.Context, zoneUUID, recordUUID string) (*DNSHealthCheck, error)
	// SetHealthCheck creates or replaces the health check of a record. Every field is written,
	// so send the full desired configuration.
	SetHealthCheck(ctx context.Context, zoneUUID, recordUUID string, req *DNSHealthCheckRequest) (*DNSHealthCheck, error)
	DeleteHealthCheck(ctx context.Context, zoneUUID, recordUUID string) error
}

// DNSZone represents a DNS zone.
type DNSZone struct {
	UUID         string   `json:"uuid"`
	Domain       string   `json:"domain"`
	Status       string   `json:"status"`
	RecordsCount int      `json:"records_count"`
	Nameservers  []string `json:"nameservers"`
	ProjectID    int      `json:"project_id"`
	// DNSTier is 1 (Free), 2 (Pro) or 3 (Business).
	DNSTier    int     `json:"dns_tier"`
	VerifiedAt *string `json:"verified_at"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
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
	// Region is the GeoDNS region of the record; nil means global.
	Region    *string `json:"region,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
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

// DNSImportResult is the result of a zone import or scan: the records imported (or found,
// for a scan without import), and the problems with single records.
type DNSImportResult struct {
	Imported int         `json:"imported"`
	Skipped  int         `json:"skipped"`
	Errors   []string    `json:"errors"`
	Records  []DNSRecord `json:"records"`
}

// ZoneScanResponse represents the response from scanning a zone.
type ZoneScanResponse = DNSImportResult

// DNSRegion is a GeoDNS region.
type DNSRegion struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// DNS health check types.
const (
	DNSHealthCheckHTTP  = "http"
	DNSHealthCheckHTTPS = "https"
	DNSHealthCheckTCP   = "tcp"
	DNSHealthCheckPing  = "ping"
)

// DNSHealthCheck is the health check of a record. Target nil means the record's own value is
// probed. LastStatus is "healthy", "unhealthy" or "unknown".
type DNSHealthCheck struct {
	UUID               string  `json:"uuid"`
	RecordUUID         string  `json:"record_uuid"`
	Name               string  `json:"name"`
	CheckType          string  `json:"check_type"`
	Target             *string `json:"target"`
	Port               *int    `json:"port"`
	Path               *string `json:"path"`
	ExpectedStatus     *int    `json:"expected_status"`
	IntervalSecs       int     `json:"interval_secs"`
	TimeoutSecs        int     `json:"timeout_secs"`
	HealthyThreshold   int     `json:"healthy_threshold"`
	UnhealthyThreshold int     `json:"unhealthy_threshold"`
	Enabled            bool    `json:"enabled"`
	LastStatus         string  `json:"last_status"`
	LastCheckAt        *string `json:"last_check_at"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// DNSHealthCheckRequest configures a health check. CheckType is one of the DNSHealthCheck
// constants; "tcp" needs Port; Path and ExpectedStatus apply to "http" and "https". Zero values
// use the API defaults (interval 60s, timeout 5s, thresholds 2 and 3, expected status 200).
// Enabled defaults to true.
type DNSHealthCheckRequest struct {
	Name               string `json:"name"`
	CheckType          string `json:"check_type"`
	Target             string `json:"target,omitempty"`
	Port               int    `json:"port,omitempty"`
	Path               string `json:"path,omitempty"`
	ExpectedStatus     int    `json:"expected_status,omitempty"`
	IntervalSecs       int    `json:"interval_secs,omitempty"`
	TimeoutSecs        int    `json:"timeout_secs,omitempty"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
	Enabled            *bool  `json:"enabled,omitempty"`
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
	// Region limits the record to a GeoDNS region (see ListRegions); empty means global.
	Region string `json:"region,omitempty"`
}

// UpdateDNSRecordRequest represents a request to update a DNS record. Region "global" makes
// the record global again.
type UpdateDNSRecordRequest struct {
	Name     *string `json:"name,omitempty"`
	Content  *string `json:"content,omitempty"`
	TTL      *int    `json:"ttl,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	Weight   *int    `json:"weight,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Comment  *string `json:"comment,omitempty"`
	Region   *string `json:"region,omitempty"`
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

func (s *dnsService) CreateZoneFromScan(ctx context.Context, domain string, projectID int) (*DNSImportResult, error) {
	v := url.Values{}
	v.Set("domain", domain)
	v.Set("project_id", strconv.Itoa(projectID))
	var result DNSImportResult
	if err := s.client.post(ctx, "/dns/zones/scan?"+v.Encode(), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *dnsService) CreateZoneFromFile(ctx context.Context, domain string, projectID int, zoneFile []byte) (*DNSImportResult, error) {
	v := url.Values{}
	v.Set("domain", domain)
	v.Set("project_id", strconv.Itoa(projectID))
	var result DNSImportResult
	if err := s.client.postFile(ctx, "/dns/zones/upload?"+v.Encode(), "file", domain+".zone", zoneFile, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *dnsService) ImportZoneFile(ctx context.Context, zoneUUID string, zoneFile []byte) (*DNSImportResult, error) {
	var result DNSImportResult
	if err := s.client.postFile(ctx, fmt.Sprintf("/dns/zones/%s/import", zoneUUID), "file", "zone.txt", zoneFile, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *dnsService) MoveZoneToProject(ctx context.Context, zoneUUID string, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/dns/zones/%s/move-project", zoneUUID), body, nil)
}

func (s *dnsService) ListRegions(ctx context.Context) ([]DNSRegion, error) {
	var regions []DNSRegion
	if err := s.client.get(ctx, "/dns/regions", &regions); err != nil {
		return nil, err
	}
	return regions, nil
}

func (s *dnsService) ListHealthChecks(ctx context.Context, zoneUUID string) ([]DNSHealthCheck, error) {
	var checks []DNSHealthCheck
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones/%s/health-checks", zoneUUID), &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

func (s *dnsService) GetHealthCheck(ctx context.Context, zoneUUID, recordUUID string) (*DNSHealthCheck, error) {
	var check DNSHealthCheck
	if err := s.client.get(ctx, fmt.Sprintf("/dns/zones/%s/records/%s/health-check", zoneUUID, recordUUID), &check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (s *dnsService) SetHealthCheck(ctx context.Context, zoneUUID, recordUUID string, req *DNSHealthCheckRequest) (*DNSHealthCheck, error) {
	var check DNSHealthCheck
	if err := s.client.put(ctx, fmt.Sprintf("/dns/zones/%s/records/%s/health-check", zoneUUID, recordUUID), req, &check); err != nil {
		return nil, err
	}
	return &check, nil
}

func (s *dnsService) DeleteHealthCheck(ctx context.Context, zoneUUID, recordUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/dns/zones/%s/records/%s/health-check", zoneUUID, recordUUID))
}
