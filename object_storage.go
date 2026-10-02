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
	// DeleteBucketWithOptions is DeleteBucket with every option, such as BypassGovernance for a
	// bucket with Object Lock.
	DeleteBucketWithOptions(ctx context.Context, uuid string, opts *DeleteObjectStorageBucketOptions) error
	// SetObjectStorageBucketObjectLock changes or removes (DefaultRetention nil) the default
	// retention of a bucket created with Object Lock. Object Lock itself can only be enabled
	// when the bucket is created.
	SetObjectStorageBucketObjectLock(ctx context.Context, uuid string, req *SetObjectStorageBucketObjectLockRequest) error
	// EnableBucketEncryption turns on encryption at rest (AES-256) for a bucket created without
	// it; the objects already stored are encrypted in the background (ReencryptJobID). It cannot
	// be turned off afterwards. Calling it on an encrypted bucket changes nothing.
	EnableBucketEncryption(ctx context.Context, uuid string) (*ObjectStorageEncryptionChange, error)

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
	// GetBucketLifecycle returns the lifecycle rules of a bucket and whether they are applied.
	GetBucketLifecycle(ctx context.Context, uuid string) (*ObjectStorageLifecycle, error)
	// PutBucketLifecycle replaces every lifecycle rule of a bucket (1 to 100 rules). Rules are
	// applied asynchronously: poll GetBucketLifecycle until AppliedGeneration reaches the
	// returned Generation. Expiration rules delete objects permanently.
	PutBucketLifecycle(ctx context.Context, uuid string, rules []ObjectStorageLifecycleRule) (*ObjectStorageLifecycleChange, error)
	// DeleteBucketLifecycle removes every lifecycle rule of a bucket.
	DeleteBucketLifecycle(ctx context.Context, uuid string) (*ObjectStorageLifecycleChange, error)
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
	// Tags are the bucket labels (empty map when none). They are managed through the API only
	// and are not visible through S3 bucket tagging.
	Tags map[string]string `json:"tags"`
	// ObjectLock tells whether the bucket was created with Object Lock and its default retention.
	ObjectLock ObjectStorageObjectLock `json:"object_lock"`
	// LockedContentKept is true when the last delete left versions protected by Object Lock
	// (retention or legal hold); the bucket stays and keeps being billed until they expire.
	LockedContentKept bool `json:"locked_content_kept"`
	// Encryption is the bucket's encryption at rest; nil while it is off.
	Encryption *ObjectStorageBucketEncryption `json:"encryption"`
}

// ObjectStorageBucketEncryption is the encryption at rest of a bucket: Algorithm "AES256"
// (SSE-S3) and Scope "all_objects", or "new_objects" while objects written before the bucket
// default may still be stored unencrypted.
type ObjectStorageBucketEncryption struct {
	Algorithm string `json:"algorithm"`
	Scope     string `json:"scope"`
	// AppliedAt is when encryption was turned on for the bucket (UTC, ISO 8601).
	AppliedAt *string `json:"applied_at,omitempty"`
}

// ObjectStorageEncryptionChange is the answer of EnableBucketEncryption. ReencryptJobID is set
// when the bucket had objects: they are being encrypted in the background.
type ObjectStorageEncryptionChange struct {
	Detail         string `json:"detail"`
	ReencryptJobID *int   `json:"reencrypt_job_id"`
}

// ObjectStorageLockRetention is an Object Lock default retention. Mode is "governance" (keys
// with BypassGovernance can still delete) or "compliance" (nobody can delete before the date).
// Set exactly one of Days or Years.
type ObjectStorageLockRetention struct {
	Mode  string `json:"mode"`
	Days  *int   `json:"days,omitempty"`
	Years *int   `json:"years,omitempty"`
}

