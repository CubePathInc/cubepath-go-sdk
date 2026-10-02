package cubepath

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Event types a bucket event rule can subscribe to.
const (
	ObjectStorageEventCreated = "object.created"
	ObjectStorageEventRemoved = "object.removed"
	ObjectStorageEventTagging = "object.tagging"
)

// ObjectStorageEventDestinationChannel is the Cloud Alerts channel (Slack or Discord) of a
// destination of type "notificator".
type ObjectStorageEventDestinationChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ObjectStorageEventDestination is where bucket events are delivered: a signed webhook or a
// Cloud Alerts channel. The webhook URL is never returned in clear (URLMasked).
type ObjectStorageEventDestination struct {
	UUID           string                                `json:"uuid"`
	Name           string                                `json:"name"`
	Type           string                                `json:"type"`
	URLMasked      *string                               `json:"url_masked"`
	Notificator    *ObjectStorageEventDestinationChannel `json:"notificator"`
	PayloadFormat  string                                `json:"payload_format"`
	Status         string                                `json:"status"`
	DisabledReason *string                               `json:"disabled_reason"`
	LastSuccessAt  *string                               `json:"last_success_at"`
	LastFailureAt  *string                               `json:"last_failure_at"`
	LastError      *string                               `json:"last_error"`
	RulesCount     int                                   `json:"rules_count"`
}

// ObjectStorageEventDestinationSecret is the answer of a create or a secret rotation: the
// signing secret is only returned by these two calls (null for a channel destination).
type ObjectStorageEventDestinationSecret struct {
	Destination   ObjectStorageEventDestination `json:"destination"`
	SigningSecret *string                       `json:"signing_secret"`
}

// CreateObjectStorageEventDestinationRequest creates a destination: Type "webhook" with URL
// (https on port 443 or 8443) or Type "notificator" with NotificatorID (a Slack or Discord
// channel of Cloud Alerts). PayloadFormat is "cubepath" (default) or "s3".
type CreateObjectStorageEventDestinationRequest struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	URL           *string `json:"url,omitempty"`
	NotificatorID *string `json:"notificator_id,omitempty"`
	PayloadFormat string  `json:"payload_format,omitempty"`
}

// UpdateObjectStorageEventDestinationRequest changes a destination; nil fields are left as they are.
type UpdateObjectStorageEventDestinationRequest struct {
	Name          *string `json:"name,omitempty"`
	URL           *string `json:"url,omitempty"`
	PayloadFormat *string `json:"payload_format,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
}

// ObjectStorageEventDelivery is one delivery of an event to a destination.
type ObjectStorageEventDelivery map[string]interface{}

// ObjectStorageEventDeliveryOptions filters the delivery history: Status success, failed or
// dead; Limit (default 50); Before, a timestamp to page back from.
type ObjectStorageEventDeliveryOptions struct {
	Status string
	Limit  int
	Before string
}

// ObjectStorageEventRuleDestination identifies the destination of a rule.
type ObjectStorageEventRuleDestination struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ObjectStorageEventRule sends a bucket's events of the listed types (object.created,
// object.removed, object.tagging) whose key matches Prefix and Suffix to a destination. Rules are
// applied asynchronously: Status goes from "pending" to "active".
type ObjectStorageEventRule struct {
	UUID         string                            `json:"uuid"`
	Name         string                            `json:"name"`
	BucketUUID   string                            `json:"bucket_uuid"`
	Destination  ObjectStorageEventRuleDestination `json:"destination"`
	Events       []string                          `json:"events"`
	Prefix       string                            `json:"prefix"`
	Suffix       string                            `json:"suffix"`
	Enabled      bool                              `json:"enabled"`
	Status       string                            `json:"status"`
	ErrorMessage *string                           `json:"error_message"`
}

// CreateObjectStorageEventRuleRequest creates a rule on a bucket. Enabled defaults to true.
type CreateObjectStorageEventRuleRequest struct {
	Name            string   `json:"name"`
	DestinationUUID string   `json:"destination_uuid"`
	Events          []string `json:"events"`
	Prefix          string   `json:"prefix,omitempty"`
	Suffix          string   `json:"suffix,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
}

// UpdateObjectStorageEventRuleRequest changes a rule; nil fields are left as they are.
type UpdateObjectStorageEventRuleRequest struct {
	Name            *string   `json:"name,omitempty"`
	DestinationUUID *string   `json:"destination_uuid,omitempty"`
	Events          *[]string `json:"events,omitempty"`
	Prefix          *string   `json:"prefix,omitempty"`
	Suffix          *string   `json:"suffix,omitempty"`
	Enabled         *bool     `json:"enabled,omitempty"`
}

func eventDestinationPath(uuid string) string {
	return "/object-storage/event-destinations/" + url.PathEscape(uuid)
}

func eventRulesPath(bucketUUID string) string {
	return fmt.Sprintf("/object-storage/buckets/%s/event-rules", url.PathEscape(bucketUUID))
}

