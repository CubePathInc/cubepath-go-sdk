package cubepath

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

const (
	eventVectorSecret    = "whsec_0123456789abcdefABCDEF0123456789"
	eventVectorTimestamp = "1790000000"
	eventVectorBody      = `{"id":"evt_01","type":"object.created"}`
	eventVectorSig       = "348e719a6c9fb75f1c6a60cc81ec5914a599eb648b1ab9d81e483fc2309c6f3b"
	eventVectorPrevSig   = "d1cc88ad9e8ba98e234697ea257b0969a065c04f879a1c9cdcab90ef316a953d"
)

func TestObjectStorageCreateEventDestination(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"destination":{"uuid":"d1","name":"hook","type":"webhook","url_masked":"https://example.com/***","payload_format":"cubepath","status":"active","rules_count":0},"signing_secret":"whsec_abc"}`, &rec)
	u := "https://example.com/hook"
	out, err := c.ObjectStorage.CreateEventDestination(context.Background(), &CreateObjectStorageEventDestinationRequest{Name: "hook", Type: "webhook", URL: &u})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/event-destinations" || rec.Body["url"] != u || rec.Body["type"] != "webhook" {
		t.Fatalf("got %+v", rec)
	}
	if _, ok := rec.Body["notificator_id"]; ok {
		t.Fatalf("notificator_id sent: %+v", rec.Body)
	}
	if out.SigningSecret == nil || *out.SigningSecret != "whsec_abc" || out.Destination.UUID != "d1" || *out.Destination.URLMasked != "https://example.com/***" {
		t.Fatalf("decoded %+v", out)
	}
}

func TestObjectStorageEventDestinationRoutes(t *testing.T) {
	ctx := context.Background()
	f := false
	cases := []struct {
		name, resp, method, path, query string
		call                            func(c *Client) error
	}{
		{"list", `[]`, "GET", "/object-storage/event-destinations", "", func(c *Client) error { _, err := c.ObjectStorage.ListEventDestinations(ctx); return err }},
		{"get", `{"uuid":"d1"}`, "GET", "/object-storage/event-destinations/d1", "", func(c *Client) error { _, err := c.ObjectStorage.GetEventDestination(ctx, "d1"); return err }},
		{"update", `{"uuid":"d1"}`, "PATCH", "/object-storage/event-destinations/d1", "", func(c *Client) error {
			_, err := c.ObjectStorage.UpdateEventDestination(ctx, "d1", &UpdateObjectStorageEventDestinationRequest{Enabled: &f})
			return err
		}},
		{"delete", ``, "DELETE", "/object-storage/event-destinations/d1", "", func(c *Client) error { return c.ObjectStorage.DeleteEventDestination(ctx, "d1") }},
		{"rotate", `{"destination":{"uuid":"d1"},"signing_secret":"whsec_new"}`, "POST", "/object-storage/event-destinations/d1/rotate-secret", "", func(c *Client) error {
			_, err := c.ObjectStorage.RotateEventDestinationSecret(ctx, "d1")
			return err
		}},
		{"test", `{"detail":"Test event sent. Check the delivery history for the result."}`, "POST", "/object-storage/event-destinations/d1/test", "", func(c *Client) error { return c.ObjectStorage.TestEventDestination(ctx, "d1") }},
		{"deliveries", `{"deliveries":[],"next_before":null}`, "GET", "/object-storage/event-destinations/d1/deliveries", "before=1790964001250&limit=10&status=failed", func(c *Client) error {
			_, err := c.ObjectStorage.ListEventDeliveries(ctx, "d1", &ObjectStorageEventDeliveryOptions{Status: "failed", Limit: 10, Before: 1790964001250})
			return err
		}},
		{"rules", `[]`, "GET", "/object-storage/buckets/b1/event-rules", "", func(c *Client) error { _, err := c.ObjectStorage.ListEventRules(ctx, "b1"); return err }},
		{"rule update", `{"uuid":"r1"}`, "PATCH", "/object-storage/buckets/b1/event-rules/r1", "", func(c *Client) error {
			_, err := c.ObjectStorage.UpdateEventRule(ctx, "b1", "r1", &UpdateObjectStorageEventRuleRequest{Enabled: &f})
			return err
		}},
		{"rule delete", ``, "DELETE", "/object-storage/buckets/b1/event-rules/r1", "", func(c *Client) error { return c.ObjectStorage.DeleteEventRule(ctx, "b1", "r1") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec recorded
			status := 200
			if tc.resp == "" {
				status = 204
			}
			c := newTestClient(t, status, tc.resp, &rec)
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			if rec.Method != tc.method || rec.Path != tc.path || rec.Query != tc.query {
				t.Fatalf("got %+v", rec)
			}
		})
	}
}

func TestObjectStorageCreateEventRule(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"r1","name":"uploads","bucket_uuid":"b1","destination":{"uuid":"d1","name":"hook","type":"webhook"},"events":["object.created"],"prefix":"in/","suffix":"","enabled":true,"status":"pending"}`, &rec)
	r, err := c.ObjectStorage.CreateEventRule(context.Background(), "b1", &CreateObjectStorageEventRuleRequest{
		Name: "uploads", DestinationUUID: "d1", Events: []string{ObjectStorageEventCreated}, Prefix: "in/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/buckets/b1/event-rules" || rec.Body["destination_uuid"] != "d1" || rec.Body["prefix"] != "in/" {
		t.Fatalf("got %+v", rec)
	}
	if r.Status != "pending" || r.Destination.UUID != "d1" || len(r.Events) != 1 {
		t.Fatalf("decoded %+v", r)
	}
}

func TestVerifyStorageEventSignature(t *testing.T) {
	now := time.Unix(1790000000, 0)
	body := []byte(eventVectorBody)
	ok := []string{
		"v1=" + eventVectorSig,
		"v1=" + eventVectorPrevSig + ",v1=" + eventVectorSig,
		"v1=" + eventVectorSig + " v1=" + eventVectorPrevSig,
		// The exact form the service sends during a rotation.
		"v1=" + eventVectorSig + ", v1=" + eventVectorPrevSig,
	}
	for _, h := range ok {
		if err := verifyStorageEventSignatureAt(eventVectorSecret, eventVectorTimestamp, body, h, 0, now); err != nil {
			t.Fatalf("%q: %v", h, err)
		}
	}
	if err := verifyStorageEventSignatureAt("whsec_previous", eventVectorTimestamp, body, "v1="+eventVectorSig+", v1="+eventVectorPrevSig, 0, now); err != nil {
		t.Fatalf("previous secret: %v", err)
	}
	bad := []struct {
		name, ts, body, header string
		now                    time.Time
	}{
		{"tampered body", eventVectorTimestamp, `{"id":"evt_02"}`, "v1=" + eventVectorSig, now},
		{"wrong secret", eventVectorTimestamp, eventVectorBody, "v1=" + eventVectorPrevSig, now},
		{"unknown scheme", eventVectorTimestamp, eventVectorBody, "v0=" + eventVectorSig, now},
		{"expired", eventVectorTimestamp, eventVectorBody, "v1=" + eventVectorSig, now.Add(6 * time.Minute)},
		{"future", eventVectorTimestamp, eventVectorBody, "v1=" + eventVectorSig, now.Add(-6 * time.Minute)},
		{"bad timestamp", "abc", eventVectorBody, "v1=" + eventVectorSig, now},
		{"empty header", eventVectorTimestamp, eventVectorBody, "", now},
	}
	for _, b := range bad {
		err := verifyStorageEventSignatureAt(eventVectorSecret, b.ts, []byte(b.body), b.header, 0, b.now)
		if !errors.Is(err, ErrStorageEventSignature) {
			t.Fatalf("%s: got %v", b.name, err)
		}
	}
	if err := verifyStorageEventSignatureAt(eventVectorSecret, eventVectorTimestamp, body, "v1="+eventVectorSig, -1, now.Add(time.Hour)); err != nil {
		t.Fatalf("negative tolerance must skip the timestamp check: %v", err)
	}
	if err := VerifyStorageEventSignature(eventVectorSecret, eventVectorTimestamp, body, "v1="+eventVectorSig, 0); !errors.Is(err, ErrStorageEventSignature) {
		t.Fatalf("old vector must be expired against the clock: %v", err)
	}
}

func TestObjectStorageListEventDeliveriesPage(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"deliveries":[{"ts":"2026-10-02T18:00:01.250","ts_ms":1790964001250,"event_id":"evt_1","delivery_id":"dlv_1","event_type":"object.created","bucket_uuid":"b1","bucket_name":"photos","rule_uuid":"r1","object_key":"uploads/a.jpg","attempt":1,"status":"success","http_status":200,"latency_ms":84,"error":""}],"next_before":1790964001250}`, &rec)
	page, err := c.ObjectStorage.ListEventDeliveries(context.Background(), "d1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Query != "" || len(page.Deliveries) != 1 || page.NextBefore == nil || *page.NextBefore != 1790964001250 {
		t.Fatalf("got %+v %+v", rec, page)
	}
	d := page.Deliveries[0]
	if d.BucketName == nil || *d.BucketName != "photos" || d.HTTPStatus != 200 || d.LatencyMs != 84 || d.TSMs != 1790964001250 {
		t.Fatalf("decoded %+v", d)
	}
}
