package cubepath

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

const replicationJSON = `{"uuid":"r1","status":"active","pause_reason":null,"direction":"outgoing",
"source":{"bucket_uuid":"b1","bucket_name":"photos","project_id":12,"organization_name":"Acme","same_organization":true},
"destination":{"type":"external","provider":"aws","endpoint":"s3.eu-west-1.amazonaws.com","region":"eu-west-1","bucket":"acme-backup","path_style":"auto","access_key_id":"****WXYZ"},
"rules":{"enabled":true,"prefix":"img/","tags":[],"delete_marker_replication":false,"delete_replication":false,"existing_objects":true},
"health":"ok","health_reason":null,"health_checked_at":"2026-10-02T10:00:00",
"backfill":{"status":"completed","started_at":"2026-10-02T09:00:00","finished_at":"2026-10-02T09:40:00","objects":120034,"bytes":53687091200,"failed_objects":0},
"error_message":null,"created_at":"2026-10-02T08:59:30","active_at":"2026-10-02T09:00:05"`

func TestObjectStorageListReplicationsQuery(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[`+replicationJSON+`}]`, &rec)
	reps, err := c.ObjectStorage.ListReplications(context.Background(), &ObjectStorageReplicationListOptions{Direction: "outgoing", BucketUUID: "b1"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/object-storage/replications" || rec.Query != "bucket_uuid=b1&direction=outgoing" {
		t.Fatalf("got %+v", rec)
	}
	r := reps[0]
	if len(reps) != 1 || r.Destination.Type != "external" || *r.Destination.AccessKeyID != "****WXYZ" ||
		*r.Rules.Prefix != "img/" || r.Backfill.Bytes != 53687091200 || *r.Source.ProjectID != 12 || r.Health != "ok" {
		t.Fatalf("decoded %+v", r)
	}

	if _, err := c.ObjectStorage.ListReplications(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if rec.Query != "" {
		t.Fatalf("query without options %q", rec.Query)
	}
}

func TestObjectStorageGetReplicationWithMetrics(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, replicationJSON+`,"metrics":{"replicated_bytes_24h":10,"queued_objects":3,"egress_bytes_month":null}}`, &rec)
	r, err := c.ObjectStorage.GetReplication(context.Background(), "r1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/object-storage/replications/r1" {
		t.Fatalf("got %+v", rec)
	}
	if r.UUID != "r1" || r.Metrics == nil || *r.Metrics.ReplicatedBytes24h != 10 || *r.Metrics.QueuedObjects != 3 || r.Metrics.EgressBytesMonth != nil {
		t.Fatalf("decoded %+v", r)
	}
}

func TestObjectStorageCreateReplicationCubepath(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"detail":"Replication is being configured","uuid":"r1","status":"pending"}`, &rec)
	res, err := c.ObjectStorage.CreateReplication(context.Background(), &CreateObjectStorageReplicationRequest{
		SourceBucketUUID: "b1",
		Destination:      ObjectStorageReplicationDestinationRequest{Type: "cubepath", BucketUUID: "b2", GrantToken: "cprg_x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/replications" {
		t.Fatalf("got %+v", rec)
	}
	dest, _ := rec.Body["destination"].(map[string]interface{})
	if rec.Body["source_bucket_uuid"] != "b1" || dest["type"] != "cubepath" || dest["bucket_uuid"] != "b2" || dest["grant_token"] != "cprg_x" {
		t.Fatalf("body %v", rec.Body)
	}
	for _, k := range []string{"endpoint", "region", "bucket", "access_key_id", "secret_access_key", "provider", "path_style"} {
		if _, ok := dest[k]; ok {
			t.Fatalf("external field %s sent for a CubePath destination: %v", k, dest)
		}
	}
	for _, k := range []string{"prefix", "tags", "existing_objects"} {
		if _, ok := rec.Body[k]; ok {
			t.Fatalf("%s sent when not set: %v", k, rec.Body)
		}
	}
	if res.UUID != "r1" || res.Status != "pending" {
		t.Fatalf("decoded %+v", res)
	}
}

func TestObjectStorageCreateReplicationExternal(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"detail":"Replication is being configured","uuid":"r1","status":"pending"}`, &rec)
	no := false
	_, err := c.ObjectStorage.CreateReplication(context.Background(), &CreateObjectStorageReplicationRequest{
		SourceBucketUUID: "b1",
		Destination: ObjectStorageReplicationDestinationRequest{Type: "external", Provider: "aws", Endpoint: "s3.eu-west-1.amazonaws.com",
			Region: "eu-west-1", Bucket: "acme-backup", AccessKeyID: "AKIA", SecretAccessKey: "s3cr3t"},
		Tags:              []ObjectStorageReplicationTag{{Key: "backup", Value: "yes"}},
		DeleteReplication: true,
		ExistingObjects:   &no,
	})
	if err != nil {
		t.Fatal(err)
	}
	dest, _ := rec.Body["destination"].(map[string]interface{})
	if dest["endpoint"] != "s3.eu-west-1.amazonaws.com" || dest["secret_access_key"] != "s3cr3t" || dest["region"] != "eu-west-1" {
		t.Fatalf("destination %v", dest)
	}
	if _, ok := dest["bucket_uuid"]; ok {
		t.Fatalf("bucket_uuid sent for an external destination: %v", dest)
	}
	tags, _ := rec.Body["tags"].([]interface{})
	if len(tags) != 1 || rec.Body["delete_replication"] != true || rec.Body["existing_objects"] != false {
		t.Fatalf("body %v", rec.Body)
	}
}

func TestObjectStorageUpdateReplicationBody(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"Replication updated"}`, &rec)
	off := false
	err := c.ObjectStorage.UpdateReplication(context.Background(), "r1", &UpdateObjectStorageReplicationRequest{
		Enabled: &off, RemovePrefix: true, Tags: []ObjectStorageReplicationTag{{Key: "k", Value: "v"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPatch || rec.Path != "/object-storage/replications/r1" {
		t.Fatalf("got %+v", rec)
	}
	if v, ok := rec.Body["prefix"]; !ok || v != nil {
		t.Fatalf("prefix should be an explicit null: %v", rec.Body)
	}
	if rec.Body["enabled"] != false || len(rec.Body) != 3 {
		t.Fatalf("body %v", rec.Body)
	}

	if err := c.ObjectStorage.UpdateReplication(context.Background(), "r1", &UpdateObjectStorageReplicationRequest{
		RemoveTags:  true,
		Destination: &ObjectStorageReplicationCredentials{AccessKeyID: "AK2", SecretAccessKey: "S2"},
	}); err != nil {
		t.Fatal(err)
	}
	if v, ok := rec.Body["tags"]; !ok || v != nil {
		t.Fatalf("tags should be an explicit null: %v", rec.Body)
	}
	dest, _ := rec.Body["destination"].(map[string]interface{})
	if dest["access_key_id"] != "AK2" || dest["secret_access_key"] != "S2" || len(rec.Body) != 2 {
		t.Fatalf("body %v", rec.Body)
	}
}

func TestUpdateObjectStorageReplicationRequestOmitsUnset(t *testing.T) {
	b, err := json.Marshal(&UpdateObjectStorageReplicationRequest{})
	if err != nil || string(b) != "{}" {
		t.Fatalf("got %s (%v)", b, err)
	}
	p := "img/"
	b, _ = json.Marshal(UpdateObjectStorageReplicationRequest{Prefix: &p})
	if string(b) != `{"prefix":"img/"}` {
		t.Fatalf("got %s", b)
	}
}

func TestObjectStorageReplicationActions(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"ok"}`, &rec)
	ctx := context.Background()

	if err := c.ObjectStorage.DeleteReplication(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/object-storage/replications/r1" {
		t.Fatalf("delete %+v", rec)
	}

	days := 30
	if err := c.ObjectStorage.ResyncReplication(ctx, "r1", &days); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/replications/r1/resync" || rec.Body["older_than_days"] != float64(30) {
		t.Fatalf("resync %+v", rec)
	}
	if err := c.ObjectStorage.ResyncReplication(ctx, "r1", nil); err != nil {
		t.Fatal(err)
	}
	if v, ok := rec.Body["older_than_days"]; !ok || v != nil {
		t.Fatalf("resync everything %+v", rec)
	}

	if err := c.ObjectStorage.RevokeReplication(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/replications/r1/revoke" {
		t.Fatalf("revoke %+v", rec)
	}
}

func TestObjectStorageReplicationGrants(t *testing.T) {
	var rec recorded
	ctx := context.Background()
	c := newTestClient(t, 201, `{"detail":"Replication grant created. Share the token now: it will not be shown again.","uuid":"g1","token":"cprg_AbCdEfGhIjKlMnOpQrStUvWxYz012345","token_prefix":"cprg_AbCd","bucket_uuid":"b2","note":"for Acme","expires_at":"2026-10-09T10:00:00"}`, &rec)
	note := "for Acme"
	g, err := c.ObjectStorage.CreateReplicationGrant(ctx, "b2", &CreateObjectStorageReplicationGrantRequest{Note: &note, ExpiresInDays: 3})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/buckets/b2/replication-grants" || rec.Body["note"] != "for Acme" || rec.Body["expires_in_days"] != float64(3) {
		t.Fatalf("create %+v", rec)
	}
	if !strings.HasPrefix(g.Token, "cprg_") || g.TokenPrefix != "cprg_AbCd" || g.BucketUUID != "b2" {
		t.Fatalf("decoded %+v", g)
	}

	if _, err := c.ObjectStorage.CreateReplicationGrant(ctx, "b2", nil); err != nil {
		t.Fatal(err)
	}
	if len(rec.Body) != 0 {
		t.Fatalf("default grant body %v", rec.Body)
	}

	c = newTestClient(t, 200, `[{"uuid":"g1","token_prefix":"cprg_AbCd","note":null,"status":"open","expires_at":"2026-10-09T10:00:00","used_at":null,"revoked_at":null,"created_at":"2026-10-02T10:00:00"}]`, &rec)
	grants, err := c.ObjectStorage.ListReplicationGrants(ctx, "b2")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/object-storage/buckets/b2/replication-grants" || len(grants) != 1 || grants[0].Status != "open" {
		t.Fatalf("list %+v %+v", rec, grants)
	}

	c = newTestClient(t, 200, `{"detail":"Replication grant revoked"}`, &rec)
	if err := c.ObjectStorage.DeleteReplicationGrant(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/object-storage/replication-grants/g1" {
		t.Fatalf("delete %+v", rec)
	}
}

func TestObjectStorageBucketDetailReplication(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"b1","name":"photos","status":"active","connection":{},"replication":{"outgoing":`+replicationJSON+`},"incoming_count":2}}`, &rec)
	b, err := c.ObjectStorage.GetBucket(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if b.Replication == nil || b.Replication.Outgoing == nil || b.Replication.Outgoing.UUID != "r1" || b.Replication.IncomingCount != 2 {
		t.Fatalf("decoded %+v", b.Replication)
	}
}
