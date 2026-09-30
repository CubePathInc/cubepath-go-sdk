package cubepath

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// rawRecorded keeps the request body as bytes, for bodies that are not JSON objects.
type rawRecorded struct {
	Method      string
	Path        string
	Query       string
	ContentType string
	Body        []byte
}

func newRawTestClient(t *testing.T, status int, response string, rec *rawRecorded) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.Method, rec.Path, rec.Query = r.Method, r.URL.Path, r.URL.RawQuery
		rec.ContentType = r.Header.Get("Content-Type")
		rec.Body, _ = io.ReadAll(r.Body)
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

// TestEndpointRoutes checks the method, path, query and body each new call sends.
func TestEndpointRoutes(t *testing.T) {
	ctx := context.Background()
	two := 2
	yes := true
	cases := []struct {
		name   string
		status int
		resp   string
		call   func(c *Client) error
		method string
		path   string
		query  string
		body   map[string]interface{}
	}{
		// Managed Databases
		{"mdb plans", 200, `[]`, func(c *Client) error { _, err := c.ManagedDatabases.ListPlans(ctx, "mysql"); return err },
			"GET", "/managed-database-plans/", "engine=mysql", nil},
		{"mdb list", 200, `[]`, func(c *Client) error { _, err := c.ManagedDatabases.List(ctx); return err },
			"GET", "/managed-databases/", "", nil},
		{"mdb create", 201, `{"uuid":"m1","status":"provisioning"}`, func(c *Client) error {
			_, err := c.ManagedDatabases.Create(ctx, &CreateManagedDatabaseRequest{ProjectID: 5, Name: "db", Engine: "postgresql", Version: "17.5.0", PlanUUID: "p1", Replicas: 2})
			return err
		}, "POST", "/managed-databases/", "", map[string]interface{}{"project_id": 5.0, "name": "db", "engine": "postgresql", "version": "17.5.0", "plan_uuid": "p1", "replicas": 2.0}},
		{"mdb update", 200, `{}`, func(c *Client) error {
			l := "x"
			return c.ManagedDatabases.Update(ctx, "m1", &UpdateManagedDatabaseRequest{Label: &l})
		}, "PATCH", "/managed-databases/m1", "", map[string]interface{}{"label": "x"}},
		{"mdb delete", 200, `{}`, func(c *Client) error { return c.ManagedDatabases.Delete(ctx, "m1") },
			"DELETE", "/managed-databases/m1", "", nil},
		{"mdb protection", 200, `{}`, func(c *Client) error { return c.ManagedDatabases.SetProtection(ctx, "m1", true) },
			"POST", "/managed-databases/m1/protection", "", map[string]interface{}{"enabled": true}},
		{"mdb scale", 200, `{"replicas":2}`, func(c *Client) error {
			_, err := c.ManagedDatabases.Scale(ctx, "m1", &ScaleManagedDatabaseRequest{Replicas: &two})
			return err
		}, "POST", "/managed-databases/m1/scale", "", map[string]interface{}{"replicas": 2.0}},
		{"mdb credentials", 200, `{"host":"h","port":5432}`, func(c *Client) error { _, err := c.ManagedDatabases.GetCredentials(ctx, "m1"); return err },
			"GET", "/managed-databases/m1/credentials", "", nil},
		{"mdb rotate", 200, `{}`, func(c *Client) error { return c.ManagedDatabases.RotateCredentials(ctx, "m1") },
			"POST", "/managed-databases/m1/credentials/rotate", "", nil},
		{"mdb config", 200, `{"engine":"mysql","params":{}}`, func(c *Client) error { _, err := c.ManagedDatabases.GetConfig(ctx, "m1"); return err },
			"GET", "/managed-databases/m1/config", "", nil},
		{"mdb config update", 200, `{"requires_restart":[]}`, func(c *Client) error {
			_, err := c.ManagedDatabases.UpdateConfig(ctx, "m1", map[string]interface{}{"max_connections": 200})
			return err
		}, "PATCH", "/managed-databases/m1/config", "", map[string]interface{}{"params": map[string]interface{}{"max_connections": 200.0}}},
		{"mdb metrics", 200, `{"start":1,"end":2,"metrics":{}}`, func(c *Client) error {
			_, err := c.ManagedDatabases.GetMetrics(ctx, "m1", &ManagedDatabaseMetricsOptions{Metrics: []string{"cpu", "memory"}, TimeRange: "24h"})
			return err
		}, "GET", "/managed-databases/m1/metrics", "metrics=cpu%2Cmemory&time_range=24h", nil},
		{"mdb databases", 200, `[]`, func(c *Client) error { _, err := c.ManagedDatabases.ListDatabases(ctx, "m1"); return err },
			"GET", "/managed-databases/m1/databases", "", nil},
		{"mdb database create", 201, `{"uuid":"d1"}`, func(c *Client) error { _, err := c.ManagedDatabases.CreateDatabase(ctx, "m1", "app"); return err },
			"POST", "/managed-databases/m1/databases", "", map[string]interface{}{"name": "app"}},
		{"mdb database delete", 200, `{}`, func(c *Client) error { return c.ManagedDatabases.DeleteDatabase(ctx, "m1", "d1") },
			"DELETE", "/managed-databases/m1/databases/d1", "", nil},
		{"mdb users", 200, `[]`, func(c *Client) error { _, err := c.ManagedDatabases.ListUsers(ctx, "m1"); return err },
			"GET", "/managed-databases/m1/users", "", nil},
		{"mdb user create", 201, `{"uuid":"u1","password":"p"}`, func(c *Client) error {
			_, err := c.ManagedDatabases.CreateUser(ctx, "m1", &CreateManagedDatabaseUserRequest{Username: "app"})
			return err
		}, "POST", "/managed-databases/m1/users", "", map[string]interface{}{"username": "app"}},
		{"mdb user delete", 200, `{}`, func(c *Client) error { return c.ManagedDatabases.DeleteUser(ctx, "m1", "u1") },
			"DELETE", "/managed-databases/m1/users/u1", "", nil},

		// DDoS Mitigation
		{"ddos ips", 200, `{"single_ips":[],"subnets":[],"total":0}`, func(c *Client) error {
			_, err := c.DDoSMitigation.ListIPs(ctx, &DDoSIPListOptions{IPType: "IPv4", HasProfile: &yes})
			return err
		}, "GET", "/ddos-mitigation/ips", "has_profile=true&ip_type=IPv4", nil},
		{"ddos asns", 200, `{"asns":[]}`, func(c *Client) error { _, err := c.DDoSMitigation.ListASNs(ctx, "cloud"); return err },
			"GET", "/ddos-mitigation/asns", "search=cloud", nil},
		{"ddos profile cidr", 200, `{}`, func(c *Client) error { _, err := c.DDoSMitigation.GetProfile(ctx, "194.26.100.0/24"); return err },
			"GET", "/ddos-mitigation/profiles/194.26.100.0/24", "", nil},
		{"ddos profile delete", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.DeleteProfile(ctx, "194.26.100.5") },
			"DELETE", "/ddos-mitigation/profiles/194.26.100.5", "", nil},
		{"ddos countries set", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.SetProfileCountries(ctx, "194.26.100.5", nil) },
			"PUT", "/ddos-mitigation/profiles/194.26.100.5/countries", "", map[string]interface{}{"iso_codes": []interface{}{}}},
		{"ddos asns set", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.SetProfileASNs(ctx, "194.26.100.5", []int64{13335}) },
			"PUT", "/ddos-mitigation/profiles/194.26.100.5/asns", "", map[string]interface{}{"asns": []interface{}{13335.0}}},
		{"ddos prefix lists set", 200, `{}`, func(c *Client) error {
			return c.DDoSMitigation.SetProfilePrefixLists(ctx, "194.26.100.5", []string{"p1"})
		}, "PUT", "/ddos-mitigation/profiles/194.26.100.5/prefix-lists", "", map[string]interface{}{"uuids": []interface{}{"p1"}}},
		{"ddos fw list", 200, `{"rules":[],"total":0}`, func(c *Client) error { _, err := c.DDoSMitigation.ListFirewallRules(ctx, "2001:db8::1"); return err },
			"GET", "/ddos-mitigation/firewall-rules/2001:db8::1", "", nil},
		{"ddos fw create", 201, `{}`, func(c *Client) error {
			return c.DDoSMitigation.CreateFirewallRule(ctx, &CreateDDoSFirewallRuleRequest{Network: "194.26.100.5", Protocol: DDoSProtocolTCP, DstPort: 22, Action: DDoSActionDrop})
		}, "POST", "/ddos-mitigation/firewall-rules", "", map[string]interface{}{"network": "194.26.100.5", "protocol": 6.0, "dst_port": 22.0, "action": 0.0}},
		{"ddos fw delete", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.DeleteFirewallRule(ctx, 9) },
			"DELETE", "/ddos-mitigation/firewall-rules/9", "", nil},
		{"ddos fw bulk", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.DeleteFirewallRules(ctx, "194.26.100.5", 17, 0) },
			"DELETE", "/ddos-mitigation/firewall-rules/bulk", "dst_port=0&network=194.26.100.5&protocol=17", nil},
		{"ddos entry add", 201, `{}`, func(c *Client) error { return c.DDoSMitigation.AddPrefixListEntry(ctx, "p1", "10.0.0.0/8") },
			"POST", "/ddos-mitigation/prefix-lists/p1/entries", "", map[string]interface{}{"network": "10.0.0.0/8"}},
		{"ddos entry delete", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.DeletePrefixListEntry(ctx, "p1", "10.0.0.0/8") },
			"DELETE", "/ddos-mitigation/prefix-lists/p1/entries/10.0.0.0/8", "", nil},
		{"ddos list delete", 200, `{}`, func(c *Client) error { return c.DDoSMitigation.DeletePrefixList(ctx, "p1") },
			"DELETE", "/ddos-mitigation/prefix-lists/p1", "", nil},
		{"ddos stats", 200, `{"buckets":[]}`, func(c *Client) error {
			_, err := c.DDoSMitigation.GetTrafficStats(ctx, &DDoSTrafficStatsRequest{StartTime: time.Unix(0, 0).UTC(), EndTime: time.Unix(60, 0).UTC(), Interval: "1m"})
			return err
		}, "POST", "/ddos-mitigation/traffic-capture/stats", "", map[string]interface{}{"start_time": "1970-01-01T00:00:00Z", "end_time": "1970-01-01T00:01:00Z", "interval": "1m"}},
		{"ddos attack details", 200, `{}`, func(c *Client) error { _, err := c.DDoS.GetAttackDetails(ctx, 77); return err },
			"GET", "/ddos-attacks/attacks/77/details", "", nil},
		{"ddos attack graph", 200, `{"data":[]}`, func(c *Client) error { _, err := c.DDoS.GetAttackTrafficGraph(ctx, 77); return err },
			"GET", "/ddos-attacks/attacks/77/traffic-graph", "", nil},

		// Cloud Alerts
		{"alerts list", 200, `[]`, func(c *Client) error {
			_, err := c.CloudAlerts.List(ctx, &CloudAlertListOptions{ProjectID: 3, Status: "enabled"})
			return err
		}, "GET", "/triggers/", "project_id=3&status=enabled", nil},
		{"alerts create", 201, `{"id":"t1"}`, func(c *Client) error {
			_, err := c.CloudAlerts.Create(ctx, &CreateCloudAlertRequest{ProjectID: 3, Name: "cpu", TargetType: CloudAlertTargetVPS, TargetID: "10", MetricType: CloudAlertMetricCPU, Operator: CloudAlertOperatorGreaterThan, Threshold: 90,
				Actions: []CloudAlertActionRequest{{ActionType: CloudAlertActionNotify, NotificationChannelID: "n1"}}})
			return err
		}, "POST", "/triggers/", "", map[string]interface{}{"project_id": 3.0, "name": "cpu", "target_type": "vps", "target_id": "10", "metric_type": "cpu", "operator": "gt", "threshold": 90.0,
			"actions": []interface{}{map[string]interface{}{"action_type": "notify", "notificator_id": "n1", "order": 0.0}}}},
		{"alerts update", 200, `{"id":"t1"}`, func(c *Client) error {
			st := "disabled"
			_, err := c.CloudAlerts.Update(ctx, "t1", &UpdateCloudAlertRequest{Status: &st})
			return err
		}, "PUT", "/triggers/t1", "", map[string]interface{}{"status": "disabled"}},
		{"alerts delete", 204, ``, func(c *Client) error { return c.CloudAlerts.Delete(ctx, "t1") },
			"DELETE", "/triggers/t1", "", nil},
		{"alerts history", 200, `[]`, func(c *Client) error { _, err := c.CloudAlerts.History(ctx, "t1", 10); return err },
			"GET", "/triggers/t1/history", "limit=10", nil},
		{"channels create", 201, `{"id":"n1"}`, func(c *Client) error {
			_, err := c.CloudAlerts.CreateNotificationChannel(ctx, &CreateNotificationChannelRequest{Name: "mail", Type: NotificationChannelEmail})
			return err
		}, "POST", "/triggers/notificators/", "", map[string]interface{}{"name": "mail", "type": "email"}},
		{"channels update", 200, `{"id":"n1"}`, func(c *Client) error {
			_, err := c.CloudAlerts.UpdateNotificationChannel(ctx, "n1", &UpdateNotificationChannelRequest{Enabled: &yes})
			return err
		}, "PUT", "/triggers/notificators/n1", "", map[string]interface{}{"enabled": true}},
		{"channels delete", 204, ``, func(c *Client) error { return c.CloudAlerts.DeleteNotificationChannel(ctx, "n1") },
			"DELETE", "/triggers/notificators/n1", "", nil},

		// Transcoder
		{"transcoder list", 200, `{"jobs":[],"limit":10,"offset":20}`, func(c *Client) error {
			_, err := c.Transcoder.ListJobs(ctx, &TranscodeJobListOptions{BatchID: "b1", Limit: 10, Offset: 20})
			return err
		}, "GET", "/transcoder/jobs", "batch_id=b1&limit=10&offset=20", nil},
		{"transcoder outputs", 200, `{"outputs":[]}`, func(c *Client) error { _, err := c.Transcoder.GetJobOutputs(ctx, "j1"); return err },
			"GET", "/transcoder/jobs/j1/outputs", "", nil},
		{"transcoder cancel", 200, `{"detail":"ok","status":"canceled"}`, func(c *Client) error { _, err := c.Transcoder.CancelJob(ctx, "j1"); return err },
			"DELETE", "/transcoder/jobs/j1", "", nil},

		// VPS
		{"vps plans", 200, `{"locations":[]}`, func(c *Client) error { _, err := c.VPS.Plans(ctx); return err },
			"GET", "/vps/plans", "", nil},
		{"vps protection", 200, `{}`, func(c *Client) error { return c.VPS.SetProtection(ctx, 7, false) },
			"POST", "/vps/7/protection", "", map[string]interface{}{"enabled": false}},
		{"vps move", 200, `{}`, func(c *Client) error { return c.VPS.MoveToProject(ctx, 7, 3) },
			"POST", "/vps/7/move-project", "", map[string]interface{}{"project_id": 3.0}},
		{"vps key remove", 200, `{}`, func(c *Client) error { return c.VPS.RemoveSSHKey(ctx, 7, 4) },
			"DELETE", "/vps/7/ssh-keys/4", "", nil},
		{"vps network attach", 200, `{}`, func(c *Client) error { return c.VPS.AttachNetwork(ctx, 7, 12) },
			"POST", "/vps/7/network", "", map[string]interface{}{"network_id": 12.0}},
		{"vps network detach", 200, `{}`, func(c *Client) error { return c.VPS.DetachNetwork(ctx, 7) },
			"DELETE", "/vps/7/network", "", nil},
		{"vps vnc", 200, `{"websocket_url":"wss://x","vnc_info":{"ticket":"t"}}`, func(c *Client) error { _, err := c.VPS.VNCURL(ctx, 7); return err },
			"POST", "/vps/7/vnc-url", "", nil},
		{"ag list", 200, `{"groups":[]}`, func(c *Client) error { _, err := c.VPS.AvailabilityGroups().ListByProject(ctx, 3); return err },
			"GET", "/vps/availability-groups/project/3", "", nil},
		{"ag create", 200, `{"uuid":"g1"}`, func(c *Client) error {
			_, err := c.VPS.AvailabilityGroups().Create(ctx, &CreateAvailabilityGroupRequest{ProjectID: 3, Name: "web", LocationName: "eu-bcn-1"})
			return err
		}, "POST", "/vps/availability-groups/", "", map[string]interface{}{"project_id": 3.0, "name": "web", "location_name": "eu-bcn-1"}},
		{"ag add", 200, `{}`, func(c *Client) error { return c.VPS.AvailabilityGroups().AddVPS(ctx, "g1", 7) },
			"POST", "/vps/availability-groups/g1/vps/7", "", nil},
		{"ag remove", 200, `{}`, func(c *Client) error { return c.VPS.AvailabilityGroups().RemoveVPS(ctx, "g1", 7) },
			"DELETE", "/vps/availability-groups/g1/vps/7", "", nil},
		{"ag move", 200, `{}`, func(c *Client) error { return c.VPS.AvailabilityGroups().MoveToProject(ctx, "g1", 4) },
			"POST", "/vps/availability-groups/g1/move-project", "", map[string]interface{}{"project_id": 4.0}},
		{"ag delete", 200, `{}`, func(c *Client) error { return c.VPS.AvailabilityGroups().Delete(ctx, "g1") },
			"DELETE", "/vps/availability-groups/g1", "", nil},

		// Baremetal
		{"bm models", 200, `{"locations":[]}`, func(c *Client) error { _, err := c.Baremetal.ListModels(ctx); return err },
			"GET", "/baremetal/models", "", nil},
		{"bm os", 200, `[]`, func(c *Client) error { _, err := c.Baremetal.ListOS(ctx, 9); return err },
			"GET", "/baremetal/os/9", "", nil},
		{"bm kvm", 200, `{"url":"u"}`, func(c *Client) error { _, err := c.Baremetal.GetKVM(ctx, 9); return err },
			"GET", "/baremetal/9/kvm", "", nil},
		{"bm protection", 200, `{}`, func(c *Client) error { return c.Baremetal.SetProtection(ctx, 9, true) },
			"POST", "/baremetal/9/protection", "", map[string]interface{}{"enabled": true}},
		{"bm move", 200, `{}`, func(c *Client) error { return c.Baremetal.MoveToProject(ctx, 9, 3) },
			"POST", "/baremetal/9/move-project", "", map[string]interface{}{"project_id": 3.0}},
		{"bm key remove", 200, `{}`, func(c *Client) error { return c.Baremetal.RemoveSSHKey(ctx, 9, 4) },
			"DELETE", "/baremetal/9/ssh-keys/4", "", nil},
		{"bm network attach", 200, `{}`, func(c *Client) error { return c.Baremetal.AttachNetwork(ctx, 9, 12) },
			"POST", "/baremetal/9/network", "", map[string]interface{}{"network_id": 12.0}},
		{"bm network detach", 200, `{}`, func(c *Client) error { return c.Baremetal.DetachNetwork(ctx, 9) },
			"DELETE", "/baremetal/9/network", "", nil},

		// Networks
		{"net move", 200, `{}`, func(c *Client) error { return c.Networks.MoveToProject(ctx, 12, 3) },
			"POST", "/networks/12/move-project", "", map[string]interface{}{"project_id": 3.0}},
		{"bgp list", 200, `[]`, func(c *Client) error { _, err := c.Networks.ListBGPPeers(ctx, 12); return err },
			"GET", "/networks/12/bgp-peers", "", nil},
		{"bgp create", 201, `{"peer_id":"b1"}`, func(c *Client) error {
			_, err := c.Networks.CreateBGPPeer(ctx, 12, &CreateBGPPeerRequest{PeerType: BGPPeerTypeVPS, PeerTarget: "7", RemoteASN: 65010})
			return err
		}, "POST", "/networks/12/bgp-peers", "", map[string]interface{}{"peer_type": "vps", "peer_target": "7", "remote_asn": 65010.0}},
		{"bgp update", 200, `{}`, func(c *Client) error {
			return c.Networks.UpdateBGPPeer(ctx, 12, "b1", &UpdateBGPPeerRequest{Enabled: &yes})
		}, "PATCH", "/networks/12/bgp-peers/b1", "", map[string]interface{}{"enabled": true}},
		{"bgp delete", 200, `{}`, func(c *Client) error { return c.Networks.DeleteBGPPeer(ctx, 12, "b1") },
			"DELETE", "/networks/12/bgp-peers/b1", "", nil},

		// CDN
		{"cdn purge", 202, `{"purge_uuid":"p1","status":"pending"}`, func(c *Client) error {
			_, err := c.CDN.PurgeCache(ctx, "z1", &CDNPurgeRequest{Paths: []string{"/a.css", "/img/*"}})
			return err
		}, "POST", "/cdn/zones/z1/purge-cache", "", map[string]interface{}{"paths": []interface{}{"/a.css", "/img/*"}}},
		{"cdn purge everything", 202, `{}`, func(c *Client) error {
			_, err := c.CDN.PurgeCache(ctx, "z1", &CDNPurgeRequest{Everything: true})
			return err
		}, "POST", "/cdn/zones/z1/purge-cache", "", map[string]interface{}{"everything": true}},
		{"cdn purges", 200, `[]`, func(c *Client) error { _, err := c.CDN.ListPurges(ctx, "z1"); return err },
			"GET", "/cdn/zones/z1/purge-cache", "", nil},
		{"cdn rotate", 200, `{"token_auth_secret":"s"}`, func(c *Client) error { _, err := c.CDN.RotateTokenSecret(ctx, "z1"); return err },
			"POST", "/cdn/zones/z1/token-auth/rotate-secret", "", nil},
		{"cdn sign", 200, `{"signed_url":"u"}`, func(c *Client) error {
			_, err := c.CDN.SignURL(ctx, "z1", &CDNSignURLRequest{Path: "/v.mp4", ExpiresIn: 60})
			return err
		}, "POST", "/cdn/zones/z1/token-auth/sign-url", "", map[string]interface{}{"path": "/v.mp4", "expires_in": 60.0}},
		{"cdn metrics", 200, `{}`, func(c *Client) error {
			_, err := c.CDN.GetMetrics(ctx, "z1", CDNMetricTopCountries, &CDNMetricsParams{Minutes: 30, Limit: 5, StatusRange: "5xx", PathPrefix: "/api"})
			return err
		}, "GET", "/cdn/zones/z1/metrics/top-countries", "limit=5&minutes=30&path_prefix=%2Fapi&status_range=5xx", nil},

		// DNS
		{"dns regions", 200, `[]`, func(c *Client) error { _, err := c.DNS.ListRegions(ctx); return err },
			"GET", "/dns/regions", "", nil},
		{"dns move", 200, `{}`, func(c *Client) error { return c.DNS.MoveZoneToProject(ctx, "z1", 3) },
			"POST", "/dns/zones/z1/move-project", "", map[string]interface{}{"project_id": 3.0}},
		{"dns scan create", 201, `{"imported":0,"skipped":0,"errors":[],"records":[]}`, func(c *Client) error {
			_, err := c.DNS.CreateZoneFromScan(ctx, "example.com", 3)
			return err
		}, "POST", "/dns/zones/scan", "domain=example.com&project_id=3", nil},
		{"dns health checks", 200, `[]`, func(c *Client) error { _, err := c.DNS.ListHealthChecks(ctx, "z1"); return err },
			"GET", "/dns/zones/z1/health-checks", "", nil},
		{"dns health check get", 200, `{"uuid":"h1"}`, func(c *Client) error { _, err := c.DNS.GetHealthCheck(ctx, "z1", "r1"); return err },
			"GET", "/dns/zones/z1/records/r1/health-check", "", nil},
		{"dns health check set", 200, `{"uuid":"h1"}`, func(c *Client) error {
			_, err := c.DNS.SetHealthCheck(ctx, "z1", "r1", &DNSHealthCheckRequest{Name: "web", CheckType: DNSHealthCheckTCP, Port: 443})
			return err
		}, "PUT", "/dns/zones/z1/records/r1/health-check", "", map[string]interface{}{"name": "web", "check_type": "tcp", "port": 443.0}},
		{"dns health check delete", 204, ``, func(c *Client) error { return c.DNS.DeleteHealthCheck(ctx, "z1", "r1") },
			"DELETE", "/dns/zones/z1/records/r1/health-check", "", nil},

		// Load balancer, Kubernetes, SSH keys, projects
		{"lb protection", 200, `{}`, func(c *Client) error { return c.LoadBalancer.SetProtection(ctx, "l1", true) },
			"POST", "/loadbalancer/l1/protection", "", map[string]interface{}{"enabled": true}},
		{"lb move", 200, `{}`, func(c *Client) error { return c.LoadBalancer.MoveToProject(ctx, "l1", 3) },
			"POST", "/loadbalancer/l1/move-project", "", map[string]interface{}{"project_id": 3.0}},
		{"lb batch", 201, `{"targets":[{"uuid":"t1"},{"uuid":"t2"}]}`, func(c *Client) error {
			_, err := c.LoadBalancer.AddTargets(ctx, "l1", "li1", []AddTargetRequest{{TargetType: "vps", TargetUUID: "7", Weight: 100}})
			return err
		}, "POST", "/loadbalancer/l1/listeners/li1/targets/batch", "", map[string]interface{}{"targets": []interface{}{map[string]interface{}{"target_type": "vps", "target_uuid": "7", "weight": 100.0}}}},
		{"k8s protection", 200, `{}`, func(c *Client) error { return c.Kubernetes.SetProtection(ctx, "k1", false) },
			"POST", "/kubernetes/k1/protection", "", map[string]interface{}{"enabled": false}},
		{"k8s metrics", 200, `{"metrics":{}}`, func(c *Client) error { _, err := c.Kubernetes.GetMetrics(ctx, "k1", "24h"); return err },
			"GET", "/kubernetes/k1/metrics", "time_range=24h", nil},
		{"k8s node metrics", 200, `{"metrics":{}}`, func(c *Client) error { _, err := c.Kubernetes.GetNodeMetrics(ctx, "k1", "node-1", ""); return err },
			"GET", "/kubernetes/k1/nodes/node-1/metrics", "", nil},
		{"ssh key update", 200, `{"detail":"ok","sshkey":{"id":4,"name":"new"}}`, func(c *Client) error { _, err := c.SSHKeys.Update(ctx, 4, "new"); return err },
			"PUT", "/sshkey/4", "", map[string]interface{}{"name": "new"}},
		{"project update", 200, `{}`, func(c *Client) error { return c.Projects.Update(ctx, 3, "prod") },
			"PUT", "/projects/3", "", map[string]interface{}{"name": "prod"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rec recorded
			c := newTestClient(t, tc.status, tc.resp, &rec)
			if err := tc.call(c); err != nil {
				t.Fatal(err)
			}
			if rec.Method != tc.method || rec.Path != tc.path || rec.Query != tc.query {
				t.Fatalf("got %s %s ?%s, want %s %s ?%s", rec.Method, rec.Path, rec.Query, tc.method, tc.path, tc.query)
			}
			if tc.body != nil {
				want, _ := json.Marshal(tc.body)
				got, _ := json.Marshal(rec.Body)
				if string(want) != string(got) {
					t.Fatalf("body %s, want %s", got, want)
				}
			}
		})
	}
}

