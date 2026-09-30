package cubepath

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFirewallAssignToVPSUsesFirewallRoute(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"Firewall groups updated","vps_id":7,"firewall_groups":[3,1],"sync_task_created":true}`, &rec)
	res, err := c.Firewall.AssignToVPS(context.Background(), 7, &VPSFirewallGroupsRequest{FirewallGroupIDs: []int{3, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPut || rec.Path != "/firewall/vps/7/groups" {
		t.Fatalf("got %+v", rec)
	}
	ids, _ := rec.Body["firewall_group_ids"].([]interface{})
	if len(ids) != 2 || ids[0].(float64) != 3 {
		t.Fatalf("body %v", rec.Body)
	}
	if res.Detail != "Firewall groups updated" || !res.SyncTaskCreated || len(res.FirewallGroups) != 2 {
		t.Fatalf("decoded %+v", res)
	}
}

func TestNATGatewayMetricsViaGraphQL(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"natGateway":{"metrics":{"start":1,"end":2,"step":30,"series":[{"name":"bytes_in","unit":"BYTES_PER_SECOND","points":[{"ts":1,"value":5}]}]}}}}`, &rec)
	raw, err := c.NATGateway.GetMetricsRange(context.Background(), "u1", "H24")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/graphql" {
		t.Fatalf("got %+v", rec)
	}
	vars, _ := rec.Body["variables"].(map[string]interface{})
	if vars["uuid"] != "u1" || vars["range"] != "H24" || !strings.Contains(rec.Body["query"].(string), "natGateway") {
		t.Fatalf("body %v", rec.Body)
	}
	var m struct {
		Step   int `json:"step"`
		Series []struct {
			Name string `json:"name"`
		} `json:"series"`
	}
	if err := json.Unmarshal(raw, &m); err != nil || m.Step != 30 || m.Series[0].Name != "bytes_in" {
		t.Fatalf("raw %s (%v)", raw, err)
	}
}

func TestNATGatewayBandwidthUsageViaGraphQL(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"natGateway":{"bandwidthUsage":{"inBytes":10,"outBytes":5,"totalBytes":15,"periodStart":1,"periodEnd":2}}}}`, &rec)
	raw, err := c.NATGateway.GetBandwidthUsage(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/graphql" || !strings.Contains(string(raw), `"totalBytes":15`) {
		t.Fatalf("got %+v raw %s", rec, raw)
	}
}

func TestGraphQLNotFoundIsAPIError404(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"natGateway":null},"errors":[{"message":"Resource not found.","extensions":{"code":"NOT_FOUND"}}]}`, &rec)
	_, err := c.NATGateway.GetMetrics(context.Background(), "nope")
	if !IsNotFound(err) {
		t.Fatalf("want 404, got %v", err)
	}
}

func TestBaremetalBMCSensorsViaGraphQL(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"baremetal":{"sensors":{"ipmiAvailable":true,"powerOn":true,"lastSeen":100,"temperatures":[{"name":"CPU","value":41.5,"unit":"CELSIUS"}],"fans":[{"name":"FAN1","value":3000,"unit":"RPM"}]}}}}`, &rec)
	res, err := c.Baremetal.BMCSensors(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	vars, _ := rec.Body["variables"].(map[string]interface{})
	if rec.Path != "/graphql" || vars["id"] != "9" {
		t.Fatalf("got %+v", rec)
	}
	if !res.IPMIAvailable || !res.PowerOn || res.LastSeen != 100 || res.Sensors.Temperatures[0].Unit != "CELSIUS" || res.Sensors.Fans[0].Value != 3000 {
		t.Fatalf("decoded %+v", res)
	}
}

func TestBaremetalBMCSensorsNeverPolled(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"data":{"baremetal":{"sensors":{"ipmiAvailable":null,"powerOn":null,"lastSeen":null,"temperatures":[],"fans":[]}}}}`, &rec)
	res, err := c.Baremetal.BMCSensors(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if res.IPMIAvailable || res.LastSeen != 0 || res.Sensors.Temperatures == nil {
		t.Fatalf("decoded %+v", res)
	}
}

func TestBaremetalReinstallStatusFromServerStatus(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"id":1,"name":"p","baremetals":[{"id":9,"hostname":"h","status":"deploying"}]}]`, &rec)
	st, err := c.Baremetal.ReinstallStatus(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/projects/" || !st.IsReinstalling || st.Status != "deploying" {
		t.Fatalf("got %+v %+v", rec, st)
	}
}

func TestBaremetalCancelReinstall(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"Reinstallation cancelled"}`, &rec)
	if err := c.Baremetal.CancelReinstall(context.Background(), 9); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodDelete || rec.Path != "/baremetal/9/reinstall" {
		t.Fatalf("got %+v", rec)
	}
}

func TestFirewallCreateSendsProjectID(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"id":5,"name":"web","project_id":12,"enabled":true,"rules":[]}`, &rec)
	if _, err := c.Firewall.Create(context.Background(), &CreateFirewallGroupRequest{ProjectID: 12, Name: "web", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/firewall/groups" || rec.Query != "project_id=12" {
		t.Fatalf("got %+v", rec)
	}
	if _, ok := rec.Body["project_id"]; ok {
		t.Fatalf("project_id must not be in the body: %v", rec.Body)
	}
	if _, err := c.Firewall.Create(context.Background(), &CreateFirewallGroupRequest{Name: "web"}); err == nil {
		t.Fatal("want an error without ProjectID")
	}
}

func TestFirewallUpdateUsesPut(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"id":5,"name":"new","project_id":12,"enabled":false,"rules":[]}`, &rec)
	name := "new"
	g, err := c.Firewall.Update(context.Background(), 5, &UpdateFirewallGroupRequest{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPut || rec.Path != "/firewall/groups/5" || g.Name != "new" {
		t.Fatalf("got %+v %+v", rec, g)
	}
}

func TestFirewallGetFromList(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"id":4,"name":"a"},{"id":5,"name":"b"}]`, &rec)
	g, err := c.Firewall.Get(context.Background(), 5)
	if err != nil || g.Name != "b" || rec.Method != http.MethodGet || rec.Path != "/firewall/groups" {
		t.Fatalf("got %+v %+v %v", rec, g, err)
	}
	if _, err := c.Firewall.Get(context.Background(), 99); !IsNotFound(err) {
		t.Fatalf("want 404, got %v", err)
	}
}

func TestLoadBalancerGetFromList(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"uuid":"a","name":"x"},{"uuid":"b","name":"y"}]`, &rec)
	lb, err := c.LoadBalancer.Get(context.Background(), "b")
	if err != nil || lb.Name != "y" || rec.Path != "/loadbalancer/" {
		t.Fatalf("got %+v %+v %v", rec, lb, err)
	}
	if _, err := c.LoadBalancer.Get(context.Background(), "zz"); !IsNotFound(err) {
		t.Fatalf("want 404, got %v", err)
	}
}
