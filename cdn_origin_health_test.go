package cubepath

import (
	"context"
	"testing"
)

func TestCreateOriginHealthCheckFields(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"o1","health_check_type":"http","health_check_method":"HEAD","health_check_expected_status":"200","health_check_interval":15,"health_check_threshold":2,"health_status":"unhealthy","health_detail":"connection refused","health_changed_at":"2026-11-25T10:00:00Z"}`, &rec)
	o, err := c.CDN.CreateOrigin(context.Background(), "z1", &CreateCDNOriginRequest{
		Name: "api", Address: "api.example.com", HealthCheckEnabled: true,
		HealthCheckType: "http", HealthCheckMethod: "HEAD", HealthCheckExpectedStatus: "200",
		HealthCheckInterval: 15, HealthCheckThreshold: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]interface{}{
		"health_check_type": "http", "health_check_method": "HEAD", "health_check_expected_status": "200",
		"health_check_interval": float64(15), "health_check_threshold": float64(2),
	} {
		if rec.Body[k] != want {
			t.Errorf("%s = %v, want %v", k, rec.Body[k], want)
		}
	}
	if _, ok := rec.Body["health_check_path"]; ok {
		t.Error("empty health_check_path must be omitted")
	}
	if o.HealthCheckInterval != 15 || o.HealthCheckType != "http" || o.HealthDetail == nil || *o.HealthDetail != "connection refused" ||
		o.HealthChangedAt == nil || *o.HealthChangedAt != "2026-11-25T10:00:00Z" {
		t.Errorf("unexpected origin: %+v", o)
	}
}

func TestCreateOriginOmitsUnsetHealthCheckFields(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"o1"}`, &rec)
	if _, err := c.CDN.CreateOrigin(context.Background(), "z1", &CreateCDNOriginRequest{Name: "a", Address: "a.example.com"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"health_check_type", "health_check_method", "health_check_expected_status", "health_check_interval", "health_check_threshold", "health_check_path"} {
		if _, ok := rec.Body[k]; ok {
			t.Errorf("%s must be omitted when unset", k)
		}
	}
}

func TestUpdateOriginHealthCheckFieldsOptional(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"o1"}`, &rec)
	typ, iv := "http", 30
	if _, err := c.CDN.UpdateOrigin(context.Background(), "z1", "o1", &UpdateCDNOriginRequest{HealthCheckType: &typ, HealthCheckInterval: &iv}); err != nil {
		t.Fatal(err)
	}
	if len(rec.Body) != 2 || rec.Body["health_check_type"] != "http" || rec.Body["health_check_interval"] != float64(30) {
		t.Errorf("unexpected body: %v", rec.Body)
	}
}
