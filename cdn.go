package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// CDNService handles communication with the CDN related methods of the CubePath API.
type CDNService interface {
	// Zones
	ListZones(ctx context.Context) ([]CDNZone, error)
	GetZone(ctx context.Context, zoneUUID string) (*CDNZone, error)
	CreateZone(ctx context.Context, req *CreateCDNZoneRequest) (*CDNZone, error)
	UpdateZone(ctx context.Context, zoneUUID string, req *UpdateCDNZoneRequest) (*CDNZone, error)
	DeleteZone(ctx context.Context, zoneUUID string) error
	GetZonePricing(ctx context.Context, zoneUUID string) (json.RawMessage, error)
	ListPlans(ctx context.Context) ([]CDNPlan, error)

	// Origins
	ListOrigins(ctx context.Context, zoneUUID string) ([]CDNOrigin, error)
	CreateOrigin(ctx context.Context, zoneUUID string, req *CreateCDNOriginRequest) (*CDNOrigin, error)
	UpdateOrigin(ctx context.Context, zoneUUID, originUUID string, req *UpdateCDNOriginRequest) (*CDNOrigin, error)
	DeleteOrigin(ctx context.Context, zoneUUID, originUUID string) error

	// Rules
	ListRules(ctx context.Context, zoneUUID string) ([]CDNRule, error)
	GetRule(ctx context.Context, zoneUUID, ruleUUID string) (*CDNRule, error)
	CreateRule(ctx context.Context, zoneUUID string, req *CreateCDNRuleRequest) (*CDNRule, error)
	UpdateRule(ctx context.Context, zoneUUID, ruleUUID string, req *UpdateCDNRuleRequest) (*CDNRule, error)
	DeleteRule(ctx context.Context, zoneUUID, ruleUUID string) error

	// WAF Rules
	ListWAFRules(ctx context.Context, zoneUUID string) ([]CDNRule, error)
	GetWAFRule(ctx context.Context, zoneUUID, ruleUUID string) (*CDNRule, error)
	CreateWAFRule(ctx context.Context, zoneUUID string, req *CreateCDNRuleRequest) (*CDNRule, error)
	UpdateWAFRule(ctx context.Context, zoneUUID, ruleUUID string, req *UpdateCDNRuleRequest) (*CDNRule, error)
	DeleteWAFRule(ctx context.Context, zoneUUID, ruleUUID string) error

	// Metrics
	// GetMetrics returns one analytics report of the zone as raw JSON. metricType is one of
	// the CDNMetric constants.
	GetMetrics(ctx context.Context, zoneUUID string, metricType string, params *CDNMetricsParams) (json.RawMessage, error)

	// Actions
	RequestSSL(ctx context.Context, zoneUUID string) error
	MoveZoneToProject(ctx context.Context, zoneUUID string, projectID int) error

	// Cache purge
	// PurgeCache queues a purge on every edge location. Set Everything, or list up to 100
	// Paths (a path ending in * purges the prefix). An identical purge that has not started yet
	// is reused.
	PurgeCache(ctx context.Context, zoneUUID string, req *CDNPurgeRequest) (*CDNPurge, error)
	// ListPurges returns the latest 20 purges of the zone with their progress per location.
	ListPurges(ctx context.Context, zoneUUID string) ([]CDNPurgeStatus, error)

	// Token Auth
	// RotateTokenSecret generates a new Token Auth secret (and enables Token Auth). Every URL
	// signed with the old secret stops working.
	RotateTokenSecret(ctx context.Context, zoneUUID string) (string, error)
	// SignURL returns a signed URL for a zone with Token Auth enabled.
	SignURL(ctx context.Context, zoneUUID string, req *CDNSignURLRequest) (*CDNSignedURL, error)
}

