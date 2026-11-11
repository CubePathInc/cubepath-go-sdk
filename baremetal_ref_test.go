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
	good := []string{testUUID, strings.ToUpper(testUUID)}
	bad := []string{"", "0", "5", "007", "9223372036854775807", " 5", "-5", "abc", "../x", "5/power",
		" " + testUUID, testUUID + " ", "3f2a8c10-5b6d-4e7f-8a9b-0c1d2e3f4a5", "3f2a8c105b6d4e7f8a9b0c1d2e3f4a5b",
		"3f2a8c10-5b6d-4e7f-8a9b-0c1d2e3f4a5g", "{" + testUUID + "}"}
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

// Every method sends the uuid in lower case and refuses anything else, a number included,
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
		_, _ = w.Write([]byte(`[{"project":{"id":1},"baremetals":[{"id":"` + testUUID + `","uuid":"` + testUUID + `","status":"active"}]}]`))
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
		for _, bad := range []string{"", "5", "007", "5/power", "abc", "-1"} {
			before := atomic.LoadInt32(&hits)
			if err := call(bad); err == nil || !strings.Contains(err.Error(), "baremetal reference must be a uuid") {
				t.Errorf("%s(%q): want a validation error, got %v", name, bad, err)
			}
			if atomic.LoadInt32(&hits) != before {
				t.Errorf("%s(%q): a request was sent", name, bad)
			}
		}
		if name == "Get" || name == "ReinstallStatus" || name == "BMCSensors" {
			continue
		}
		path = ""
		_ = call(strings.ToUpper(testUUID))
		if !strings.Contains(path, testUUID) {
			t.Errorf("%s: path %q does not carry the lower case uuid", name, path)
		}
	}
}

func TestBaremetalGetByUUID(t *testing.T) {
	other := "aaaaaaaa-0000-4000-8000-000000000000"
	body := `[{"project":{"id":1},"baremetals":[{"id":"` + other + `","uuid":"` + other + `","hostname":"a"},{"id":"` + testUUID + `","uuid":"` + testUUID + `","hostname":"b"}]}]`
	var rec recorded
	c := newTestClient(t, 200, body, &rec)
	bm, err := c.Baremetal.Get(context.Background(), strings.ToUpper(testUUID))
	if err != nil || bm.Hostname != "b" || bm.ID != testUUID || bm.UUID != testUUID {
		t.Fatalf("by uuid: %+v %v", bm, err)
	}
	if _, err := c.Baremetal.Get(context.Background(), "5"); err == nil || IsNotFound(err) {
		t.Fatalf("a number must be refused locally, not looked up: %v", err)
	}
}

func TestBaremetalGetNotFoundIsTyped(t *testing.T) {
	var rec recorded
	other := "aaaaaaaa-0000-4000-8000-000000000000"
	c := newTestClient(t, 200, `[{"project":{"id":1},"baremetals":[{"id":"`+other+`","uuid":"`+other+`"}]}]`, &rec)
	_, err := c.Baremetal.Get(context.Background(), testUUID)
	if !IsNotFound(err) {
		t.Fatalf("want a 404 APIError, got %v", err)
	}
	if _, err := c.Baremetal.ReinstallStatus(context.Background(), testUUID); !IsNotFound(err) {
		t.Fatalf("ReinstallStatus: %v", err)
	}
}

