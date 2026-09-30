package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ManagedDatabaseService handles communication with the Managed Databases (MySQL, PostgreSQL
// and Valkey) related methods of the CubePath API.
//
// Every change is applied asynchronously: a database is created with status "provisioning",
// and scaling, credential rotation and configuration changes move it to "scaling" or
// "updating". Poll Get until the status is back to "active" before the next operation (the
// API answers 409 while another operation of the same database is running).
type ManagedDatabaseService interface {
	// ListPlans returns the plans per location. Engine ("mysql", "postgresql", "valkey")
	// filters the list; leave it empty for every engine. Prices are per node per hour.
	ListPlans(ctx context.Context, engine string) ([]ManagedDatabaseLocationPlans, error)

	List(ctx context.Context) ([]ManagedDatabaseSummary, error)
	Get(ctx context.Context, uuid string) (*ManagedDatabase, error)
	Create(ctx context.Context, req *CreateManagedDatabaseRequest) (*ManagedDatabaseCreated, error)
	Update(ctx context.Context, uuid string, req *UpdateManagedDatabaseRequest) error
	// Delete deletes the database and all its data. A protected database, or one that is
	// still provisioning, cannot be deleted.
	Delete(ctx context.Context, uuid string) error
	SetProtection(ctx context.Context, uuid string, enabled bool) error

	// Scale changes the number of replicas or the plan. Set exactly one of the two fields.
	Scale(ctx context.Context, uuid string, req *ScaleManagedDatabaseRequest) (*ManagedDatabaseScaleResponse, error)
	// GetCredentials returns the admin user, password and connection URI. Only available
	// while the database is active or degraded.
	GetCredentials(ctx context.Context, uuid string) (*ManagedDatabaseCredentials, error)
	// RotateCredentials sets a new admin password in the background. Read it with
	// GetCredentials once the database is active again.
	RotateCredentials(ctx context.Context, uuid string) error
	GetConfig(ctx context.Context, uuid string) (*ManagedDatabaseConfig, error)
	// UpdateConfig changes tunable parameters. RequiresRestart in the response lists the
	// parameters that are applied with a rolling restart.
	UpdateConfig(ctx context.Context, uuid string, params map[string]interface{}) (*ManagedDatabaseConfigUpdateResponse, error)
	GetMetrics(ctx context.Context, uuid string, opts *ManagedDatabaseMetricsOptions) (*ManagedDatabaseMetrics, error)

	// Logical databases
	ListDatabases(ctx context.Context, uuid string) ([]ManagedDatabaseLogicalDatabase, error)
	CreateDatabase(ctx context.Context, uuid, name string) (*ManagedDatabaseLogicalDatabaseCreated, error)
	DeleteDatabase(ctx context.Context, uuid, databaseUUID string) error

	// Users
	ListUsers(ctx context.Context, uuid string) ([]ManagedDatabaseUser, error)
	// CreateUser creates a database user. Leave Password empty to have one generated; the
	// password is only returned by this call.
	CreateUser(ctx context.Context, uuid string, req *CreateManagedDatabaseUserRequest) (*ManagedDatabaseUserCreated, error)
	DeleteUser(ctx context.Context, uuid, userUUID string) error
}

// ManagedDatabasePlan represents a Managed Database plan. PricePerHour is per node.
type ManagedDatabasePlan struct {
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Engine       string  `json:"engine"`
	CPU          int     `json:"cpu"`
	MemoryGB     int     `json:"memory_gb"`
	StorageGB    int     `json:"storage_gb"`
	MaxReplicas  int     `json:"max_replicas"`
	PricePerHour float64 `json:"price_per_hour"`
}

// ManagedDatabaseLocationPlans represents the plans available at a location.
type ManagedDatabaseLocationPlans struct {
	LocationName        string                `json:"location_name"`
	LocationDescription string                `json:"location_description"`
	Plans               []ManagedDatabasePlan `json:"plans"`
}

// ManagedDatabaseSummary represents a Managed Database in a list. EndpointHost and
// EndpointPort are nil until the database is provisioned.
type ManagedDatabaseSummary struct {
	UUID         string  `json:"uuid"`
	ProjectID    int     `json:"project_id"`
	Name         string  `json:"name"`
	Label        *string `json:"label"`
	Engine       string  `json:"engine"`
	Version      string  `json:"version"`
	Status       string  `json:"status"`
	EndpointHost *string `json:"endpoint_host"`
	EndpointPort *int    `json:"endpoint_port"`
	Replicas     int     `json:"replicas"`
	Protected    bool    `json:"protected"`
}

// ManagedDatabase represents a Managed Database with its plan, location and backup policy.
// Credentials are not included; use GetCredentials.
type ManagedDatabase struct {
	ManagedDatabaseSummary
	Topology            string              `json:"topology"`
	Plan                ManagedDatabasePlan `json:"plan"`
	Location            Location            `json:"location"`
	BackupEnabled       bool                `json:"backup_enabled"`
	BackupScheduleCron  *string             `json:"backup_schedule_cron"`
	BackupRetentionDays int                 `json:"backup_retention_days"`
	BillingType         string              `json:"billing_type"`
	UpdatedAt           string              `json:"updated_at"`
}

// ManagedDatabaseBackupConfig is the backup policy set at creation: a 5-field cron schedule
// (minute hour day month weekday) and the retention in days (1-365, default 7).
type ManagedDatabaseBackupConfig struct {
	ScheduleCron  string `json:"schedule_cron"`
	RetentionDays int    `json:"retention_days,omitempty"`
}

// CreateManagedDatabaseRequest represents a request to create a Managed Database.
//
// Engine is "mysql", "postgresql" or "valkey"; Version must be one of the versions the
// platform offers for the engine; PlanUUID comes from ListPlans. Replicas defaults to 3 and
// has an engine minimum (mysql 3, postgresql 2, valkey 2). Topology defaults per engine.
type CreateManagedDatabaseRequest struct {
	ProjectID int                          `json:"project_id"`
	Name      string                       `json:"name"`
	Engine    string                       `json:"engine"`
	Version   string                       `json:"version"`
	PlanUUID  string                       `json:"plan_uuid"`
	Replicas  int                          `json:"replicas,omitempty"`
	Topology  string                       `json:"topology,omitempty"`
	Backup    *ManagedDatabaseBackupConfig `json:"backup,omitempty"`
}

// ManagedDatabaseCreated is the response of a creation. Provisioning continues in the
// background; poll Get until Status is "active".
type ManagedDatabaseCreated struct {
	Detail  string `json:"detail"`
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Engine  string `json:"engine"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// ManagedDatabaseBackupUpdate changes the backup policy; nil fields are left unchanged.
type ManagedDatabaseBackupUpdate struct {
	Enabled       *bool   `json:"enabled,omitempty"`
	ScheduleCron  *string `json:"schedule_cron,omitempty"`
	RetentionDays *int    `json:"retention_days,omitempty"`
}

// UpdateManagedDatabaseRequest represents a request to update the name, label or backup
// policy of a Managed Database. At least one field is required.
type UpdateManagedDatabaseRequest struct {
	Name   *string                      `json:"name,omitempty"`
	Label  *string                      `json:"label,omitempty"`
	Backup *ManagedDatabaseBackupUpdate `json:"backup,omitempty"`
}

// ScaleManagedDatabaseRequest scales a database horizontally (Replicas) or vertically
// (PlanUUID, a plan of the same engine and location). Set exactly one of them.
type ScaleManagedDatabaseRequest struct {
	Replicas *int    `json:"replicas,omitempty"`
	PlanUUID *string `json:"plan_uuid,omitempty"`
}

// ManagedDatabaseScaleResponse is the response of a scale request: Replicas is set for
// horizontal scaling and Plan (the plan name) for vertical scaling.
type ManagedDatabaseScaleResponse struct {
	Detail   string  `json:"detail"`
	UUID     string  `json:"uuid"`
	Replicas *int    `json:"replicas"`
	Plan     *string `json:"plan"`
}

// ManagedDatabaseCredentials are the connection credentials of the admin user.
type ManagedDatabaseCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	URI      string `json:"uri"`
}

// ManagedDatabaseConfigParam describes one tunable parameter. Min and Max are set for
// numeric parameters and Enum for parameters with a fixed set of values.
type ManagedDatabaseConfigParam struct {
	Type            string          `json:"type"`
	Default         json.RawMessage `json:"default"`
	Value           json.RawMessage `json:"value"`
	ValueSource     string          `json:"value_source"`
	RequiresRestart bool            `json:"requires_restart"`
	Description     string          `json:"description"`
	Min             json.RawMessage `json:"min,omitempty"`
	Max             json.RawMessage `json:"max,omitempty"`
	Enum            json.RawMessage `json:"enum,omitempty"`
}

// ManagedDatabaseConfig is the tunable configuration of a database, keyed by parameter name.
type ManagedDatabaseConfig struct {
	Engine string                                `json:"engine"`
	Params map[string]ManagedDatabaseConfigParam `json:"params"`
	Note   string                                `json:"note"`
}

// ManagedDatabaseConfigUpdateResponse is the response of a configuration change.
type ManagedDatabaseConfigUpdateResponse struct {
	Detail          string   `json:"detail"`
	UUID            string   `json:"uuid"`
	RequiresRestart []string `json:"requires_restart"`
}

// ManagedDatabaseMetricsOptions selects the metrics and the time range. Metrics is a subset of
// "connections", "cpu", "memory" and "replication_lag" (default: all). TimeRange is like "1h",
// "24h", "7d" or "30d" (default "1h").
type ManagedDatabaseMetricsOptions struct {
	Metrics   []string
	TimeRange string
}

// ManagedDatabaseMetrics are time series keyed by metric name; each point is
// [unix_timestamp, value]. A series is empty when no data is available.
type ManagedDatabaseMetrics struct {
	Start   int64                   `json:"start"`
	End     int64                   `json:"end"`
	Metrics map[string][][2]float64 `json:"metrics"`
}

// ManagedDatabaseLogicalDatabase represents a logical database inside a Managed Database.
type ManagedDatabaseLogicalDatabase struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	CreatedAt *string `json:"created_at"`
}

// ManagedDatabaseLogicalDatabaseCreated is the response of a logical database creation.
type ManagedDatabaseLogicalDatabaseCreated struct {
	Detail string `json:"detail"`
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ManagedDatabaseUser represents a database user. The password is never listed.
type ManagedDatabaseUser struct {
	UUID      string  `json:"uuid"`
	Username  string  `json:"username"`
	Status    string  `json:"status"`
	CreatedAt *string `json:"created_at"`
}

// CreateManagedDatabaseUserRequest represents a request to create a database user. The
// password (12-64 characters) is generated when empty.
type CreateManagedDatabaseUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

// ManagedDatabaseUserCreated is the response of a user creation, with its password.
type ManagedDatabaseUserCreated struct {
	Detail   string `json:"detail"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type managedDatabaseService struct {
	client *Client
}

func mdbPath(uuid string, rest ...string) string {
	p := "/managed-databases/" + url.PathEscape(uuid)
	for _, r := range rest {
		p += "/" + r
	}
	return p
}

func (s *managedDatabaseService) ListPlans(ctx context.Context, engine string) ([]ManagedDatabaseLocationPlans, error) {
	path := "/managed-database-plans/"
	if engine != "" {
		path += "?engine=" + url.QueryEscape(engine)
	}
	var plans []ManagedDatabaseLocationPlans
	if err := s.client.get(ctx, path, &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *managedDatabaseService) List(ctx context.Context) ([]ManagedDatabaseSummary, error) {
	var dbs []ManagedDatabaseSummary
	if err := s.client.get(ctx, "/managed-databases/", &dbs); err != nil {
		return nil, err
	}
	return dbs, nil
}

func (s *managedDatabaseService) Get(ctx context.Context, uuid string) (*ManagedDatabase, error) {
	var db ManagedDatabase
	if err := s.client.get(ctx, mdbPath(uuid), &db); err != nil {
		return nil, err
	}
	return &db, nil
}

func (s *managedDatabaseService) Create(ctx context.Context, req *CreateManagedDatabaseRequest) (*ManagedDatabaseCreated, error) {
	var created ManagedDatabaseCreated
	if err := s.client.post(ctx, "/managed-databases/", req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *managedDatabaseService) Update(ctx context.Context, uuid string, req *UpdateManagedDatabaseRequest) error {
	return s.client.patch(ctx, mdbPath(uuid), req, nil)
}

func (s *managedDatabaseService) Delete(ctx context.Context, uuid string) error {
	return s.client.del(ctx, mdbPath(uuid))
}

func (s *managedDatabaseService) SetProtection(ctx context.Context, uuid string, enabled bool) error {
	body := map[string]interface{}{
		"enabled": enabled,
	}
	return s.client.post(ctx, mdbPath(uuid, "protection"), body, nil)
}

func (s *managedDatabaseService) Scale(ctx context.Context, uuid string, req *ScaleManagedDatabaseRequest) (*ManagedDatabaseScaleResponse, error) {
	var result ManagedDatabaseScaleResponse
	if err := s.client.post(ctx, mdbPath(uuid, "scale"), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *managedDatabaseService) GetCredentials(ctx context.Context, uuid string) (*ManagedDatabaseCredentials, error) {
	var creds ManagedDatabaseCredentials
	if err := s.client.get(ctx, mdbPath(uuid, "credentials"), &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (s *managedDatabaseService) RotateCredentials(ctx context.Context, uuid string) error {
	return s.client.post(ctx, mdbPath(uuid, "credentials", "rotate"), nil, nil)
}

func (s *managedDatabaseService) GetConfig(ctx context.Context, uuid string) (*ManagedDatabaseConfig, error) {
	var cfg ManagedDatabaseConfig
	if err := s.client.get(ctx, mdbPath(uuid, "config"), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *managedDatabaseService) UpdateConfig(ctx context.Context, uuid string, params map[string]interface{}) (*ManagedDatabaseConfigUpdateResponse, error) {
	body := map[string]interface{}{
		"params": params,
	}
	var result ManagedDatabaseConfigUpdateResponse
	if err := s.client.patch(ctx, mdbPath(uuid, "config"), body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *managedDatabaseService) GetMetrics(ctx context.Context, uuid string, opts *ManagedDatabaseMetricsOptions) (*ManagedDatabaseMetrics, error) {
	path := mdbPath(uuid, "metrics")
	if opts != nil {
		v := url.Values{}
		if len(opts.Metrics) > 0 {
			v.Set("metrics", strings.Join(opts.Metrics, ","))
		}
		if opts.TimeRange != "" {
			v.Set("time_range", opts.TimeRange)
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}
	var metrics ManagedDatabaseMetrics
	if err := s.client.get(ctx, path, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}

func (s *managedDatabaseService) ListDatabases(ctx context.Context, uuid string) ([]ManagedDatabaseLogicalDatabase, error) {
	var dbs []ManagedDatabaseLogicalDatabase
	if err := s.client.get(ctx, mdbPath(uuid, "databases"), &dbs); err != nil {
		return nil, err
	}
	return dbs, nil
}

func (s *managedDatabaseService) CreateDatabase(ctx context.Context, uuid, name string) (*ManagedDatabaseLogicalDatabaseCreated, error) {
	body := map[string]interface{}{
		"name": name,
	}
	var created ManagedDatabaseLogicalDatabaseCreated
	if err := s.client.post(ctx, mdbPath(uuid, "databases"), body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *managedDatabaseService) DeleteDatabase(ctx context.Context, uuid, databaseUUID string) error {
	return s.client.del(ctx, mdbPath(uuid, "databases", url.PathEscape(databaseUUID)))
}

func (s *managedDatabaseService) ListUsers(ctx context.Context, uuid string) ([]ManagedDatabaseUser, error) {
	var users []ManagedDatabaseUser
	if err := s.client.get(ctx, mdbPath(uuid, "users"), &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *managedDatabaseService) CreateUser(ctx context.Context, uuid string, req *CreateManagedDatabaseUserRequest) (*ManagedDatabaseUserCreated, error) {
	if req == nil || req.Username == "" {
		return nil, fmt.Errorf("CreateManagedDatabaseUserRequest.Username is required")
	}
	var created ManagedDatabaseUserCreated
	if err := s.client.post(ctx, mdbPath(uuid, "users"), req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (s *managedDatabaseService) DeleteUser(ctx context.Context, uuid, userUUID string) error {
	return s.client.del(ctx, mdbPath(uuid, "users", url.PathEscape(userUUID)))
}