// CDN analytics reports, for GetMetrics.
const (
	CDNMetricSummary        = "summary"
	CDNMetricRequests       = "requests"
	CDNMetricBandwidth      = "bandwidth"
	CDNMetricCache          = "cache"
	CDNMetricStatusCodes    = "status-codes"
	CDNMetricTopURLs        = "top-urls"
	CDNMetricTopCountries   = "top-countries"
	CDNMetricTopASN         = "top-asn"
	CDNMetricTopUserAgents  = "top-user-agents"
	CDNMetricBlocked        = "blocked"
	CDNMetricPOPs           = "pops"
	CDNMetricFileExtensions = "file-extensions"
)

// CDNZone represents a CDN zone.
type CDNZone struct {
	UUID         string      `json:"uuid"`
	Name         string      `json:"name"`
	Domain       string      `json:"domain"`
	CustomDomain string      `json:"custom_domain"`
	Status       string      `json:"status"`
	PlanName     string      `json:"plan_name"`
	SSLType      string      `json:"ssl_type"`
	ProjectID    int         `json:"project_id"`
	Origins      []CDNOrigin `json:"origins"`
	Rules        []CDNRule   `json:"rules"`
	// TokenAuthSecret is only returned to callers that can change the zone, and by
	// UpdateZone when Token Auth is enabled for the first time.
	TokenAuthEnabled   bool    `json:"token_auth_enabled"`
	TokenAuthIPBinding bool    `json:"token_auth_ip_binding"`
	TokenAuthSecret    *string `json:"token_auth_secret,omitempty"`
	CORSEnabled        bool    `json:"cors_enabled"`
	CORSAllowOrigins   *string `json:"cors_allow_origins,omitempty"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// CDNOrigin represents a CDN origin server.
type CDNOrigin struct {
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	Address            string `json:"address"`
	Port               int    `json:"port"`
	Protocol           string `json:"protocol"`
	Weight             int    `json:"weight"`
	Priority           int    `json:"priority"`
	IsBackup           bool   `json:"is_backup"`
	HealthCheckEnabled bool   `json:"health_check_enabled"`
	HealthCheckPath    string `json:"health_check_path"`
	HealthStatus       string `json:"health_status"`
	VerifySSL          bool   `json:"verify_ssl"`
	HostHeader         string `json:"host_header"`
	BasePath           string `json:"base_path"`
	Enabled            bool   `json:"enabled"`
	// ObjectStorageBucketUUID is set when the origin serves a CubePath Object Storage bucket.
	ObjectStorageBucketUUID *string `json:"object_storage_bucket_uuid,omitempty"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

// CDNRule represents a CDN edge rule or WAF rule.
type CDNRule struct {
	UUID            string          `json:"uuid"`
	Name            string          `json:"name"`
	RuleType        string          `json:"rule_type"`
	Priority        int             `json:"priority"`
	MatchConditions json.RawMessage `json:"match_conditions"`
	ActionConfig    json.RawMessage `json:"action_config"`
	Enabled         bool            `json:"enabled"`
	ExpiresAt       string          `json:"expires_at"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

// CDNPlan represents a CDN plan.
type CDNPlan struct {
	UUID              string          `json:"uuid"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	PricePerGB        json.RawMessage `json:"price_per_gb"`
	BasePricePerHour  float64         `json:"base_price_per_hour"`
	MaxZones          int             `json:"max_zones"`
	MaxOriginsPerZone int             `json:"max_origins_per_zone"`
	MaxRulesPerZone   int             `json:"max_rules_per_zone"`
	CustomSSLAllowed  bool            `json:"custom_ssl_allowed"`
}

// CDNMetricsParams represents query parameters for CDN metrics. Minutes is the look-back
// window (default 60), IntervalSeconds the bucket size of time series and Limit the size of
// top lists. The filters are optional: Country and ASN are comma separated lists, Status a
// comma separated list of status codes (it wins over StatusRange: "2xx" to "5xx"),
// CacheStatus "HIT" or "MISS", DeviceType "mobile", "desktop" or "bot". GroupBy is not
// supported by the API and is ignored.
type CDNMetricsParams struct {
	Minutes         int    `json:"minutes,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	GroupBy         string `json:"group_by,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	Country         string `json:"country,omitempty"`
	ASN             string `json:"asn,omitempty"`
	Status          string `json:"status,omitempty"`
	StatusRange     string `json:"status_range,omitempty"`
	CacheStatus     string `json:"cache_status,omitempty"`
	DeviceType      string `json:"device_type,omitempty"`
	PathPrefix      string `json:"path_prefix,omitempty"`
}

// CDNPurgeRequest selects what to purge: Everything, or a list of Paths.
type CDNPurgeRequest struct {
	Everything bool     `json:"everything,omitempty"`
	Paths      []string `json:"paths,omitempty"`
}

// CDNPurge is the response of a purge request.
type CDNPurge struct {
	Detail    string `json:"detail"`
	PurgeUUID string `json:"purge_uuid"`
	Status    string `json:"status"`
}

// CDNPurgeProgress counts the edge nodes of a purge.
type CDNPurgeProgress struct {
	Expected  int `json:"expected"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// CDNPurgePOP is the progress of a purge in one location.
type CDNPurgePOP struct {
	POP       string `json:"pop"`
	Expected  int    `json:"expected"`
	Completed int    `json:"completed"`
	Failed    int    `json:"failed"`
}

// CDNPurgeStatus is a purge with its progress. Status is "pending" or "in_progress", or one of
// the final states "completed", "partial", "failed" and "expired".
type CDNPurgeStatus struct {
	PurgeUUID   string           `json:"purge_uuid"`
	Scope       string           `json:"scope"`
	Paths       []string         `json:"paths"`
	Status      string           `json:"status"`
	RequestedAt string           `json:"requested_at"`
	CompletedAt *string          `json:"completed_at"`
	Nodes       CDNPurgeProgress `json:"nodes"`
	POPs        []CDNPurgePOP    `json:"pops"`
}

// CDNSignURLRequest represents a request to sign a URL. Path is a path or full URL (only the
// path is signed). ExpiresIn is 60-604800 seconds (default 3600). ClientIP is required when
// the zone binds tokens to the client IP.
type CDNSignURLRequest struct {
	Path      string `json:"path"`
	ExpiresIn int    `json:"expires_in,omitempty"`
	ClientIP  string `json:"client_ip,omitempty"`
}

// CDNSignedURL is a signed URL. Expires is a Unix time.
type CDNSignedURL struct {
	Detail    string `json:"detail"`
	SignedURL string `json:"signed_url"`
	Token     string `json:"token"`
	Expires   int64  `json:"expires"`
}

// CreateCDNZoneRequest represents a request to create a CDN zone.
type CreateCDNZoneRequest struct {
	Name         string `json:"name"`
	PlanName     string `json:"plan_name"`
	CustomDomain string `json:"custom_domain,omitempty"`
	ProjectID    *int   `json:"project_id,omitempty"`
}

// UpdateCDNZoneRequest represents a request to update a CDN zone.
//
// Enabling Token Auth for the first time generates a secret, returned once in
// CDNZone.TokenAuthSecret. CORSAllowOrigins is "*" or a comma separated list of origins.
type UpdateCDNZoneRequest struct {
	Name               *string `json:"name,omitempty"`
	CustomDomain       *string `json:"custom_domain,omitempty"`
	SSLType            *string `json:"ssl_type,omitempty"`
	CertificateUUID    *string `json:"certificate_uuid,omitempty"`
	TokenAuthEnabled   *bool   `json:"token_auth_enabled,omitempty"`
	TokenAuthIPBinding *bool   `json:"token_auth_ip_binding,omitempty"`
	CORSEnabled        *bool   `json:"cors_enabled,omitempty"`
	CORSAllowOrigins   *string `json:"cors_allow_origins,omitempty"`
}

// CreateCDNOriginRequest represents a request to create a CDN origin.
//
// To serve a CubePath Object Storage bucket, set ObjectStorageBucketUUID and only Name,
// Weight, Priority and IsBackup: the API fills the address, TLS, health check and read only
// credentials of the bucket, and refuses any other field. The request is serialized that way
// automatically when ObjectStorageBucketUUID is set.
type CreateCDNOriginRequest struct {
	Name               string `json:"name"`
	OriginURL          string `json:"origin_url,omitempty"`
	Address            string `json:"address,omitempty"`
	Port               *int   `json:"port,omitempty"`
	Protocol           string `json:"protocol,omitempty"`
	Weight             int    `json:"weight"`
	Priority           int    `json:"priority"`
	IsBackup           bool   `json:"is_backup"`
	HealthCheckEnabled bool   `json:"health_check_enabled"`
	HealthCheckPath    string `json:"health_check_path"`
	VerifySSL          bool   `json:"verify_ssl"`
	HostHeader         string `json:"host_header,omitempty"`
	BasePath           string `json:"base_path,omitempty"`
	Enabled            bool   `json:"enabled"`
	// ObjectStorageBucketUUID makes the origin serve a CubePath Object Storage bucket.
	ObjectStorageBucketUUID string `json:"object_storage_bucket_uuid,omitempty"`
}

// MarshalJSON sends only the fields the API accepts next to object_storage_bucket_uuid when the
// origin is a bucket origin, and the regular fields otherwise.
func (r CreateCDNOriginRequest) MarshalJSON() ([]byte, error) {
	type plain CreateCDNOriginRequest
	if r.ObjectStorageBucketUUID == "" {
		return json.Marshal(plain(r))
	}
	body := map[string]interface{}{
		"name":                       r.Name,
		"object_storage_bucket_uuid": r.ObjectStorageBucketUUID,
		"is_backup":                  r.IsBackup,
	}
	if r.Weight > 0 {
		body["weight"] = r.Weight
	}
	if r.Priority > 0 {
		body["priority"] = r.Priority
	}
	return json.Marshal(body)
}

// UpdateCDNOriginRequest represents a request to update a CDN origin.
type UpdateCDNOriginRequest struct {
	Name               *string `json:"name,omitempty"`
	Address            *string `json:"address,omitempty"`
	Port               *int    `json:"port,omitempty"`
	Protocol           *string `json:"protocol,omitempty"`
	Weight             *int    `json:"weight,omitempty"`
	Priority           *int    `json:"priority,omitempty"`
	IsBackup           *bool   `json:"is_backup,omitempty"`
	HostHeader         *string `json:"host_header,omitempty"`
	BasePath           *string `json:"base_path,omitempty"`
	HealthCheckEnabled *bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath    *string `json:"health_check_path,omitempty"`
	VerifySSL          *bool   `json:"verify_ssl,omitempty"`
	Enabled            *bool   `json:"enabled,omitempty"`
}

// CreateCDNRuleRequest represents a request to create a CDN rule.
type CreateCDNRuleRequest struct {
	Name            string          `json:"name"`
	RuleType        string          `json:"rule_type"`
	Priority        int             `json:"priority"`
	MatchConditions json.RawMessage `json:"match_conditions,omitempty"`
	ActionConfig    json.RawMessage `json:"action_config"`
	Enabled         bool            `json:"enabled"`
}

// UpdateCDNRuleRequest represents a request to update a CDN rule.
type UpdateCDNRuleRequest struct {
	Name            *string          `json:"name,omitempty"`
	Priority        *int             `json:"priority,omitempty"`
	MatchConditions *json.RawMessage `json:"match_conditions,omitempty"`
	ActionConfig    *json.RawMessage `json:"action_config,omitempty"`
	Enabled         *bool            `json:"enabled,omitempty"`
}

type cdnService struct {
	client *Client
}

// Zones

func (s *cdnService) ListZones(ctx context.Context) ([]CDNZone, error) {
	var zones []CDNZone
	if err := s.client.get(ctx, "/cdn/zones", &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (s *cdnService) GetZone(ctx context.Context, zoneUUID string) (*CDNZone, error) {
	var zone CDNZone
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s", zoneUUID), &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

func (s *cdnService) CreateZone(ctx context.Context, req *CreateCDNZoneRequest) (*CDNZone, error) {
	var zone CDNZone
	if err := s.client.post(ctx, "/cdn/zones", req, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

func (s *cdnService) UpdateZone(ctx context.Context, zoneUUID string, req *UpdateCDNZoneRequest) (*CDNZone, error) {
	var zone CDNZone
	if err := s.client.patch(ctx, fmt.Sprintf("/cdn/zones/%s", zoneUUID), req, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

func (s *cdnService) DeleteZone(ctx context.Context, zoneUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/cdn/zones/%s", zoneUUID))
}

func (s *cdnService) GetZonePricing(ctx context.Context, zoneUUID string) (json.RawMessage, error) {
	data, err := s.client.getRaw(ctx, fmt.Sprintf("/cdn/zones/%s/pricing", zoneUUID))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func (s *cdnService) ListPlans(ctx context.Context) ([]CDNPlan, error) {
	var plans []CDNPlan
	if err := s.client.get(ctx, "/cdn/plans", &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

// Origins

func (s *cdnService) ListOrigins(ctx context.Context, zoneUUID string) ([]CDNOrigin, error) {
	var origins []CDNOrigin
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/origins", zoneUUID), &origins); err != nil {
		return nil, err
	}
	return origins, nil
}

func (s *cdnService) CreateOrigin(ctx context.Context, zoneUUID string, req *CreateCDNOriginRequest) (*CDNOrigin, error) {
	var origin CDNOrigin
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/origins", zoneUUID), req, &origin); err != nil {
		return nil, err
	}
	return &origin, nil
}

func (s *cdnService) UpdateOrigin(ctx context.Context, zoneUUID, originUUID string, req *UpdateCDNOriginRequest) (*CDNOrigin, error) {
	var origin CDNOrigin
	if err := s.client.patch(ctx, fmt.Sprintf("/cdn/zones/%s/origins/%s", zoneUUID, originUUID), req, &origin); err != nil {
		return nil, err
	}
	return &origin, nil
}

func (s *cdnService) DeleteOrigin(ctx context.Context, zoneUUID, originUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/cdn/zones/%s/origins/%s", zoneUUID, originUUID))
}

// Rules

func (s *cdnService) ListRules(ctx context.Context, zoneUUID string) ([]CDNRule, error) {
	var rules []CDNRule
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/rules", zoneUUID), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *cdnService) GetRule(ctx context.Context, zoneUUID, ruleUUID string) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/rules/%s", zoneUUID, ruleUUID), &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) CreateRule(ctx context.Context, zoneUUID string, req *CreateCDNRuleRequest) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/rules", zoneUUID), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) UpdateRule(ctx context.Context, zoneUUID, ruleUUID string, req *UpdateCDNRuleRequest) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.patch(ctx, fmt.Sprintf("/cdn/zones/%s/rules/%s", zoneUUID, ruleUUID), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) DeleteRule(ctx context.Context, zoneUUID, ruleUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/cdn/zones/%s/rules/%s", zoneUUID, ruleUUID))
}

// WAF Rules

func (s *cdnService) ListWAFRules(ctx context.Context, zoneUUID string) ([]CDNRule, error) {
	var rules []CDNRule
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/waf-rules", zoneUUID), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *cdnService) GetWAFRule(ctx context.Context, zoneUUID, ruleUUID string) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/waf-rules/%s", zoneUUID, ruleUUID), &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) CreateWAFRule(ctx context.Context, zoneUUID string, req *CreateCDNRuleRequest) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/waf-rules", zoneUUID), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) UpdateWAFRule(ctx context.Context, zoneUUID, ruleUUID string, req *UpdateCDNRuleRequest) (*CDNRule, error) {
	var rule CDNRule
	if err := s.client.patch(ctx, fmt.Sprintf("/cdn/zones/%s/waf-rules/%s", zoneUUID, ruleUUID), req, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func (s *cdnService) DeleteWAFRule(ctx context.Context, zoneUUID, ruleUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/cdn/zones/%s/waf-rules/%s", zoneUUID, ruleUUID))
}

// Metrics

func (s *cdnService) GetMetrics(ctx context.Context, zoneUUID string, metricType string, params *CDNMetricsParams) (json.RawMessage, error) {
	path := fmt.Sprintf("/cdn/zones/%s/metrics/%s", zoneUUID, metricType)

	if params != nil {
		v := url.Values{}
		if params.Minutes > 0 {
			v.Set("minutes", strconv.Itoa(params.Minutes))
		}
		if params.IntervalSeconds > 0 {
			v.Set("interval_seconds", strconv.Itoa(params.IntervalSeconds))
		}
		if params.Limit > 0 {
			v.Set("limit", strconv.Itoa(params.Limit))
		}
		for k, val := range map[string]string{
			"country":      params.Country,
			"asn":          params.ASN,
			"status":       params.Status,
			"status_range": params.StatusRange,
			"cache_status": params.CacheStatus,
			"device_type":  params.DeviceType,
			"path_prefix":  params.PathPrefix,
		} {
			if val != "" {
				v.Set(k, val)
			}
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}

	data, err := s.client.getRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// Actions

// RequestSSL re-triggers automatic SSL issuance for the zone's current
// custom_domain. Use after fixing a missing/incorrect CNAME — the initial
// PATCH zone flow only queues a cert task when custom_domain changes, so
// this is the way to retry without resetting the field.
func (s *cdnService) RequestSSL(ctx context.Context, zoneUUID string) error {
	return s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/request-ssl", zoneUUID), nil, nil)
}

// MoveZoneToProject reassigns a CDN zone to a different project within
// the same organization.
func (s *cdnService) MoveZoneToProject(ctx context.Context, zoneUUID string, projectID int) error {
	body := map[string]any{"project_id": projectID}
	return s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/move-project", zoneUUID), body, nil)
}

// Cache purge

func (s *cdnService) PurgeCache(ctx context.Context, zoneUUID string, req *CDNPurgeRequest) (*CDNPurge, error) {
	var purge CDNPurge
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/purge-cache", zoneUUID), req, &purge); err != nil {
		return nil, err
	}
	return &purge, nil
}

func (s *cdnService) ListPurges(ctx context.Context, zoneUUID string) ([]CDNPurgeStatus, error) {
	var purges []CDNPurgeStatus
	if err := s.client.get(ctx, fmt.Sprintf("/cdn/zones/%s/purge-cache", zoneUUID), &purges); err != nil {
		return nil, err
	}
	return purges, nil
}

// Token Auth

func (s *cdnService) RotateTokenSecret(ctx context.Context, zoneUUID string) (string, error) {
	var result struct {
		TokenAuthSecret string `json:"token_auth_secret"`
	}
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/token-auth/rotate-secret", zoneUUID), nil, &result); err != nil {
		return "", err
	}
	return result.TokenAuthSecret, nil
}

func (s *cdnService) SignURL(ctx context.Context, zoneUUID string, req *CDNSignURLRequest) (*CDNSignedURL, error) {
	var signed CDNSignedURL
	if err := s.client.post(ctx, fmt.Sprintf("/cdn/zones/%s/token-auth/sign-url", zoneUUID), req, &signed); err != nil {
		return nil, err
	}
	return &signed, nil
}
