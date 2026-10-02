package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ObjectStorageService handles communication with the Object Storage (S3 compatible) related
// methods of the CubePath API.
//
// Buckets and access keys are created asynchronously: they start as "pending" and become
// "active" after a few seconds (poll GetBucket or ListKeys until then).
type ObjectStorageService interface {
	ListTiers(ctx context.Context) ([]ObjectStorageTier, error)

	ListBuckets(ctx context.Context, opts *ObjectStorageListOptions) ([]ObjectStorageBucket, error)
	GetBucket(ctx context.Context, uuid string) (*ObjectStorageBucketDetail, error)
	CreateBucket(ctx context.Context, req *CreateObjectStorageBucketRequest) (*ObjectStorageBucketCreated, error)
	UpdateBucket(ctx context.Context, uuid string, req *UpdateObjectStorageBucketRequest) error
	// DeleteBucket deletes a bucket. Without force only an empty bucket is deleted; with force its
	// content is purged first.
	DeleteBucket(ctx context.Context, uuid string, force bool) error

	ListKeys(ctx context.Context, opts *ObjectStorageListOptions) ([]ObjectStorageKey, error)
	// CreateKey creates an access key. The secret is only returned by this call.
	CreateKey(ctx context.Context, req *CreateObjectStorageKeyRequest) (*ObjectStorageKeyCreated, error)
	DeleteKey(ctx context.Context, uuid string) error

	GetUsage(ctx context.Context, opts *ObjectStorageUsageOptions) (*ObjectStorageUsage, error)

	// GetBucketMetrics returns the chart series of a bucket over a window (H1, H3, H6, H12, H24,
	// D3, D7 or D30) as the raw GraphQL ObjectStorageBucket: {"uuid", "name", "storageMeasuredAt",
	// "storage", "traffic", "responses"}, each part a MetricsResult {"start", "end", "step",
	// "series": [{"name", "unit", "points": [{"ts", "value"}]}]}. storage: size_bytes, objects
	// (hourly); traffic: egress_bytes, cdn_bytes, ingress_bytes, class_a_requests,
	// class_b_requests, free_requests; responses: responses_2xx, responses_3xx, responses_4xx,
	// responses_5xx, responses_429, responses_other. Traffic and responses are totals per step,
	// not rates. Served through GraphQL.
	GetBucketMetrics(ctx context.Context, uuid string, timeRange string) (json.RawMessage, error)
}

