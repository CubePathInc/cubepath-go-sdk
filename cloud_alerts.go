package cubepath

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// CloudAlertService handles communication with the Cloud Alerts related methods of the CubePath
// API: alerts that watch a metric of a VPS, a baremetal server or an availability group and
// run actions (notify a channel, create or destroy a VPS) when a threshold is crossed, and
// the notification channels they use.
type CloudAlertService interface {
	// Alerts
	List(ctx context.Context, opts *CloudAlertListOptions) ([]CloudAlertSummary, error)
	Get(ctx context.Context, alertID string) (*CloudAlert, error)
	Create(ctx context.Context, req *CreateCloudAlertRequest) (*CloudAlert, error)
	// Update changes an alert; nil fields are left unchanged and a non-nil Actions replaces
	// every action. Set Status to "enabled" or "disabled" to switch the alert on or off.
	Update(ctx context.Context, alertID string, req *UpdateCloudAlertRequest) (*CloudAlert, error)
	Delete(ctx context.Context, alertID string) error
	// History returns the fired and resolved events of an alert, newest first. Limit is
	// 1-200 (default 50 when 0).
	History(ctx context.Context, alertID string, limit int) ([]CloudAlertEvent, error)

	// Notification channels
	ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error)
	GetNotificationChannel(ctx context.Context, channelID string) (*NotificationChannel, error)
	CreateNotificationChannel(ctx context.Context, req *CreateNotificationChannelRequest) (*NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, channelID string, req *UpdateNotificationChannelRequest) (*NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, channelID string) error
}

// Cloud Alert target types.
const (
	CloudAlertTargetVPS               = "vps"
	CloudAlertTargetBaremetal         = "baremetal"
	CloudAlertTargetAvailabilityGroup = "availability_group"
)

// Cloud Alert metrics.
const (
	CloudAlertMetricCPU        = "cpu"
	CloudAlertMetricRAM        = "ram"
	CloudAlertMetricDisk       = "disk"
	CloudAlertMetricNetworkIn  = "network_in"
	CloudAlertMetricNetworkOut = "network_out"
)

// Cloud Alert operators.
const (
	CloudAlertOperatorGreaterThan      = "gt"
	CloudAlertOperatorLessThan         = "lt"
	CloudAlertOperatorGreaterThanEqual = "gte"
	CloudAlertOperatorLessThanEqual    = "lte"
	CloudAlertOperatorEqual            = "eq"
)

// Cloud Alert action types.
const (
	CloudAlertActionNotify     = "notify"
	CloudAlertActionCreateVPS  = "create_vps"
	CloudAlertActionDestroyVPS = "destroy_vps"
)

// Notification channel types.
const (
	NotificationChannelSlack   = "slack"
	NotificationChannelDiscord = "discord"
	NotificationChannelEmail   = "email"
)

// CloudAlertListOptions filters List. Status is "enabled", "disabled", "triggered" or
// "resolved".
type CloudAlertListOptions struct {
	ProjectID int
	Status    string
}

// CloudAlertAction is an action run when an alert fires.
type CloudAlertAction struct {
	ID                    string                 `json:"id"`
	ActionType            string                 `json:"action_type"`
	NotificationChannelID *string                `json:"notificator_id"`
	Config                map[string]interface{} `json:"config"`
	Order                 int                    `json:"order"`
	Enabled               bool                   `json:"enabled"`
	CreatedAt             string                 `json:"created_at"`
}

