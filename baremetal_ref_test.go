package cubepath

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const testUUID = "3f2a8c10-5b6d-4e7f-8a9b-0c1d2e3f4a5b"

func TestBaremetalRefValidation(t *testing.T) {
	good := []string{"5", "007", testUUID, strings.ToUpper(testUUID)}
	bad := []string{"", " 5", "5 ", "-5", "5.0", "1e3", "abc", "../x", "5/power", "5?x=1",
		"3f2a8c10-5b6d-4e7f-8a9b-0c1d2e3f4a5", "3f2a8c105b6d4e7f8a9b0c1d2e3f4a5b",
		"3f2a8c10-5b6d-4e7f-8a9b-0c1d2e3f4a5g"}
	for _, g := range good {
		if !IsBaremetalRef(g) {
			t.Errorf("%q should be valid", g)
		}
	}
	for _, b := range bad {
		if IsBaremetalRef(b) {
			t.Errorf("%q should be invalid", b)
		}
	}
}

// Every method sends the reference as given (uuid in lower case) and refuses a bad one
// before any request.
func TestBaremetalRefPassThroughAndRejection(t *testing.T) {
	ctx := context.Background()
	var hits int32
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/graphql" {
			_, _ = w.Write([]byte(`{"data":{"baremetal":{"sensors":{"temperatures":[],"fans":[]}}}}`))
			return
		}
		_, _ = w.Write([]byte(`[{"project":{"id":1},"baremetals":[{"id":5,"uuid":"` + testUUID + `","status":"active"}]}]`))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient("token", WithBaseURL(srv.URL), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	s := "x"
	calls := map[string]func(ref string) error{
		"Update":            func(r string) error { return c.Baremetal.Update(ctx, r, &UpdateBaremetalRequest{Label: &s}) },
		"Power":             func(r string) error { return c.Baremetal.Power(ctx, r, "restart_metal") },
		"Rescue":            func(r string) error { _, e := c.Baremetal.Rescue(ctx, r); return e },
		"ResetBMC":          func(r string) error { return c.Baremetal.ResetBMC(ctx, r) },
		"IPMISession":       func(r string) error { _, e := c.Baremetal.IPMISession(ctx, r); return e },
		"Reinstall":         func(r string) error { return c.Baremetal.Reinstall(ctx, r, &ReinstallBaremetalRequest{OSName: "d"}) },
		"CancelReinstall":   func(r string) error { return c.Baremetal.CancelReinstall(ctx, r) },
		"MonitoringEnable":  func(r string) error { return c.Baremetal.MonitoringEnable(ctx, r) },
		"MonitoringDisable": func(r string) error { return c.Baremetal.MonitoringDisable(ctx, r) },
		"ListOS":            func(r string) error { _, e := c.Baremetal.ListOS(ctx, r); return e },
		"GetKVM":            func(r string) error { _, e := c.Baremetal.GetKVM(ctx, r); return e },
		"SetProtection":     func(r string) error { return c.Baremetal.SetProtection(ctx, r, true) },
		"MoveToProject":     func(r string) error { return c.Baremetal.MoveToProject(ctx, r, 2) },
		"AddSSHKeys":        func(r string) error { return c.Baremetal.AddSSHKeys(ctx, r, []int{1}) },
		"RemoveSSHKey":      func(r string) error { return c.Baremetal.RemoveSSHKey(ctx, r, 1) },
		"AttachNetwork":     func(r string) error { return c.Baremetal.AttachNetwork(ctx, r, 3) },
		"DetachNetwork":     func(r string) error { return c.Baremetal.DetachNetwork(ctx, r) },
		"Get":               func(r string) error { _, e := c.Baremetal.Get(ctx, r); return e },
		"ReinstallStatus":   func(r string) error { _, e := c.Baremetal.ReinstallStatus(ctx, r); return e },
		"BMCSensors":        func(r string) error { _, e := c.Baremetal.BMCSensors(ctx, r); return e },
		"FloatingAssign":    func(r string) error { return c.FloatingIPs.Assign(ctx, "baremetal", r, "10.0.0.1") },
	}
	if len(calls) != 21 {
		t.Fatalf("expected 20 baremetal methods plus the floating IP assign, got %d", len(calls))
	}
	for name, call := range calls {
		for _, bad := range []string{"", "5/power", "abc", "-1"} {
			before := atomic.LoadInt32(&hits)
			if err := call(bad); err == nil || !strings.Contains(err.Error(), "invalid baremetal reference") {
				t.Errorf("%s(%q): want a validation error, got %v", name, bad, err)
			}
			if atomic.LoadInt32(&hits) != before {
				t.Errorf("%s(%q): a request was sent", name, bad)
			}
		}
		if name == "Get" || name == "ReinstallStatus" || name == "BMCSensors" {
			continue
		}
		_ = call(strings.ToUpper(testUUID))
		if !strings.Contains(path, testUUID) {
			t.Errorf("%s: path %q does not carry the lower case uuid", name, path)
		}
		_ = call("5")
		if !strings.Contains(path, "/5") {
			t.Errorf("%s: path %q does not carry the number", name, path)
		}
	}
}