func (s *objectStorageService) ListEventDestinations(ctx context.Context) ([]ObjectStorageEventDestination, error) {
	var out []ObjectStorageEventDestination
	if err := s.client.get(ctx, "/object-storage/event-destinations", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *objectStorageService) CreateEventDestination(ctx context.Context, req *CreateObjectStorageEventDestinationRequest) (*ObjectStorageEventDestinationSecret, error) {
	var out ObjectStorageEventDestinationSecret
	if err := s.client.post(ctx, "/object-storage/event-destinations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) GetEventDestination(ctx context.Context, uuid string) (*ObjectStorageEventDestination, error) {
	var out ObjectStorageEventDestination
	if err := s.client.get(ctx, eventDestinationPath(uuid), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) UpdateEventDestination(ctx context.Context, uuid string, req *UpdateObjectStorageEventDestinationRequest) (*ObjectStorageEventDestination, error) {
	var out ObjectStorageEventDestination
	if err := s.client.patch(ctx, eventDestinationPath(uuid), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) DeleteEventDestination(ctx context.Context, uuid string) error {
	return s.client.del(ctx, eventDestinationPath(uuid))
}

func (s *objectStorageService) RotateEventDestinationSecret(ctx context.Context, uuid string) (*ObjectStorageEventDestinationSecret, error) {
	var out ObjectStorageEventDestinationSecret
	if err := s.client.post(ctx, eventDestinationPath(uuid)+"/rotate-secret", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) TestEventDestination(ctx context.Context, uuid string) error {
	return s.client.post(ctx, eventDestinationPath(uuid)+"/test", nil, nil)
}

func (s *objectStorageService) ListEventDeliveries(ctx context.Context, uuid string, opts *ObjectStorageEventDeliveryOptions) ([]ObjectStorageEventDelivery, error) {
	path := eventDestinationPath(uuid) + "/deliveries"
	if opts != nil {
		v := url.Values{}
		if opts.Status != "" {
			v.Set("status", opts.Status)
		}
		if opts.Limit > 0 {
			v.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Before != "" {
			v.Set("before", opts.Before)
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}
	var out []ObjectStorageEventDelivery
	if err := s.client.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *objectStorageService) ListEventRules(ctx context.Context, bucketUUID string) ([]ObjectStorageEventRule, error) {
	var out []ObjectStorageEventRule
	if err := s.client.get(ctx, eventRulesPath(bucketUUID), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *objectStorageService) CreateEventRule(ctx context.Context, bucketUUID string, req *CreateObjectStorageEventRuleRequest) (*ObjectStorageEventRule, error) {
	var out ObjectStorageEventRule
	if err := s.client.post(ctx, eventRulesPath(bucketUUID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) UpdateEventRule(ctx context.Context, bucketUUID, ruleUUID string, req *UpdateObjectStorageEventRuleRequest) (*ObjectStorageEventRule, error) {
	var out ObjectStorageEventRule
	if err := s.client.patch(ctx, eventRulesPath(bucketUUID)+"/"+url.PathEscape(ruleUUID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *objectStorageService) DeleteEventRule(ctx context.Context, bucketUUID, ruleUUID string) error {
	return s.client.del(ctx, eventRulesPath(bucketUUID)+"/"+url.PathEscape(ruleUUID))
}

// ErrStorageEventSignature is returned by VerifyStorageEventSignature when no signature of the
// header matches the body, or the timestamp is outside the tolerance.
var ErrStorageEventSignature = errors.New("cubepath: storage event signature mismatch")

// DefaultStorageEventTolerance is how far the CubePath-Timestamp of a delivery may be from now.
const DefaultStorageEventTolerance = 5 * time.Minute

// VerifyStorageEventSignature checks a storage event webhook delivery: secret is the signing
// secret of the destination, timestamp the CubePath-Timestamp header (unix seconds), body the raw
// request body and header the CubePath-Signature header ("v1=<hex>", several during a secret
// rotation, separated by commas or spaces). The signature is the hex HMAC-SHA256 of
// timestamp + "." + body. A tolerance of 0 uses DefaultStorageEventTolerance; a negative one
// skips the timestamp check.
func VerifyStorageEventSignature(secret, timestamp string, body []byte, header string, tolerance time.Duration) error {
	return verifyStorageEventSignatureAt(secret, timestamp, body, header, tolerance, time.Now())
}

func verifyStorageEventSignatureAt(secret, timestamp string, body []byte, header string, tolerance time.Duration, now time.Time) error {
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return ErrStorageEventSignature
	}
	if tolerance == 0 {
		tolerance = DefaultStorageEventTolerance
	}
	if tolerance > 0 && math.Abs(float64(now.Unix()-ts)) > tolerance.Seconds() {
		return ErrStorageEventSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.TrimSpace(timestamp) + "."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, part := range strings.FieldsFunc(header, func(r rune) bool { return r == ',' || r == ' ' }) {
		value, ok := strings.CutPrefix(part, "v1=")
		if !ok {
			continue
		}
		got, err := hex.DecodeString(value)
		if err == nil && hmac.Equal(got, expected) {
			return nil
		}
	}
	return ErrStorageEventSignature
}