// CloudAlertSummary represents an alert in a list.
type CloudAlertSummary struct {
	ID           string  `json:"id"`
	ProjectID    int     `json:"project_id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	TargetType   string  `json:"target_type"`
	TargetID     string  `json:"target_id"`
	MetricType   string  `json:"metric_type"`
	Operator     string  `json:"operator"`
	Threshold    float64 `json:"threshold"`
	Status       string  `json:"status"`
	ActionsCount int     `json:"actions_count"`
	CreatedAt    string  `json:"created_at"`
}

// CloudAlert represents an alert with its actions. DurationSeconds is how long the condition
// must hold before the alert fires and CooldownSeconds the minimum time between two firings.
type CloudAlert struct {
	ID              string             `json:"id"`
	ProjectID       int                `json:"project_id"`
	Name            string             `json:"name"`
	Description     *string            `json:"description"`
	TargetType      string             `json:"target_type"`
	TargetID        string             `json:"target_id"`
	MetricType      string             `json:"metric_type"`
	Operator        string             `json:"operator"`
	Threshold       float64            `json:"threshold"`
	DurationSeconds int                `json:"duration_seconds"`
	CooldownSeconds int                `json:"cooldown_seconds"`
	Status          string             `json:"status"`
	LastTriggeredAt *string            `json:"last_triggered_at"`
	LastResolvedAt  *string            `json:"last_resolved_at"`
	CreatedAt       string             `json:"created_at"`
	UpdatedAt       string             `json:"updated_at"`
	Actions         []CloudAlertAction `json:"actions"`
}

// CloudAlertActionRequest describes an action of an alert. NotificationChannelID is required for
// "notify"; Config is required for "create_vps" (name, template_name, plan_name and
// location_name).
type CloudAlertActionRequest struct {
	ActionType            string                 `json:"action_type"`
	NotificationChannelID string                 `json:"notificator_id,omitempty"`
	Config                map[string]interface{} `json:"config,omitempty"`
	Order                 int                    `json:"order"`
	Enabled               *bool                  `json:"enabled,omitempty"`
}

// CreateCloudAlertRequest represents a request to create an alert. TargetID is the VPS or
// baremetal id, or the availability group uuid. DurationSeconds defaults to 300 and
// CooldownSeconds to 600 when 0.
type CreateCloudAlertRequest struct {
	ProjectID       int                       `json:"project_id"`
	Name            string                    `json:"name"`
	Description     string                    `json:"description,omitempty"`
	TargetType      string                    `json:"target_type"`
	TargetID        string                    `json:"target_id"`
	MetricType      string                    `json:"metric_type"`
	Operator        string                    `json:"operator"`
	Threshold       float64                   `json:"threshold"`
	DurationSeconds int                       `json:"duration_seconds,omitempty"`
	CooldownSeconds int                       `json:"cooldown_seconds,omitempty"`
	Actions         []CloudAlertActionRequest `json:"actions"`
}

// UpdateCloudAlertRequest represents a request to update an alert.
type UpdateCloudAlertRequest struct {
	Name            *string                    `json:"name,omitempty"`
	Description     *string                    `json:"description,omitempty"`
	TargetType      *string                    `json:"target_type,omitempty"`
	TargetID        *string                    `json:"target_id,omitempty"`
	MetricType      *string                    `json:"metric_type,omitempty"`
	Operator        *string                    `json:"operator,omitempty"`
	Threshold       *float64                   `json:"threshold,omitempty"`
	DurationSeconds *int                       `json:"duration_seconds,omitempty"`
	CooldownSeconds *int                       `json:"cooldown_seconds,omitempty"`
	Status          *string                    `json:"status,omitempty"`
	Actions         *[]CloudAlertActionRequest `json:"actions,omitempty"`
}

// CloudAlertEvent is a fired or resolved event of an alert.
type CloudAlertEvent struct {
	ID          string                 `json:"id"`
	TriggerID   string                 `json:"trigger_id"`
	EventType   string                 `json:"event_type"`
	MetricValue *float64               `json:"metric_value"`
	Details     map[string]interface{} `json:"details"`
	CreatedAt   string                 `json:"created_at"`
}

// NotificationChannel is a destination for alert notifications. Webhook URLs in Config are
// masked in responses.
type NotificationChannel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// CreateNotificationChannelRequest represents a request to create a notification channel.
// Slack and Discord need an HTTPS "webhook_url" in Config; email needs no settings (the
// organization's email is used). Enabled defaults to true.
type CreateNotificationChannelRequest struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"`
}

// UpdateNotificationChannelRequest represents a request to update a notification channel; nil
// fields are left unchanged.
type UpdateNotificationChannelRequest struct {
	Name    *string                `json:"name,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled *bool                  `json:"enabled,omitempty"`
}

type cloudAlertService struct {
	client *Client
}

func (s *cloudAlertService) List(ctx context.Context, opts *CloudAlertListOptions) ([]CloudAlertSummary, error) {
	path := "/triggers/"
	if opts != nil {
		v := url.Values{}
		if opts.ProjectID > 0 {
			v.Set("project_id", strconv.Itoa(opts.ProjectID))
		}
		if opts.Status != "" {
			v.Set("status", opts.Status)
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}
	var alerts []CloudAlertSummary
	if err := s.client.get(ctx, path, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (s *cloudAlertService) Get(ctx context.Context, alertID string) (*CloudAlert, error) {
	var alert CloudAlert
	if err := s.client.get(ctx, "/triggers/"+url.PathEscape(alertID), &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *cloudAlertService) Create(ctx context.Context, req *CreateCloudAlertRequest) (*CloudAlert, error) {
	var alert CloudAlert
	if err := s.client.post(ctx, "/triggers/", req, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *cloudAlertService) Update(ctx context.Context, alertID string, req *UpdateCloudAlertRequest) (*CloudAlert, error) {
	var alert CloudAlert
	if err := s.client.put(ctx, "/triggers/"+url.PathEscape(alertID), req, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

func (s *cloudAlertService) Delete(ctx context.Context, alertID string) error {
	return s.client.del(ctx, "/triggers/"+url.PathEscape(alertID))
}

func (s *cloudAlertService) History(ctx context.Context, alertID string, limit int) ([]CloudAlertEvent, error) {
	path := fmt.Sprintf("/triggers/%s/history", url.PathEscape(alertID))
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var events []CloudAlertEvent
	if err := s.client.get(ctx, path, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *cloudAlertService) ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error) {
	var channels []NotificationChannel
	if err := s.client.get(ctx, "/triggers/notificators/", &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func (s *cloudAlertService) GetNotificationChannel(ctx context.Context, channelID string) (*NotificationChannel, error) {
	var channel NotificationChannel
	if err := s.client.get(ctx, "/triggers/notificators/"+url.PathEscape(channelID), &channel); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *cloudAlertService) CreateNotificationChannel(ctx context.Context, req *CreateNotificationChannelRequest) (*NotificationChannel, error) {
	var channel NotificationChannel
	if err := s.client.post(ctx, "/triggers/notificators/", req, &channel); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *cloudAlertService) UpdateNotificationChannel(ctx context.Context, channelID string, req *UpdateNotificationChannelRequest) (*NotificationChannel, error) {
	var channel NotificationChannel
	if err := s.client.put(ctx, "/triggers/notificators/"+url.PathEscape(channelID), req, &channel); err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *cloudAlertService) DeleteNotificationChannel(ctx context.Context, channelID string) error {
	return s.client.del(ctx, "/triggers/notificators/"+url.PathEscape(channelID))
}
