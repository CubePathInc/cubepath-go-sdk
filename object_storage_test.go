package cubepath

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recorded struct {
	Method string
	Path   string
	Query  string
	Body   map[string]interface{}
}

func newTestClient(t *testing.T, status int, response string, rec *recorded) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Method, rec.Path, rec.Query = r.Method, r.URL.Path, r.URL.RawQuery
		rec.Body = nil
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			if err := json.Unmarshal(b, &rec.Body); err != nil {
				t.Fatalf("request body is not JSON: %s", b)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient("token", WithBaseURL(srv.URL), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestObjectStorageListBucketsQuery(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"uuid":"b1","name":"photos","status":"active","tier":{"slug":"infrequent_access"},"size_bytes":42,"cdn_connected":true}]`, &rec)
	buckets, err := c.ObjectStorage.ListBuckets(context.Background(), &ObjectStorageListOptions{ProjectID: 12, Tier: "infrequent_access"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodGet || rec.Path != "/object-storage/buckets" || rec.Query != "project_id=12&tier=infrequent_access" {
		t.Fatalf("got %+v", rec)
	}
	if len(buckets) != 1 || buckets[0].Tier.Slug != "infrequent_access" || buckets[0].SizeBytes != 42 || !buckets[0].CDNConnected {
		t.Fatalf("decoded %+v", buckets)
	}
}

func TestObjectStorageCreateBucket(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"detail":"Bucket is being created","uuid":"b1","name":"photos","status":"pending","region":"eu","endpoint":"https://eu.cubestorage.io"}`, &rec)
	pid := 12
	b, err := c.ObjectStorage.CreateBucket(context.Background(), &CreateObjectStorageBucketRequest{Name: "photos", Tier: "infrequent_access", ProjectID: &pid})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/object-storage/buckets" {
		t.Fatalf("got %+v", rec)
	}
	if rec.Body["name"] != "photos" || rec.Body["tier"] != "infrequent_access" || rec.Body["project_id"] != float64(12) {
		t.Fatalf("body %v", rec.Body)
	}
	if _, ok := rec.Body["versioning"]; ok {
		t.Fatalf("versioning sent when not asked: %v", rec.Body)
	}
	if b.UUID != "b1" || b.Status != "pending" || b.Endpoint != "https://eu.cubestorage.io" {
		t.Fatalf("decoded %+v", b)
	}
}

func TestObjectStorageGetBucketDetail(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"b1","name":"photos","status":"active","connection":{"path_style_url":"https://eu.cubestorage.io/photos"},"usage":null,"cdn":{"status":"connected","domain":"photos.cubecdn.io"}}`, &rec)
	b, err := c.ObjectStorage.GetBucket(context.Background(), "b1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/object-storage/buckets/b1" {
		t.Fatalf("got %+v", rec)
	}
	if b.Name != "photos" || b.Connection == nil || b.Connection.PathStyleURL != "https://eu.cubestorage.io/photos" || b.Usage != nil || b.CDN == nil || b.CDN.Domain != "photos.cubecdn.io" {
		t.Fatalf("decoded %+v", b)
	}
}

func TestObjectStorageUpdateAndDeleteBucket(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"ok"}`, &rec)
	v := "enabled"
	if err := c.ObjectStorage.UpdateBucket(context.Background(), "b1", &UpdateObjectStorageBucketRequest{Versioning: &v}); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPatch || rec.Path != "/object-storage/buckets/b1" || rec.Body["versioning"] != "enabled" {
		t.Fatalf("got %+v", rec)
	}
	if _, ok := rec.Body["protected"]; ok {
		t.Fatalf("protected sent when not set: %v", rec.Body)
	}
	if err := c.ObjectStorage.DeleteBucket(context.Background(), "b1", true); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/object-storage/buckets/b1" || rec.Query != "force=true" {
		t.Fatalf("got %+v", rec)
	}
	if err := c.ObjectStorage.DeleteBucket(context.Background(), "b1", false); err != nil {
		t.Fatal(err)
	}
	if rec.Query != "" {
		t.Fatalf("force sent: %+v", rec)
	}
}