func TestBaremetalDecodeKeepsProjectsWorking(t *testing.T) {
	var rec recorded
	body := `[{"project":{"id":1,"name":"p"},"vps":[{"id":12,"hostname":"v"}],"networks":[{"id":3}],
	"baremetals":[{"id":"` + testUUID + `","uuid":"` + testUUID + `","hostname":"b","floating_ips":[{"id":1,"address":"1.1.1.1"}]}]}]`
	c := newTestClient(t, 200, body, &rec)
	ps, err := c.Projects.List(context.Background())
	if err != nil || len(ps) != 1 {
		t.Fatalf("%v %v", ps, err)
	}
	if ps[0].VPS[0].ID != 12 || ps[0].Networks[0].ID != 3 {
		t.Fatalf("vps/networks changed: %+v", ps[0])
	}
	b := ps[0].Baremetals[0]
	if b.ID != testUUID || b.UUID != testUUID || b.Hostname != "b" || len(b.FloatingIPs) != 1 {
		t.Fatalf("baremetal %+v", b)
	}
	vps, err := c.VPS.Get(context.Background(), 12)
	if err != nil || vps.ID != 12 {
		t.Fatalf("VPS.Get: %+v %v", vps, err)
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
	if err := c.FloatingIPs.Assign(ctx, "baremetal", "12", "1.2.3.4"); err == nil || rec.Path != "" {
		t.Fatalf("a number for a baremetal must be refused before the request: %v %+v", err, rec)
	}
	if err := c.FloatingIPs.Assign(ctx, "vps", testUUID, "1.2.3.4"); err == nil || rec.Path != "" {
		t.Fatalf("a uuid for a VPS must be refused before the request: %v %+v", err, rec)
	}
}

// The fields that name a baremetal server in other resources take the uuid (lower-cased),
// refuse a number before the request, and leave the other target types alone.
func TestBaremetalEchoFields(t *testing.T) {
	ctx := context.Background()
	var rec recorded
	c := newTestClient(t, 200, `{}`, &rec)
	upper := strings.ToUpper(testUUID)

	routeReq := &CreateNetworkRouteRequest{Destination: "10.1.0.0/24", NextHopType: "baremetal", NextHopTarget: upper}
	if _, err := c.Networks.CreateRoute(ctx, 3, routeReq); err != nil || rec.Body["next_hop_target"] != testUUID {
		t.Fatalf("route: %v %+v", err, rec.Body)
	}
	if routeReq.NextHopTarget != upper {
		t.Fatalf("the caller's request was modified: %+v", routeReq)
	}
	if _, err := c.Networks.CreateBGPPeer(ctx, 3, &CreateBGPPeerRequest{PeerType: BGPPeerTypeBaremetal, PeerTarget: upper, RemoteASN: 65001}); err != nil || rec.Body["peer_target"] != testUUID {
		t.Fatalf("bgp: %v %+v", err, rec.Body)
	}
	if _, err := c.LoadBalancer.AddTarget(ctx, "lb", "l", &AddTargetRequest{TargetType: "baremetal", TargetUUID: upper}); err != nil || rec.Body["target_uuid"] != testUUID {
		t.Fatalf("lb target: %v %+v", err, rec.Body)
	}
	if _, err := c.CloudAlerts.Create(ctx, &CreateCloudAlertRequest{TargetType: CloudAlertTargetBaremetal, TargetID: upper}); err != nil || rec.Body["target_id"] != testUUID {
		t.Fatalf("alert: %v %+v", err, rec.Body)
	}

	rec = recorded{}
	for name, call := range map[string]func() error{
		"route": func() error {
			_, e := c.Networks.CreateRoute(ctx, 3, &CreateNetworkRouteRequest{NextHopType: "baremetal", NextHopTarget: "5"})
			return e
		},
		"bgp": func() error {
			_, e := c.Networks.CreateBGPPeer(ctx, 3, &CreateBGPPeerRequest{PeerType: BGPPeerTypeBaremetal, PeerTarget: "5"})
			return e
		},
		"lb": func() error {
			_, e := c.LoadBalancer.AddTarget(ctx, "lb", "l", &AddTargetRequest{TargetType: "baremetal", TargetUUID: "5"})
			return e
		},
		"lb batch": func() error {
			_, e := c.LoadBalancer.AddTargets(ctx, "lb", "l", []AddTargetRequest{{TargetType: "vps", TargetUUID: "7"}, {TargetType: "baremetal", TargetUUID: "5"}})
			return e
		},
		"alert": func() error {
			_, e := c.CloudAlerts.Create(ctx, &CreateCloudAlertRequest{TargetType: CloudAlertTargetBaremetal, TargetID: "5"})
			return e
		},
		"alert update": func() error {
			typ, id := CloudAlertTargetBaremetal, "5"
			_, e := c.CloudAlerts.Update(ctx, "a", &UpdateCloudAlertRequest{TargetType: &typ, TargetID: &id})
			return e
		},
	} {
		if err := call(); err == nil || !strings.Contains(err.Error(), "baremetal reference must be a uuid") {
			t.Errorf("%s: want a validation error, got %v", name, err)
		}
		if rec.Path != "" {
			t.Errorf("%s: a request was sent", name)
		}
	}

	if _, err := c.Networks.CreateBGPPeer(ctx, 3, &CreateBGPPeerRequest{PeerType: BGPPeerTypeVPS, PeerTarget: "12", RemoteASN: 65001}); err != nil || rec.Body["peer_target"] != "12" {
		t.Fatalf("vps bgp target changed: %v %+v", err, rec.Body)
	}
}