func TestBaremetalGetByNumberAndUUID(t *testing.T) {
	for _, body := range []string{
		// today's API: number, no uuid
		`[{"project":{"id":1},"baremetals":[{"id":4,"hostname":"a"},{"id":5,"hostname":"b"}]}]`,
		// post-cutover API: number plus uuid
		`[{"project":{"id":1},"baremetals":[{"id":4,"uuid":"aaaaaaaa-0000-0000-0000-000000000000","hostname":"a"},{"id":5,"uuid":"` + testUUID + `","hostname":"b"}]}]`,
		// numeric string id
		`[{"project":{"id":1},"baremetals":[{"id":"4","hostname":"a"},{"id":"5","hostname":"b"}]}]`,
	} {
		var rec recorded
		c := newTestClient(t, 200, body, &rec)
		bm, err := c.Baremetal.Get(context.Background(), "5")
		if err != nil || bm.Hostname != "b" || bm.ID != 5 || bm.Ref() != "5" {
			t.Fatalf("by number: %+v %v (%s)", bm, err, body)
		}
		bm, err = c.Baremetal.Get(context.Background(), "005")
		if err != nil || bm.ID != 5 {
			t.Fatalf("zeros: %+v %v", bm, err)
		}
		if strings.Contains(body, testUUID) {
			bm, err = c.Baremetal.Get(context.Background(), strings.ToUpper(testUUID))
			if err != nil || bm.ID != 5 || bm.UUID != testUUID {
				t.Fatalf("by uuid: %+v %v", bm, err)
			}
		} else if _, err = c.Baremetal.Get(context.Background(), testUUID); !IsNotFound(err) {
			t.Fatalf("uuid against the old API should be not found: %v", err)
		}
	}
}

func TestBaremetalGetNotFoundIsTyped(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"project":{"id":1},"baremetals":[{"id":4}]}]`, &rec)
	_, err := c.Baremetal.Get(context.Background(), "9")
	if !IsNotFound(err) {
		t.Fatalf("want a 404 APIError, got %v", err)
	}
	if _, err := c.Baremetal.ReinstallStatus(context.Background(), "9"); !IsNotFound(err) {
		t.Fatalf("ReinstallStatus: %v", err)
	}
}

func TestBaremetalDecodeKeepsProjectsWorking(t *testing.T) {
	var rec recorded
	body := `[{"project":{"id":1,"name":"p"},"vps":[{"id":12,"hostname":"v"}],"networks":[{"id":3}],
	"baremetals":[{"id":5,"uuid":"` + testUUID + `","hostname":"b","floating_ips":[{"id":1,"address":"1.1.1.1"}]}]}]`
	c := newTestClient(t, 200, body, &rec)
	ps, err := c.Projects.List(context.Background())
	if err != nil || len(ps) != 1 {
		t.Fatalf("%v %v", ps, err)
	}
	if ps[0].VPS[0].ID != 12 || ps[0].Networks[0].ID != 3 {
		t.Fatalf("vps/networks changed: %+v", ps[0])
	}
	b := ps[0].Baremetals[0]
	if b.ID != 5 || b.UUID != testUUID || b.Hostname != "b" || len(b.FloatingIPs) != 1 {
		t.Fatalf("baremetal %+v", b)
	}
	vps, err := c.VPS.Get(context.Background(), 12)
	if err != nil || vps.ID != 12 {
		t.Fatalf("VPS.Get: %+v %v", vps, err)
	}
}

func TestBaremetalIDDecoding(t *testing.T) {
	cases := map[string]struct {
		id   int
		uuid string
		err  bool
	}{
		`{"id":5}`:                           {5, "", false},
		`{"id":"5"}`:                         {5, "", false},
		`{"uuid":"` + testUUID + `"}`:        {0, testUUID, false},
		`{"id":null}`:                        {0, "", false},
		`{"id":"` + testUUID + `"}`:          {0, testUUID, false},
		`{"id":"nope"}`:                      {0, "", true},
		`{"id":5,"uuid":"` + testUUID + `"}`: {5, testUUID, false},
	}
	for in, want := range cases {
		var b Baremetal
		err := b.UnmarshalJSON([]byte(in))
		if (err != nil) != want.err || b.ID != want.id || b.UUID != want.uuid {
			t.Errorf("%s: got %d %q %v", in, b.ID, b.UUID, err)
		}
	}
}

func TestFloatingIPAssignRefs(t *testing.T) {
	ctx := context.Background()
	var rec recorded
	c := newTestClient(t, 200, `{}`, &rec)
	if err := c.FloatingIPs.Assign(ctx, "vps", "12", "1.2.3.4"); err != nil || rec.Path != "/floating_ips/assign/vps/12" || rec.Query != "address=1.2.3.4" {
		t.Fatalf("vps: %+v %v", rec, err)
	}
	if err := c.FloatingIPs.Assign(ctx, "baremetal", strings.ToUpper(testUUID), "1.2.3.4"); err != nil || rec.Path != "/floating_ips/assign/baremetal/"+testUUID {
		t.Fatalf("baremetal uuid: %+v %v", rec, err)
	}
	rec = recorded{}
	if err := c.FloatingIPs.Assign(ctx, "vps", testUUID, "1.2.3.4"); err == nil || rec.Path != "" {
		t.Fatalf("a uuid for a VPS must be refused before the request: %v %+v", err, rec)
	}
}