func TestObjectStorageKeys(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"k1","access_key_id":"CPABC","secret_access_key":"s3cr3t","permission":"read_only","bucket_scope":[{"uuid":"b1","name":"photos"}],"status":"pending"}`, &rec)
	k, err := c.ObjectStorage.CreateKey(context.Background(), &CreateObjectStorageKeyRequest{Name: "web", Tier: "ia", Permission: "read_only", BucketUUIDs: []string{"b1"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/object-storage/keys" || rec.Body["permission"] != "read_only" {
		t.Fatalf("got %+v", rec)
	}
	if ids, _ := rec.Body["bucket_uuids"].([]interface{}); len(ids) != 1 || ids[0] != "b1" {
		t.Fatalf("bucket_uuids %v", rec.Body)
	}
	if k.SecretAccessKey != "s3cr3t" || len(k.BucketScope) != 1 || k.BucketScope[0].Name != "photos" {
		t.Fatalf("decoded %+v", k)
	}
	if err := c.ObjectStorage.DeleteKey(context.Background(), "k1"); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/object-storage/keys/k1" {
		t.Fatalf("got %+v", rec)
	}
}

func TestObjectStorageUsageQuery(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"period":"2026-09","metrics_available":false,"total_cost":1.5,"tiers":[{"storage_gib_month":null,"cost":1.5,"free_tier":{"egress_gb":{"included":5,"used":null}}}],"buckets":[],"available_months":["2026-09"]}`, &rec)
	u, err := c.ObjectStorage.GetUsage(context.Background(), &ObjectStorageUsageOptions{Period: "2026-09"})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/object-storage/usage" || rec.Query != "period=2026-09" {
		t.Fatalf("got %+v", rec)
	}
	if u.MetricsAvailable || u.TotalCost != 1.5 || u.Tiers[0].StorageGiBMonth != nil || u.Tiers[0].FreeTier["egress_gb"].Included != 5 || len(u.AvailableMonths) != 1 {
		t.Fatalf("decoded %+v", u)
	}
}