func TestSSHKeyBodiesAreArrays(t *testing.T) {
	ctx := context.Background()
	for name, call := range map[string]func(c *Client) error{
		"/vps/7/ssh-keys":       func(c *Client) error { return c.VPS.AddSSHKeys(ctx, 7, []int{4, 5}) },
		"/baremetal/7/ssh-keys": func(c *Client) error { return c.Baremetal.AddSSHKeys(ctx, 7, []int{4, 5}) },
	} {
		var rec rawRecorded
		c := newRawTestClient(t, 200, `{"detail":"ok"}`, &rec)
		if err := call(c); err != nil {
			t.Fatal(err)
		}
		if rec.Method != http.MethodPost || rec.Path != name || string(rec.Body) != "[4,5]" {
			t.Fatalf("got %s %s %s", rec.Method, rec.Path, rec.Body)
		}
	}
	c, _ := NewClient("token")
	if err := c.VPS.AddSSHKeys(ctx, 7, nil); err == nil {
		t.Fatal("want an error for an empty list")
	}
}

func TestDNSZoneFileUploadIsMultipart(t *testing.T) {
	var rec rawRecorded
	c := newRawTestClient(t, 201, `{"imported":1,"skipped":1,"errors":["line 3: unknown type"],"records":[{"uuid":"r1","name":"www.example.com","record_type":"A","content":"1.2.3.4","ttl":300,"region":null}]}`, &rec)
	zone := []byte("www 300 IN A 1.2.3.4\n")
	res, err := c.DNS.CreateZoneFromFile(context.Background(), "example.com", 3, zone)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/dns/zones/upload" || rec.Query != "domain=example.com&project_id=3" {
		t.Fatalf("got %+v", rec)
	}
	mediaType, params, err := mime.ParseMediaType(rec.ContentType)
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type %q", rec.ContentType)
	}
	form, err := multipart.NewReader(strings.NewReader(string(rec.Body)), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("form %+v", form)
	}
	f, _ := files[0].Open()
	got, _ := io.ReadAll(f)
	if string(got) != string(zone) {
		t.Fatalf("file %q", got)
	}
	if res.Imported != 1 || res.Errors[0] != "line 3: unknown type" || res.Records[0].RecordType != "A" || res.Records[0].Region != nil {
		t.Fatalf("decoded %+v", res)
	}

	if _, err := c.DNS.ImportZoneFile(context.Background(), "z1", zone); err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/dns/zones/z1/import" || !strings.HasPrefix(rec.ContentType, "multipart/form-data") {
		t.Fatalf("got %+v", rec)
	}
}