// ObjectStorageTierSummary identifies the tier of a bucket or access key.
type ObjectStorageTierSummary struct {
	UUID  string `json:"uuid"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Media string `json:"media"`
}

// ObjectStorageTierPrices are the tier prices in USD.
type ObjectStorageTierPrices struct {
	StorageGBMonth float64 `json:"storage_gb_month"`
	EgressGB       float64 `json:"egress_gb"`
	ClassAPer1K    float64 `json:"class_a_per_1k"`
	ClassBPer1K    float64 `json:"class_b_per_1k"`
}

// ObjectStorageFreeTier is the monthly free tier, per organization and tier.
type ObjectStorageFreeTier struct {
	StorageGBMonth float64 `json:"storage_gb_month"`
	EgressGB       float64 `json:"egress_gb"`
	Requests       int64   `json:"requests"`
}

// ObjectStorageTier represents a storage tier.
type ObjectStorageTier struct {
	UUID                string                  `json:"uuid"`
	Slug                string                  `json:"slug"`
	Name                string                  `json:"name"`
	Media               string                  `json:"media"`
	LocationID          int                     `json:"location_id"`
	LocationName        string                  `json:"location_name"`
	LocationDescription string                  `json:"location_description"`
	Region              string                  `json:"region"`
	Endpoint            string                  `json:"endpoint"`
	Prices              ObjectStorageTierPrices `json:"prices"`
	FreeTier            ObjectStorageFreeTier   `json:"free_tier"`
	AcceptingNew        bool                    `json:"accepting_new"`
}

// ObjectStorageBucket represents a bucket as returned by the list.
type ObjectStorageBucket struct {
	UUID           string                   `json:"uuid"`
	Name           string                   `json:"name"`
	Status         string                   `json:"status"`
	SuspendReason  *string                  `json:"suspend_reason"`
	WriteBlocked   bool                     `json:"write_blocked"`
	ErrorMessage   *string                  `json:"error_message"`
	ProjectID      *int                     `json:"project_id"`
	Tier           ObjectStorageTierSummary `json:"tier"`
	LocationName   string                   `json:"location_name"`
	Region         string                   `json:"region"`
	Endpoint       string                   `json:"endpoint"`
	Versioning     string                   `json:"versioning"`
	Protected      bool                     `json:"protected"`
	SizeBytes      int64                    `json:"size_bytes"`
	ObjectsCount   int64                    `json:"objects_count"`
	UsageUpdatedAt *string                  `json:"usage_updated_at"`
	MonthlyCharges float64                  `json:"monthly_charges"`
	CDNConnected   bool                     `json:"cdn_connected"`
}

// ObjectStorageConnection holds the S3 connection details of a bucket.
type ObjectStorageConnection struct {
	Endpoint       string `json:"endpoint"`
	Region         string `json:"region"`
	PathStyleURL   string `json:"path_style_url"`
	VirtualHostURL string `json:"virtual_host_url"`
}

// ObjectStorageBucketUsage is the month to date usage of a bucket.
type ObjectStorageBucketUsage struct {
	Period            string  `json:"period"`
	Since             string  `json:"since"`
	Until             string  `json:"until"`
	StorageGiBHours   float64 `json:"storage_gib_hours"`
	StorageGiBMonth   float64 `json:"storage_gib_month"`
	EgressBytes       int64   `json:"egress_bytes"`
	CDNBytes          int64   `json:"cdn_bytes"`
	ClassARequests    int64   `json:"class_a_requests"`
	ClassBRequests    int64   `json:"class_b_requests"`
	ClassBCDNRequests int64   `json:"class_b_cdn_requests"`
}

// ObjectStorageBucketCDN describes the CDN origin serving a bucket.
type ObjectStorageBucketCDN struct {
	Status        string  `json:"status"`
	ZoneUUID      string  `json:"zone_uuid"`
	ZoneName      string  `json:"zone_name"`
	Domain        string  `json:"domain"`
	CustomDomain  *string `json:"custom_domain"`
	ZoneStatus    string  `json:"zone_status"`
	OriginUUID    string  `json:"origin_uuid"`
	OriginEnabled bool    `json:"origin_enabled"`
}

// ObjectStorageBucketDetail is a bucket with its connection details, usage and CDN block.
type ObjectStorageBucketDetail struct {
	ObjectStorageBucket
	ActiveAt       *string                   `json:"active_at"`
	LastBilledTime *string                   `json:"last_billed_time"`
	Connection     *ObjectStorageConnection  `json:"connection"`
	Usage          *ObjectStorageBucketUsage `json:"usage"`
	CDN            *ObjectStorageBucketCDN   `json:"cdn"`
}

// CreateObjectStorageBucketRequest represents a request to create a bucket.
type CreateObjectStorageBucketRequest struct {
	Name       string `json:"name"`
	Tier       string `json:"tier"`
	ProjectID  *int   `json:"project_id,omitempty"`
	Versioning bool   `json:"versioning,omitempty"`
}

// ObjectStorageBucketCreated is the response of a bucket creation.
type ObjectStorageBucketCreated struct {
	Detail    string                   `json:"detail"`
	UUID      string                   `json:"uuid"`
	Name      string                   `json:"name"`
	Status    string                   `json:"status"`
	ProjectID *int                     `json:"project_id"`
	Tier      ObjectStorageTierSummary `json:"tier"`
	Region    string                   `json:"region"`
	Endpoint  string                   `json:"endpoint"`
}

// UpdateObjectStorageBucketRequest represents a request to update a bucket. Versioning is
// "enabled" or "suspended".
type UpdateObjectStorageBucketRequest struct {
	Versioning *string `json:"versioning,omitempty"`
	Protected  *bool   `json:"protected,omitempty"`
}

// ObjectStorageBucketRef identifies a bucket an access key is limited to.
type ObjectStorageBucketRef struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// ObjectStorageKey represents an access key. The secret is never returned by the list.
type ObjectStorageKey struct {
	UUID        string                   `json:"uuid"`
	Name        string                   `json:"name"`
	AccessKeyID string                   `json:"access_key_id"`
	Permission  string                   `json:"permission"`
	BucketScope []ObjectStorageBucketRef `json:"bucket_scope"`
	ProjectID   *int                     `json:"project_id"`
	Tier        ObjectStorageTierSummary `json:"tier"`
	Region      string                   `json:"region"`
	Endpoint    string                   `json:"endpoint"`
	Status      string                   `json:"status"`
	ExpiresAt   *string                  `json:"expires_at"`
}

// CreateObjectStorageKeyRequest represents a request to create an access key. Permission is
// "read_write" or "read_only". Leave BucketUUIDs empty to give the key access to every bucket
// of the project in the tier.
type CreateObjectStorageKeyRequest struct {
	Name        string     `json:"name"`
	Tier        string     `json:"tier"`
	ProjectID   *int       `json:"project_id,omitempty"`
	Permission  string     `json:"permission"`
	BucketUUIDs []string   `json:"bucket_uuids,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ObjectStorageKeyCreated is the response of an access key creation, including the secret.
type ObjectStorageKeyCreated struct {
	Detail          string                   `json:"detail"`
	UUID            string                   `json:"uuid"`
	Name            string                   `json:"name"`
	AccessKeyID     string                   `json:"access_key_id"`
	SecretAccessKey string                   `json:"secret_access_key"`
	Permission      string                   `json:"permission"`
	BucketScope     []ObjectStorageBucketRef `json:"bucket_scope"`
	ProjectID       *int                     `json:"project_id"`
	Tier            ObjectStorageTierSummary `json:"tier"`
	Region          string                   `json:"region"`
	Endpoint        string                   `json:"endpoint"`
	Status          string                   `json:"status"`
	ExpiresAt       *string                  `json:"expires_at"`
}

// ObjectStorageListOptions filters bucket and key lists. Tier is a tier uuid or slug.
type ObjectStorageListOptions struct {
	ProjectID int
	Tier      string
}

// ObjectStorageUsageOptions selects the usage report. Period is "YYYY-MM" (default: current
// month).
type ObjectStorageUsageOptions struct {
	Period    string
	ProjectID int
	Tier      string
}

// ObjectStorageFreeTierUsage is the included and used amount of one free tier item.
type ObjectStorageFreeTierUsage struct {
	Included float64  `json:"included"`
	Used     *float64 `json:"used"`
}

// ObjectStorageUsageTier is the month usage of one tier.
type ObjectStorageUsageTier struct {
	Tier              ObjectStorageTierSummary              `json:"tier"`
	StorageGiBHours   *float64                              `json:"storage_gib_hours"`
	StorageGiBMonth   *float64                              `json:"storage_gib_month"`
	EgressBytes       *int64                                `json:"egress_bytes"`
	CDNBytes          *int64                                `json:"cdn_bytes"`
	ClassARequests    *int64                                `json:"class_a_requests"`
	ClassBRequests    *int64                                `json:"class_b_requests"`
	ClassBCDNRequests *int64                                `json:"class_b_cdn_requests"`
	Cost              float64                               `json:"cost"`
	ProjectedCost     float64                               `json:"projected_cost"`
	FreeTier          map[string]ObjectStorageFreeTierUsage `json:"free_tier"`
}

// ObjectStorageUsageBucket is the month usage of one bucket.
type ObjectStorageUsageBucket struct {
	UUID              string   `json:"uuid"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	ProjectID         *int     `json:"project_id"`
	TierUUID          string   `json:"tier_uuid"`
	StorageGiBHours   *float64 `json:"storage_gib_hours"`
	StorageGiBMonth   *float64 `json:"storage_gib_month"`
	EgressBytes       *int64   `json:"egress_bytes"`
	CDNBytes          *int64   `json:"cdn_bytes"`
	ClassARequests    *int64   `json:"class_a_requests"`
	ClassBRequests    *int64   `json:"class_b_requests"`
	ClassBCDNRequests *int64   `json:"class_b_cdn_requests"`
	Cost              float64  `json:"cost"`
}

// ObjectStorageUsage is the month usage of the organization per tier and bucket. Quantity
// fields are nil when MetricsAvailable is false; costs are always present.
type ObjectStorageUsage struct {
	Period           string                     `json:"period"`
	Since            string                     `json:"since"`
	Until            string                     `json:"until"`
	MetricsAvailable bool                       `json:"metrics_available"`
	TotalCost        float64                    `json:"total_cost"`
	ProjectedCost    float64                    `json:"projected_cost"`
	Tiers            []ObjectStorageUsageTier   `json:"tiers"`
	Buckets          []ObjectStorageUsageBucket `json:"buckets"`
	AvailableMonths  []string                   `json:"available_months"`
}

type objectStorageService struct {
	client *Client
}

func (o *ObjectStorageListOptions) query() string {
	if o == nil {
		return ""
	}
	v := url.Values{}
	if o.ProjectID > 0 {
		v.Set("project_id", strconv.Itoa(o.ProjectID))
	}
	if o.Tier != "" {
		v.Set("tier", o.Tier)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

func (o *ObjectStorageUsageOptions) query() string {
	if o == nil {
		return ""
	}
	v := url.Values{}
	if o.Period != "" {
		v.Set("period", o.Period)
	}
	if o.ProjectID > 0 {
		v.Set("project_id", strconv.Itoa(o.ProjectID))
	}
	if o.Tier != "" {
		v.Set("tier", o.Tier)
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

func (s *objectStorageService) ListTiers(ctx context.Context) ([]ObjectStorageTier, error) {
	var tiers []ObjectStorageTier
	if err := s.client.get(ctx, "/object-storage/tiers", &tiers); err != nil {
		return nil, err
	}
	return tiers, nil
}

func (s *objectStorageService) ListBuckets(ctx context.Context, opts *ObjectStorageListOptions) ([]ObjectStorageBucket, error) {
	var buckets []ObjectStorageBucket
	if err := s.client.get(ctx, "/object-storage/buckets"+opts.query(), &buckets); err != nil {
		return nil, err
	}
	return buckets, nil
}

func (s *objectStorageService) GetBucket(ctx context.Context, uuid string) (*ObjectStorageBucketDetail, error) {
	var bucket ObjectStorageBucketDetail
	if err := s.client.get(ctx, fmt.Sprintf("/object-storage/buckets/%s", url.PathEscape(uuid)), &bucket); err != nil {
		return nil, err
	}
	return &bucket, nil
}

func (s *objectStorageService) CreateBucket(ctx context.Context, req *CreateObjectStorageBucketRequest) (*ObjectStorageBucketCreated, error) {
	var bucket ObjectStorageBucketCreated
	if err := s.client.post(ctx, "/object-storage/buckets", req, &bucket); err != nil {
		return nil, err
	}
	return &bucket, nil
}

func (s *objectStorageService) UpdateBucket(ctx context.Context, uuid string, req *UpdateObjectStorageBucketRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/object-storage/buckets/%s", url.PathEscape(uuid)), req, nil)
}

func (s *objectStorageService) DeleteBucket(ctx context.Context, uuid string, force bool) error {
	path := fmt.Sprintf("/object-storage/buckets/%s", url.PathEscape(uuid))
	if force {
		path += "?force=true"
	}
	return s.client.del(ctx, path)
}

func (s *objectStorageService) ListKeys(ctx context.Context, opts *ObjectStorageListOptions) ([]ObjectStorageKey, error) {
	var keys []ObjectStorageKey
	if err := s.client.get(ctx, "/object-storage/keys"+opts.query(), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *objectStorageService) CreateKey(ctx context.Context, req *CreateObjectStorageKeyRequest) (*ObjectStorageKeyCreated, error) {
	var key ObjectStorageKeyCreated
	if err := s.client.post(ctx, "/object-storage/keys", req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

func (s *objectStorageService) DeleteKey(ctx context.Context, uuid string) error {
	return s.client.del(ctx, fmt.Sprintf("/object-storage/keys/%s", url.PathEscape(uuid)))
}

func (s *objectStorageService) GetUsage(ctx context.Context, opts *ObjectStorageUsageOptions) (*ObjectStorageUsage, error) {
	var usage ObjectStorageUsage
	if err := s.client.get(ctx, "/object-storage/usage"+opts.query(), &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

// objectStorageBucketMetricsQuery is shared by GetBucketMetrics and its tests.
const objectStorageBucketMetricsQuery = `query($uuid: ID!, $range: TimeRange!) { objectStorageBucket(uuid: $uuid) { uuid name storageMeasuredAt storage(range: $range) { start end step series { name unit points { ts value } } } traffic(range: $range) { start end step series { name unit points { ts value } } } responses(range: $range) { start end step series { name unit points { ts value } } } } }`

func (s *objectStorageService) GetBucketMetrics(ctx context.Context, uuid string, timeRange string) (json.RawMessage, error) {
	if timeRange == "" {
		timeRange = "H24"
	}
	var data struct {
		Bucket json.RawMessage `json:"objectStorageBucket"`
	}
	vars := map[string]interface{}{"uuid": uuid, "range": timeRange}
	if err := s.client.graphQL(ctx, objectStorageBucketMetricsQuery, vars, &data); err != nil {
		return nil, err
	}
	if len(data.Bucket) == 0 || string(data.Bucket) == "null" {
		return nil, &APIError{StatusCode: 404, Message: "Not Found", Detail: "Bucket not found"}
	}
	return data.Bucket, nil
}