func TestObjectStorageBucketTags(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"uuid":"b1","name":"photos","tags":{"env":"prod","team":""}}]`, &rec)
	buckets, err := c.ObjectStorage.ListBuckets(context.Background(), &ObjectStorageListOptions{Tags: []string{"env=prod", "team"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Query != "tag=env%3Dprod&tag=team" {
		t.Fatalf("query %q", rec.Query)
	}
	if len(buckets) != 1 || buckets[0].Tags["env"] != "prod" || len(buckets[0].Tags) != 2 {
		t.Fatalf("decoded %+v", buckets)
	}

	// Keys share the options struct but the tag filter is only for buckets.
	if _, err := c.ObjectStorage.ListKeys(context.Background(), &ObjectStorageListOptions{Tier: "ia", Tags: []string{"env"}}); err != nil || rec.Query != "tier=ia" {
		t.Fatalf("keys query %q (%v)", rec.Query, err)
	}

	c = newTestClient(t, 201, `{"uuid":"b1","status":"pending","tags":{"env":"prod"}}`, &rec)
	created, err := c.ObjectStorage.CreateBucket(context.Background(), &CreateObjectStorageBucketRequest{Name: "photos", Tier: "ia", Tags: map[string]string{"env": "prod"}})
	if err != nil || created.Tags["env"] != "prod" {
		t.Fatalf("created %+v (%v)", created, err)
	}
	if tags, _ := rec.Body["tags"].(map[string]interface{}); tags["env"] != "prod" {
		t.Fatalf("create body %v", rec.Body)
	}
	if _, err := c.ObjectStorage.CreateBucket(context.Background(), &CreateObjectStorageBucketRequest{Name: "photos", Tier: "ia"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := rec.Body["tags"]; ok {
		t.Fatalf("tags sent when not set: %v", rec.Body)
	}

	// nil leaves tags untouched, an empty map clears them, a map replaces them.
	v := "enabled"
	if err := c.ObjectStorage.UpdateBucket(context.Background(), "b1", &UpdateObjectStorageBucketRequest{Versioning: &v}); err != nil {
		t.Fatal(err)
	}
	if _, ok := rec.Body["tags"]; ok {
		t.Fatalf("tags sent when nil: %v", rec.Body)
	}
	empty := map[string]string{}
	if err := c.ObjectStorage.UpdateBucket(context.Background(), "b1", &UpdateObjectStorageBucketRequest{Tags: &empty}); err != nil {
		t.Fatal(err)
	}
	if tags, ok := rec.Body["tags"].(map[string]interface{}); !ok || len(tags) != 0 {
		t.Fatalf("clear body %v", rec.Body)
	}
	set := map[string]string{"env": "dev"}
	if err := c.ObjectStorage.UpdateBucket(context.Background(), "b1", &UpdateObjectStorageBucketRequest{Tags: &set}); err != nil {
		t.Fatal(err)
	}
	if tags, _ := rec.Body["tags"].(map[string]interface{}); len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("replace body %v", rec.Body)
	}
}

func TestObjectStorageUsageTagFilter(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"period":"2026-09","buckets":[{"uuid":"b1","cost":1,"tags":{"env":"prod"}}]}`, &rec)
	u, err := c.ObjectStorage.GetUsage(context.Background(), &ObjectStorageUsageOptions{Tags: []string{"env=prod"}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Query != "tag=env%3Dprod" {
		t.Fatalf("query %q", rec.Query)
	}
	if u.Buckets[0].Tags["env"] != "prod" {
		t.Fatalf("decoded %+v", u)
	}
}

func TestCDNBucketOriginSendsOnlyAllowedFields(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"o1","name":"photos","object_storage_bucket_uuid":"b1"}`, &rec)
	o, err := c.CDN.CreateOrigin(context.Background(), "z1", &CreateCDNOriginRequest{
		Name:                    "photos",
		ObjectStorageBucketUUID: "b1",
		Weight:                  100,
		HealthCheckEnabled:      true,
		VerifySSL:               true,
		Enabled:                 true,
	})
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"name": true, "object_storage_bucket_uuid": true, "weight": true, "priority": true, "is_backup": true}
	for k := range rec.Body {
		if !allowed[k] {
			t.Fatalf("%s sent with a bucket origin: %v", k, rec.Body)
		}
	}
	if rec.Body["object_storage_bucket_uuid"] != "b1" || rec.Body["weight"] != float64(100) {
		t.Fatalf("body %v", rec.Body)
	}
	if _, ok := rec.Body["priority"]; ok {
		t.Fatalf("zero priority sent: %v", rec.Body)
	}
	if o.ObjectStorageBucketUUID == nil || *o.ObjectStorageBucketUUID != "b1" {
		t.Fatalf("decoded %+v", o)
	}
}

func TestCDNRegularOriginUnchanged(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"o1"}`, &rec)
	if _, err := c.CDN.CreateOrigin(context.Background(), "z1", &CreateCDNOriginRequest{Name: "web", Address: "1.2.3.4", Weight: 100, Priority: 1, VerifySSL: true, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"address", "verify_ssl", "enabled", "health_check_enabled", "is_backup"} {
		if _, ok := rec.Body[k]; !ok {
			t.Fatalf("%s missing from a regular origin: %v", k, rec.Body)
		}
	}
	if _, ok := rec.Body["object_storage_bucket_uuid"]; ok {
		t.Fatalf("bucket uuid sent on a regular origin: %v", rec.Body)
	}
}

func TestObjectStorageBucketMetricsViaGraphQL(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"objectStorageBucket":{"uuid":"b1","name":"photos","storageMeasuredAt":1,"storage":{"start":1,"end":2,"step":3600,"series":[{"name":"size_bytes","unit":"BYTES","points":[{"ts":1,"value":42}]}]},"traffic":{"start":1,"end":2,"step":300,"series":[]},"responses":{"start":1,"end":2,"step":300,"series":[]}}}}`, &rec)
	raw, err := c.ObjectStorage.GetBucketMetrics(context.Background(), "b1", "D7")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/graphql" {
		t.Fatalf("got %+v", rec)
	}
	vars, _ := rec.Body["variables"].(map[string]interface{})
	if vars["uuid"] != "b1" || vars["range"] != "D7" {
		t.Fatalf("body %v", rec.Body)
	}
	var b struct {
		Name    string `json:"name"`
		Storage struct {
			Step int `json:"step"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(raw, &b); err != nil || b.Name != "photos" || b.Storage.Step != 3600 {
		t.Fatalf("raw %s (%v)", raw, err)
	}
}