func TestDNSScanZoneDecodesStringErrors(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"imported":0,"skipped":0,"errors":["CNAME conflict"],"records":[{"uuid":"preview-www-A","name":"www.example.com","record_type":"A","content":"1.2.3.4","ttl":300}]}`, &rec)
	res, err := c.DNS.ScanZone(context.Background(), "z1", false)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Query != "auto_import=false" || res.Errors[0] != "CNAME conflict" || res.Records[0].Content != "1.2.3.4" {
		t.Fatalf("got %+v %+v", rec, res)
	}
}

func TestDDoSListAttacksWithoutAttacks(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"No recent DDoS attacks were found for your IPs."}`, &rec)
	attacks, err := c.DDoS.ListAttacks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if attacks == nil || len(attacks) != 0 {
		t.Fatalf("got %+v", attacks)
	}
}

func TestDDoSListAttacksDecimalPeaks(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `[{"attack_id":7,"ip_address":"194.26.100.5","start_time":"2026-09-01 10:00:00","duration":12.5,"packets_second_peak":1500000,"gbps_peak":2.75,"status":"ended"}]`, &rec)
	attacks, err := c.DDoS.ListAttacks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(attacks) != 1 || attacks[0].Duration != 12.5 || attacks[0].GbpsPeak != 2.75 || attacks[0].PacketsSecondPeak != 1500000 {
		t.Fatalf("got %+v", attacks)
	}
}