// ObjectStorageObjectLock is the Object Lock state of a bucket. DefaultRetention is nil when the
// bucket has no default rule.
type ObjectStorageObjectLock struct {
	Enabled          bool                        `json:"enabled"`
	DefaultRetention *ObjectStorageLockRetention `json:"default_retention"`
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
	// Tags are optional bucket labels: at most 50, key 1-128 and value 0-256 characters.
	Tags map[string]string `json:"tags,omitempty"`
	// ObjectLock creates the bucket with Object Lock (WORM). It can only be enabled now, never
	// later, and implies versioning: leave Versioning false (it is not sent) or set it to true.
	// The bucket is also created with deletion protection on.
	ObjectLock bool `json:"object_lock,omitempty"`
	// ObjectLockDefault is the optional default retention of new objects (only with ObjectLock).
	ObjectLockDefault *ObjectStorageLockRetention `json:"object_lock_default,omitempty"`
	// AcceptObjectLockTerms must be true with ObjectLock: it accepts the Object Lock terms.
	AcceptObjectLockTerms bool `json:"accept_object_lock_terms,omitempty"`
	// Encryption chooses encryption at rest (AES-256). nil keeps the default: on. Set it to a
	// pointer to false to create the bucket without it; it can be enabled later
	// (EnableBucketEncryption), never turned off.
	Encryption *bool `json:"encryption,omitempty"`
}

// SetObjectStorageBucketObjectLockRequest changes the default retention of a bucket with Object
// Lock. A nil DefaultRetention removes it (not allowed when the current rule is compliance, which
// can only be kept or lengthened). AcceptObjectLockTerms is required when the change turns
// compliance on or lengthens the retention.
type SetObjectStorageBucketObjectLockRequest struct {
	DefaultRetention      *ObjectStorageLockRetention `json:"default_retention"`
	AcceptObjectLockTerms bool                        `json:"accept_object_lock_terms"`
}

// DeleteObjectStorageBucketOptions are the options of DeleteBucketWithOptions. Force purges the
// bucket content first; BypassGovernance (only with Force, on a bucket with Object Lock) also
// deletes the versions under governance retention. Versions under compliance retention or a
// legal hold are always kept: the bucket then returns with LockedContentKept set.
type DeleteObjectStorageBucketOptions struct {
	Force            bool
	BypassGovernance bool
}

// ObjectStorageBucketCreated is the response of a bucket creation.
type ObjectStorageBucketCreated struct {
	Detail     string                   `json:"detail"`
	UUID       string                   `json:"uuid"`
	Name       string                   `json:"name"`
	Status     string                   `json:"status"`
	ProjectID  *int                     `json:"project_id"`
	Tier       ObjectStorageTierSummary `json:"tier"`
	Region     string                   `json:"region"`
	Endpoint   string                   `json:"endpoint"`
	Tags       map[string]string        `json:"tags"`
	ObjectLock ObjectStorageObjectLock  `json:"object_lock"`
}

// UpdateObjectStorageBucketRequest represents a request to update a bucket. Versioning is
// "enabled" or "suspended".
//
// Tags replaces every label of the bucket when set: nil leaves them unchanged and a pointer to
// an empty map removes them all.
type UpdateObjectStorageBucketRequest struct {
	Versioning *string            `json:"versioning,omitempty"`
	Protected  *bool              `json:"protected,omitempty"`
	Tags       *map[string]string `json:"tags,omitempty"`
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
	// BypassGovernance: the key may delete versions under governance retention (read_write only).
	BypassGovernance bool `json:"bypass_governance"`
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
	// BypassGovernance lets the key delete object versions under governance retention (sending
	// x-amz-bypass-governance-retention: true). Only for read_write keys; it cannot be changed
	// later.
	BypassGovernance bool `json:"bypass_governance,omitempty"`
}

// ObjectStorageKeyCreated is the response of an access key creation, including the secret.
type ObjectStorageKeyCreated struct {
	Detail           string                   `json:"detail"`
	UUID             string                   `json:"uuid"`
	Name             string                   `json:"name"`
	AccessKeyID      string                   `json:"access_key_id"`
	SecretAccessKey  string                   `json:"secret_access_key"`
	Permission       string                   `json:"permission"`
	BucketScope      []ObjectStorageBucketRef `json:"bucket_scope"`
	ProjectID        *int                     `json:"project_id"`
	Tier             ObjectStorageTierSummary `json:"tier"`
	Region           string                   `json:"region"`
	Endpoint         string                   `json:"endpoint"`
	Status           string                   `json:"status"`
	ExpiresAt        *string                  `json:"expires_at"`
	BypassGovernance bool                     `json:"bypass_governance"`
}

// ObjectStorageListOptions filters bucket and key lists. Tier is a tier uuid or slug.
//
// Tags only applies to ListBuckets: each item is "key" (the bucket has that label) or
// "key=value" (split on the first "="); a bucket must match all of them (at most 10).
type ObjectStorageListOptions struct {
	ProjectID int
	Tier      string
	Tags      []string
}