func TestObjectStorageBucketMetricsNotFound(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"objectStorageBucket":null},"errors":[{"message":"Resource not found.","extensions":{"code":"NOT_FOUND"}}]}`, &rec)
	_, err := c.ObjectStorage.GetBucketMetrics(context.Background(), "nope", "")
	if !IsNotFound(err) {
		t.Fatalf("err %v", err)
	}
	vars, _ := rec.Body["variables"].(map[string]interface{})
	if vars["range"] != "H24" {
		t.Fatalf("default range %v", vars["range"])
	}
}

func TestObjectStorageObjectLock(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"b1","status":"pending","object_lock":{"enabled":true,"default_retention":{"mode":"governance","days":30,"years":null}}}`, &rec)
	days := 30
	b, err := c.ObjectStorage.CreateBucket(context.Background(), &CreateObjectStorageBucketRequest{
		Name: "vault", Tier: "infrequent_access", ObjectLock: true,
		ObjectLockDefault:     &ObjectStorageLockRetention{Mode: "governance", Days: &days},
		AcceptObjectLockTerms: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rec.Body["versioning"]; ok {
		t.Fatalf("versioning false sent with Object Lock: %v", rec.Body)
	}
	def, _ := rec.Body["object_lock_default"].(map[string]interface{})
	if rec.Body["object_lock"] != true || rec.Body["accept_object_lock_terms"] != true || def["mode"] != "governance" || def["days"] != float64(30) {
		t.Fatalf("body %v", rec.Body)
	}
	if _, ok := def["years"]; ok {
		t.Fatalf("years sent: %v", def)
	}
	if !b.ObjectLock.Enabled || b.ObjectLock.DefaultRetention == nil || *b.ObjectLock.DefaultRetention.Days != 30 || b.ObjectLock.DefaultRetention.Years != nil {
		t.Fatalf("decoded %+v", b.ObjectLock)
	}

	// Without lock nothing about it is sent
	if _, err := c.ObjectStorage.CreateBucket(context.Background(), &CreateObjectStorageBucketRequest{Name: "plain", Tier: "ia"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"object_lock", "object_lock_default", "accept_object_lock_terms"} {
		if _, ok := rec.Body[k]; ok {
			t.Fatalf("%s sent without lock: %v", k, rec.Body)
		}
	}

	years := 1
	if err := c.ObjectStorage.SetObjectStorageBucketObjectLock(context.Background(), "b1", &SetObjectStorageBucketObjectLockRequest{
		DefaultRetention: &ObjectStorageLockRetention{Mode: "compliance", Years: &years}, AcceptObjectLockTerms: true,
	}); err != nil {
		t.Fatal(err)
	}
	def, _ = rec.Body["default_retention"].(map[string]interface{})
	if rec.Method != http.MethodPut || rec.Path != "/object-storage/buckets/b1/object-lock" || def["mode"] != "compliance" || def["years"] != float64(1) || rec.Body["accept_object_lock_terms"] != true {
		t.Fatalf("got %+v", rec)
	}

	// Removing the rule sends an explicit null
	if err := c.ObjectStorage.SetObjectStorageBucketObjectLock(context.Background(), "b1", &SetObjectStorageBucketObjectLockRequest{}); err != nil {
		t.Fatal(err)
	}
	if v, ok := rec.Body["default_retention"]; !ok || v != nil {
		t.Fatalf("default_retention not null: %v", rec.Body)
	}

	if err := c.ObjectStorage.DeleteBucketWithOptions(context.Background(), "b1", &DeleteObjectStorageBucketOptions{Force: true, BypassGovernance: true}); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Query != "bypass_governance=true&force=true" {
		t.Fatalf("got %+v", rec)
	}
}

func TestObjectStorageBucketLockFieldsAndBypassKey(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"uuid":"b1","object_lock":{"enabled":false,"default_retention":null},"locked_content_kept":true}]`, &rec)
	buckets, err := c.ObjectStorage.ListBuckets(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if buckets[0].ObjectLock.Enabled || buckets[0].ObjectLock.DefaultRetention != nil || !buckets[0].LockedContentKept {
		t.Fatalf("decoded %+v", buckets[0])
	}

	c = newTestClient(t, 201, `{"uuid":"k1","permission":"read_write","bypass_governance":true}`, &rec)
	k, err := c.ObjectStorage.CreateKey(context.Background(), &CreateObjectStorageKeyRequest{Name: "veeam", Tier: "ia", Permission: "read_write", BypassGovernance: true})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Body["bypass_governance"] != true || !k.BypassGovernance {
		t.Fatalf("body %v decoded %+v", rec.Body, k)
	}
	if _, err := c.ObjectStorage.CreateKey(context.Background(), &CreateObjectStorageKeyRequest{Name: "web", Tier: "ia", Permission: "read_only"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := rec.Body["bypass_governance"]; ok {
		t.Fatalf("bypass_governance sent when not asked: %v", rec.Body)
	}
}