func TestDDoSCreatePrefixListReadsItBack(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(201)
			_, _ = io.WriteString(w, `{"detail":"Prefix list created"}`)
			return
		}
		_, _ = io.WriteString(w, `{"prefix_lists":[{"uuid":"g1","name":"office","is_global":true},{"uuid":"p1","name":"office","is_global":false,"entries_count":0}],"total":2}`)
	}))
	defer srv.Close()
	c, _ := NewClient("token", WithBaseURL(srv.URL), WithMaxRetries(0))
	list, err := c.DDoSMitigation.CreatePrefixList(context.Background(), "office", "")
	if err != nil {
		t.Fatal(err)
	}
	if list.UUID != "p1" || calls != 2 {
		t.Fatalf("got %+v after %d calls", list, calls)
	}
}

func TestTranscodeOutputSpecFlattensParams(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 201, `{"uuid":"j1","status":"queued","spec":{"outputs":[{"type":"file","codec":"h264","height":360}]},"outputs":null,"progress":0}`, &rec)
	job, err := c.Transcoder.CreateJob(context.Background(), &CreateTranscodeJobRequest{
		Input:   TranscodeInput{Source: "url", URL: "https://example.com/in.mp4"},
		Output:  TranscodeDestination{S3: TranscodeS3{Bucket: "out", Path: "videos/", AccessKey: "a", SecretKey: "s"}},
		Outputs: []TranscodeOutputSpec{{Type: "file", Params: map[string]interface{}{"codec": "h264", "height": 360}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Method != http.MethodPost || rec.Path != "/transcoder/jobs" {
		t.Fatalf("got %+v", rec)
	}
	outputs := rec.Body["outputs"].([]interface{})
	first := outputs[0].(map[string]interface{})
	if first["type"] != "file" || first["codec"] != "h264" || first["height"] != 360.0 {
		t.Fatalf("outputs %v", outputs)
	}
	input := rec.Body["input"].(map[string]interface{})
	if input["source"] != "url" || input["s3"] != nil {
		t.Fatalf("input %v", input)
	}
	if job.UUID != "j1" || job.Spec.Outputs[0].Type != "file" || job.Spec.Outputs[0].Params["codec"] != "h264" || job.Outputs != nil {
		t.Fatalf("decoded %+v", job)
	}
}

func TestManagedDatabaseDecodes(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"uuid":"m1","project_id":3,"name":"db","label":null,"engine":"postgresql","version":"17.5.0","topology":"replication","replicas":2,"status":"active","endpoint_host":"45.90.1.2","endpoint_port":5432,"plan":{"uuid":"p1","name":"postgresql.micro","engine":"postgresql","cpu":2,"memory_gb":4,"storage_gb":60,"max_replicas":5,"price_per_hour":0.02778},"location":{"id":1,"location_name":"eu-bcn-1","description":"Barcelona"},"backup_enabled":false,"backup_schedule_cron":null,"backup_retention_days":7,"billing_type":"hourly","protected":true,"updated_at":"2026-09-30T10:00:00"}`, &rec)
	db, err := c.ManagedDatabases.Get(context.Background(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if rec.Path != "/managed-databases/m1" || db.Engine != "postgresql" || *db.EndpointPort != 5432 || db.Plan.MaxReplicas != 5 || db.Location.LocationName != "eu-bcn-1" || !db.Protected || db.Label != nil {
		t.Fatalf("decoded %+v", db)
	}

	c = newTestClient(t, 200, `{"start":1,"end":2,"metrics":{"cpu":[[1,0.5],[2,1]],"memory":[]}}`, &rec)
	m, err := c.ManagedDatabases.GetMetrics(context.Background(), "m1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Query != "" || m.Metrics["cpu"][1][1] != 1 || len(m.Metrics["memory"]) != 0 {
		t.Fatalf("decoded %+v", m)
	}
}

func TestVPSPlansDecode(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"locations":[{"location_name":"eu-bcn-1","description":"Barcelona, Spain","clusters":[{"cluster_name":"General Purpose","type":"shared","plans":[{"plan_name":"gp.nano","ram":2048,"cpu":1,"storage":40,"bandwidth":3,"price_per_hour":"0.00556","status":2}]}]}]}`, &rec)
	plans, err := c.VPS.Plans(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	p := plans.Locations[0].Clusters[0].Plans[0]
	if p.PlanName != "gp.nano" || p.PricePerHour != "0.00556" || p.Status != 2 {
		t.Fatalf("decoded %+v", p)
	}
}

func TestVPSCreateAlwaysSendsLabel(t *testing.T) {
	var rec recorded
	c := newTestClient(t, 200, `{"detail":"ok"}`, &rec)
	if _, err := c.VPS.Create(context.Background(), 1, &CreateVPSRequest{Name: "a", PlanName: "gp.nano", TemplateName: "debian-12", LocationName: "eu-bcn-1"}); err != nil {
		t.Fatal(err)
	}
	if v, ok := rec.Body["label"]; !ok || v != "" {
		t.Fatalf("label not sent: %v", rec.Body)
	}
}
