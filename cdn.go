package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
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
	GetMetrics(ctx context.Context, zoneUUID string, metricType string, params *CDNMetricsParams) (json.RawMessage, error)
}

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
	CreatedAt    string      `json:"created_at"`
	UpdatedAt    string      `json:"updated_at"`
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
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
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

// CDNMetricsParams represents query parameters for CDN metrics.
type CDNMetricsParams struct {
	Minutes         int    `json:"minutes,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	GroupBy         string `json:"group_by,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

// CreateCDNZoneRequest represents a request to create a CDN zone.
type CreateCDNZoneRequest struct {
	Name         string `json:"name"`
	PlanName     string `json:"plan_name"`
	CustomDomain string `json:"custom_domain,omitempty"`
	ProjectID    *int   `json:"project_id,omitempty"`
}

// UpdateCDNZoneRequest represents a request to update a CDN zone.
type UpdateCDNZoneRequest struct {
	Name            *string `json:"name,omitempty"`
	CustomDomain    *string `json:"custom_domain,omitempty"`
	SSLType         *string `json:"ssl_type,omitempty"`
	CertificateUUID *string `json:"certificate_uuid,omitempty"`
}

// CreateCDNOriginRequest represents a request to create a CDN origin.
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
}

// UpdateCDNOriginRequest represents a request to update a CDN origin.
type UpdateCDNOriginRequest struct {
	Name               *string `json:"name,omitempty"`
	Address            *string `json:"address,omitempty"`
	Port               *int    `json:"port,omitempty"`
	Protocol           *string `json:"protocol,omitempty"`
	Weight             *int    `json:"weight,omitempty"`
	Priority           *int    `json:"priority,omitempty"`
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

	query := ""
	if params != nil {
		if params.Minutes > 0 {
			query += fmt.Sprintf("minutes=%d", params.Minutes)
		}
		if params.IntervalSeconds > 0 {
			if query != "" {
				query += "&"
			}
			query += fmt.Sprintf("interval_seconds=%d", params.IntervalSeconds)
		}
		if params.GroupBy != "" {
			if query != "" {
				query += "&"
			}
			query += fmt.Sprintf("group_by=%s", params.GroupBy)
		}
		if params.Limit > 0 {
			if query != "" {
				query += "&"
			}
			query += fmt.Sprintf("limit=%d", params.Limit)
		}
	}
	if query != "" {
		path += "?" + query
	}

	data, err := s.client.getRaw(ctx, path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