// ObjectStorageUsageOptions selects the usage report. Period is "YYYY-MM" (default: current
// month). Tags filters the buckets like ObjectStorageListOptions.Tags.
type ObjectStorageUsageOptions struct {
	Period    string
	ProjectID int
	Tier      string
	Tags      []string
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
	UUID              string            `json:"uuid"`
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	ProjectID         *int              `json:"project_id"`
	TierUUID          string            `json:"tier_uuid"`
	StorageGiBHours   *float64          `json:"storage_gib_hours"`
	StorageGiBMonth   *float64          `json:"storage_gib_month"`
	EgressBytes       *int64            `json:"egress_bytes"`
	CDNBytes          *int64            `json:"cdn_bytes"`
	ClassARequests    *int64            `json:"class_a_requests"`
	ClassBRequests    *int64            `json:"class_b_requests"`
	ClassBCDNRequests *int64            `json:"class_b_cdn_requests"`
	Cost              float64           `json:"cost"`
	Tags              map[string]string `json:"tags"`
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

// ObjectStorageLifecycleTag is one tag of a lifecycle rule filter.
type ObjectStorageLifecycleTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ObjectStorageLifecycleFilter limits a rule to part of the bucket; nil fields (or a nil
// filter) mean the whole bucket. A leading "/" of the prefix is removed by the API.
type ObjectStorageLifecycleFilter struct {
	Prefix                *string                     `json:"prefix,omitempty"`
	Tags                  []ObjectStorageLifecycleTag `json:"tags,omitempty"`
	ObjectSizeGreaterThan *int64                      `json:"object_size_greater_than,omitempty"`
	ObjectSizeLessThan    *int64                      `json:"object_size_less_than,omitempty"`
}

// ObjectStorageLifecycleExpiration deletes current objects: after Days (1 to 36500), on Date
// ("YYYY-MM-DD", after today UTC), or removes orphan delete markers
// (ExpiredObjectDeleteMarker, alone).
type ObjectStorageLifecycleExpiration struct {
	Days                      *int    `json:"days,omitempty"`
	Date                      *string `json:"date,omitempty"`
	ExpiredObjectDeleteMarker *bool   `json:"expired_object_delete_marker,omitempty"`
}

// ObjectStorageLifecycleNoncurrentExpiration deletes noncurrent versions NoncurrentDays after
// they stop being current, keeping the newest NewerNoncurrentVersions (1 to 100) if set.
type ObjectStorageLifecycleNoncurrentExpiration struct {
	NoncurrentDays          int  `json:"noncurrent_days"`
	NewerNoncurrentVersions *int `json:"newer_noncurrent_versions,omitempty"`
}

// ObjectStorageLifecycleAbortUpload aborts incomplete multipart uploads after 1 to 7 days
// (CubePath aborts them after 7 days anyway).
type ObjectStorageLifecycleAbortUpload struct {
	DaysAfterInitiation int `json:"days_after_initiation"`
}

// ObjectStorageLifecycleRule is one lifecycle rule. ID: 1 to 64 letters, numbers, dots,
// hyphens and underscores, unique, not starting with "cubepath-". At least one action.
type ObjectStorageLifecycleRule struct {
	ID                             string                                      `json:"id"`
	Enabled                        bool                                        `json:"enabled"`
	Filter                         *ObjectStorageLifecycleFilter               `json:"filter,omitempty"`
	Expiration                     *ObjectStorageLifecycleExpiration           `json:"expiration,omitempty"`
	NoncurrentVersionExpiration    *ObjectStorageLifecycleNoncurrentExpiration `json:"noncurrent_version_expiration,omitempty"`
	AbortIncompleteMultipartUpload *ObjectStorageLifecycleAbortUpload          `json:"abort_incomplete_multipart_upload,omitempty"`
}

// ObjectStorageLifecycle is the lifecycle of a bucket. Status: none, pending, active,
// paused (the bucket is blocked or on hold) or error.
type ObjectStorageLifecycle struct {
	BucketUUID        string                       `json:"bucket_uuid"`
	Status            string                       `json:"status"`
	Rules             []ObjectStorageLifecycleRule `json:"rules"`
	PlatformRules     []map[string]interface{}     `json:"platform_rules"`
	Generation        int                          `json:"generation"`
	AppliedGeneration int                          `json:"applied_generation"`
	Error             *string                      `json:"error"`
	UpdatedAt         *string                      `json:"updated_at"`
	Notes             []string                     `json:"notes"`
}

// Applied reports whether the latest change of the rules reached the storage service.
func (l *ObjectStorageLifecycle) Applied() bool {
	return l.AppliedGeneration >= l.Generation
}

// ObjectStorageLifecycleChange is the answer of PutBucketLifecycle and DeleteBucketLifecycle.
// Generation is nil when nothing changed.
type ObjectStorageLifecycleChange struct {
	Detail     string   `json:"detail"`
	Generation *int     `json:"generation"`
	Notes      []string `json:"notes"`
}

type objectStorageService struct {
	client *Client
}

func (o *ObjectStorageListOptions) query(withTags bool) string {
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
	if withTags {
		for _, t := range o.Tags {
			v.Add("tag", t)
		}
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
	for _, t := range o.Tags {
		v.Add("tag", t)
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
	if err := s.client.get(ctx, "/object-storage/buckets"+opts.query(true), &buckets); err != nil {
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
	return s.DeleteBucketWithOptions(ctx, uuid, &DeleteObjectStorageBucketOptions{Force: force})
}

func (s *objectStorageService) DeleteBucketWithOptions(ctx context.Context, uuid string, opts *DeleteObjectStorageBucketOptions) error {
	path := fmt.Sprintf("/object-storage/buckets/%s", url.PathEscape(uuid))
	v := url.Values{}
	if opts != nil && opts.Force {
		v.Set("force", "true")
	}
	if opts != nil && opts.BypassGovernance {
		v.Set("bypass_governance", "true")
	}
	if len(v) > 0 {
		path += "?" + v.Encode()
	}
	return s.client.del(ctx, path)
}

func (s *objectStorageService) SetObjectStorageBucketObjectLock(ctx context.Context, uuid string, req *SetObjectStorageBucketObjectLockRequest) error {
	if req == nil {
		req = &SetObjectStorageBucketObjectLockRequest{}
	}
	return s.client.put(ctx, fmt.Sprintf("/object-storage/buckets/%s/object-lock", url.PathEscape(uuid)), req, nil)
}

func (s *objectStorageService) EnableBucketEncryption(ctx context.Context, uuid string) (*ObjectStorageEncryptionChange, error) {
	var change ObjectStorageEncryptionChange
	body := map[string]bool{"enabled": true}
	if err := s.client.put(ctx, fmt.Sprintf("/object-storage/buckets/%s/encryption", url.PathEscape(uuid)), body, &change); err != nil {
		return nil, err
	}
	return &change, nil
}

func (s *objectStorageService) ListKeys(ctx context.Context, opts *ObjectStorageListOptions) ([]ObjectStorageKey, error) {
	var keys []ObjectStorageKey
	if err := s.client.get(ctx, "/object-storage/keys"+opts.query(false), &keys); err != nil {
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

func lifecyclePath(uuid string) string {
	return fmt.Sprintf("/object-storage/buckets/%s/lifecycle", url.PathEscape(uuid))
}

func (s *objectStorageService) GetBucketLifecycle(ctx context.Context, uuid string) (*ObjectStorageLifecycle, error) {
	var lifecycle ObjectStorageLifecycle
	if err := s.client.get(ctx, lifecyclePath(uuid), &lifecycle); err != nil {
		return nil, err
	}
	return &lifecycle, nil
}

func (s *objectStorageService) PutBucketLifecycle(ctx context.Context, uuid string, rules []ObjectStorageLifecycleRule) (*ObjectStorageLifecycleChange, error) {
	var change ObjectStorageLifecycleChange
	body := map[string]interface{}{"rules": rules}
	if err := s.client.put(ctx, lifecyclePath(uuid), body, &change); err != nil {
		return nil, err
	}
	return &change, nil
}

func (s *objectStorageService) DeleteBucketLifecycle(ctx context.Context, uuid string) (*ObjectStorageLifecycleChange, error) {
	var change ObjectStorageLifecycleChange
	if err := s.client.delWithResult(ctx, lifecyclePath(uuid), &change); err != nil {
		return nil, err
	}
	return &change, nil
}
