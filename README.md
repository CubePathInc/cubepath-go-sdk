# CubePath Go SDK

Official Go client library for the [CubePath](https://cubepath.com) cloud infrastructure API.

CubePath is a cloud infrastructure provider offering virtual private servers (VPS), bare metal servers, managed Kubernetes, load balancers, CDN, DNS hosting, private networking, and DDoS protection across multiple datacenter locations.

## Installation

```bash
go get github.com/CubePathInc/cubepath-go-sdk
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/CubePathInc/cubepath-go-sdk"
)

func main() {
    client, err := cubepath.NewClient("your-api-token")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // List all projects
    projects, err := client.Projects.List(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, p := range projects {
        fmt.Printf("Project: %s (ID: %d)\n", p.Project.Name, p.Project.ID)
    }
}
```

## Authentication

All API requests require a Bearer token. You can generate one from the [CubePath dashboard](https://cubepath.com).

```go
client, err := cubepath.NewClient("your-api-token")
```

The token is sent as an `Authorization: Bearer <token>` header on every request.

## Configuration

The client can be customized using functional options:

```go
client, err := cubepath.NewClient("your-api-token",
    cubepath.WithBaseURL("https://api.custom.com"),
    cubepath.WithMaxRetries(5),
    cubepath.WithRetryWaitMin(2 * time.Second),
    cubepath.WithRetryWaitMax(60 * time.Second),
    cubepath.WithHTTPClient(customHTTPClient),
    cubepath.WithUserAgent("my-app/1.0"),
    cubepath.WithRateLimiter(customRateLimiter),
)
```

### Defaults

| Setting | Default |
|---------|---------|
| Base URL | `https://api.cubepath.com` |
| HTTP timeout | 30 seconds |
| Max retries | 3 |
| Retry wait (min) | 1 second |
| Retry wait (max) | 30 seconds |
| Rate limit | 10 requests/second |

Retries use exponential backoff with jitter and are triggered on 429 (rate limited) and 5xx (server error) responses.

## Services

The client exposes the following services:

### Compute

#### VPS

```go
// Create a VPS
task, err := client.VPS.Create(ctx, projectID, &cubepath.CreateVPSRequest{
    Name:         "web-server",
    PlanName:     "cp.4-8-160",
    TemplateName: "ubuntu-24.04",
    LocationName: "MIA",
    SSHKeyNames:  []string{"my-key"},
})

// List all VPS instances (grouped by project)
projects, err := client.VPS.List(ctx)

// Get a specific VPS
vps, err := client.VPS.Get(ctx, 12345)

// Power operations (start_vps, stop_vps, restart_vps, reset_vps)
err = client.VPS.Power(ctx, 12345, "restart_vps")

// Resize
err = client.VPS.Resize(ctx, 12345, "cp.8-16-320")

// Destroy
err = client.VPS.Destroy(ctx, 12345, true) // true = release floating IPs
```

#### VPS Backups

```go
// List backups
backups, err := client.VPS.Backups().List(ctx, vpsID)

// Create a backup
err = client.VPS.Backups().Create(ctx, vpsID, &cubepath.CreateVPSBackupRequest{
    Notes: "before upgrade",
})

// Restore from backup
err = client.VPS.Backups().Restore(ctx, vpsID, backupID)

// Configure automatic backups
err = client.VPS.Backups().UpdateSettings(ctx, vpsID, &cubepath.UpdateVPSBackupSettingsRequest{
    Enabled:       true,
    ScheduleHour:  3,
    RetentionDays: 7,
    MaxBackups:    5,
})
```

#### VPS ISOs

```go
// List available ISOs
isos, err := client.VPS.ISOs().List(ctx, vpsID)

// Mount an ISO
err = client.VPS.ISOs().Mount(ctx, vpsID, "iso-uuid")

// Unmount
err = client.VPS.ISOs().Unmount(ctx, vpsID)
```

#### Bare Metal

```go
// Deploy a bare metal server
task, err := client.Baremetal.Deploy(ctx, projectID, &cubepath.CreateBaremetalRequest{
    ModelName:    "E-2388G",
    LocationName: "MIA",
    Hostname:     "db-server",
    Password:     "secure-password",
    OSName:       "ubuntu-24.04",
})

// Power operations (start_metal, stop_metal, restart_metal)
err = client.Baremetal.Power(ctx, bmID, "restart_metal")

// Activate rescue mode
rescue, err := client.Baremetal.Rescue(ctx, bmID)
fmt.Printf("Username: %s, Password: %s\n", rescue.Username, rescue.Password)

// Read BMC sensors
sensors, err := client.Baremetal.BMCSensors(ctx, bmID)

// Create IPMI proxy session
session, err := client.Baremetal.IPMISession(ctx, bmID)
fmt.Printf("IPMI URL: %s\n", session.ProxyURL)

// Enable/disable monitoring
err = client.Baremetal.MonitoringEnable(ctx, bmID)
err = client.Baremetal.MonitoringDisable(ctx, bmID)

// Reinstall OS
err = client.Baremetal.Reinstall(ctx, bmID, &cubepath.ReinstallBaremetalRequest{
    OSName:   "debian-12",
    Password: "new-password",
})
```

### Kubernetes

```go
// List available versions
versions, err := client.Kubernetes.ListVersions(ctx)

// List compatible plans
plans, err := client.Kubernetes.ListPlans(ctx, "1.31")

// Create a cluster
cluster, err := client.Kubernetes.Create(ctx, &cubepath.CreateKubernetesClusterRequest{
    ProjectID:      1,
    Name:           "production",
    LocationName:   "MIA",
    Version:        "1.31",
    HAControlPlane: true,
    NodePools: []cubepath.CreateNodePoolConfig{
        {Name: "workers", Plan: "cp.4-8-160", Count: 3},
    },
})

// Get cluster details
cluster, err := client.Kubernetes.Get(ctx, "cluster-uuid")

// Download kubeconfig
kubeconfig, err := client.Kubernetes.GetKubeconfig(ctx, "cluster-uuid")

// Move cluster to another project
err = client.Kubernetes.Move(ctx, "cluster-uuid", newProjectID)

// Node pool management
pool, err := client.Kubernetes.CreateNodePool(ctx, "cluster-uuid", &cubepath.CreateNodePoolRequest{
    Name:      "gpu-pool",
    Plan:      "cp.8-32-640",
    Count:     2,
    AutoScale: true,
    Labels:    map[string]string{"workload": "ml"},
    Taints: []cubepath.NodeTaint{
        {Key: "gpu", Value: "true", Effect: "NoSchedule"},
    },
})

// Scale nodes
err = client.Kubernetes.AddNodes(ctx, "cluster-uuid", "pool-uuid", 2)
err = client.Kubernetes.RemoveNode(ctx, "cluster-uuid", "pool-uuid", "vps-id")

// Addon management
addons, err := client.Kubernetes.ListAvailableAddons(ctx)
err = client.Kubernetes.InstallAddon(ctx, "cluster-uuid", "cert-manager", nil)
err = client.Kubernetes.UninstallAddon(ctx, "cluster-uuid", "addon-uuid")
```

### Networking

#### Private Networks

```go
network, err := client.Networks.Create(ctx, &cubepath.CreateNetworkRequest{
    Name:         "internal",
    LocationName: "MIA",
    IPRange:      "10.0.0.0",
    Prefix:       24,
    ProjectID:    1,
})
```

#### Floating IPs

```go
// Acquire a new IP
ip, err := client.FloatingIPs.Acquire(ctx, "IPv4", "MIA")

// Assign to a VPS
err = client.FloatingIPs.Assign(ctx, "vps", vpsID, ip.Address)

// Configure reverse DNS
err = client.FloatingIPs.ConfigureReverseDNS(ctx, "203.0.113.10", "mail.example.com")

// Release
err = client.FloatingIPs.Release(ctx, "203.0.113.10")
```

#### Firewall

```go
group, err := client.Firewall.Create(ctx, &cubepath.CreateFirewallGroupRequest{
    Name:    "web-servers",
    Enabled: true,
    Rules: []cubepath.FirewallRule{
        {Direction: "in", Protocol: "tcp", Port: strPtr("80")},
        {Direction: "in", Protocol: "tcp", Port: strPtr("443")},
    },
})
```

### DNS

```go
// Create a zone
zone, err := client.DNS.CreateZone(ctx, &cubepath.CreateDNSZoneRequest{
    Domain: "example.com",
})

// Verify zone setup
verify, err := client.DNS.VerifyZone(ctx, zone.UUID)

// Scan and auto-import records
scan, err := client.DNS.ScanZone(ctx, zone.UUID, true)

// Create a record
record, err := client.DNS.CreateRecord(ctx, zone.UUID, &cubepath.CreateDNSRecordRequest{
    Name:    "www",
    Type:    "A",
    Content: "203.0.113.10",
    TTL:     3600,
})

// Manage SOA record
soa, err := client.DNS.GetSOA(ctx, zone.UUID)
soa, err = client.DNS.UpdateSOA(ctx, zone.UUID, &cubepath.UpdateSOARequest{
    Refresh: intPtr(7200),
})
```

### Load Balancers

```go
// Create a load balancer
lb, err := client.LoadBalancer.Create(ctx, &cubepath.CreateLoadBalancerRequest{
    Name:         "web-lb",
    PlanName:     "lb-small",
    LocationName: "MIA",
})

// Add a listener
listener, err := client.LoadBalancer.CreateListener(ctx, lb.UUID, &cubepath.CreateListenerRequest{
    Name:       "http",
    Protocol:   "tcp",
    SourcePort: 80,
    TargetPort: 8080,
    Algorithm:  "round_robin",
})

// Add targets
target, err := client.LoadBalancer.AddTarget(ctx, lb.UUID, listener.UUID, &cubepath.AddTargetRequest{
    TargetType: "vps",
    TargetUUID: "vps-uuid",
    Weight:     100,
})

// Configure health check
err = client.LoadBalancer.ConfigureHealthCheck(ctx, lb.UUID, listener.UUID, &cubepath.HealthCheckConfig{
    Protocol:           "http",
    Path:               "/health",
    IntervalSeconds:    10,
    TimeoutSeconds:     5,
    HealthyThreshold:   3,
    UnhealthyThreshold: 3,
    ExpectedCodes:      "200",
})

// Drain a target before removal
err = client.LoadBalancer.DrainTarget(ctx, lb.UUID, listener.UUID, target.UUID)
```

### CDN

```go
// Create a CDN zone
zone, err := client.CDN.CreateZone(ctx, &cubepath.CreateCDNZoneRequest{
    Name:     "my-cdn",
    PlanName: "cdn-starter",
})

// Add an origin
origin, err := client.CDN.CreateOrigin(ctx, zone.UUID, &cubepath.CreateCDNOriginRequest{
    Name:               "primary",
    Address:            "origin.example.com",
    Port:               intPtr(443),
    Protocol:           "https",
    Weight:             100,
    Priority:           1,
    HealthCheckEnabled: true,
    HealthCheckPath:    "/health",
    VerifySSL:          true,
    Enabled:            true,
})

// Create edge rules
rule, err := client.CDN.CreateRule(ctx, zone.UUID, &cubepath.CreateCDNRuleRequest{
    Name:     "cache-static",
    RuleType: "cache",
    Priority: 1,
    Enabled:  true,
    ActionConfig: json.RawMessage(`{"cache_ttl": 86400}`),
})

// WAF rules
waf, err := client.CDN.CreateWAFRule(ctx, zone.UUID, &cubepath.CreateCDNRuleRequest{
    Name:     "block-bots",
    RuleType: "block",
    Priority: 1,
    Enabled:  true,
    ActionConfig: json.RawMessage(`{"action": "block"}`),
})

// Query metrics
metrics, err := client.CDN.GetMetrics(ctx, zone.UUID, "summary", &cubepath.CDNMetricsParams{
    Minutes: 60,
})
```

Available metric types: `summary`, `requests`, `bandwidth`, `cache`, `status-codes`, `top-urls`, `top-countries`, `top-asn`, `top-user-agents`, `blocked`, `pops`, `file-extensions`.

### Other Services

#### Projects

```go
project, err := client.Projects.Create(ctx, &cubepath.CreateProjectRequest{
    Name:        "production",
    Description: "Production environment",
})
projects, err := client.Projects.List(ctx)
```

#### SSH Keys

```go
key, err := client.SSHKeys.Create(ctx, &cubepath.CreateSSHKeyRequest{
    Name:   "deploy-key",
    SSHKey: "ssh-ed25519 AAAA...",
})
keys, err := client.SSHKeys.List(ctx)
```

#### Pricing

```go
pricing, err := client.Pricing.Get(ctx)
```

#### DDoS Attacks

```go
attacks, err := client.DDoS.ListAttacks(ctx)
```

## Error Handling

API errors are returned as `*cubepath.APIError` with classification helpers:

```go
vps, err := client.VPS.Get(ctx, 99999)
if err != nil {
    if cubepath.IsNotFound(err) {
        fmt.Println("VPS not found")
    } else if cubepath.IsRateLimited(err) {
        fmt.Println("Rate limited, try again later")
    } else if cubepath.IsBadRequest(err) {
        fmt.Println("Invalid request:", err)
    } else {
        fmt.Println("Error:", err)
    }
}
```

You can also inspect the error directly:

```go
var apiErr *cubepath.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Detail)
}
```

## Related Projects

| Project | Description |
|---------|-------------|
| [cubecli](https://github.com/CubePathInc/cubecli) | Official CLI tool for CubePath |
| [terraform-provider-cubepath](https://github.com/CubePathInc/terraform-provider-cubepath) | Terraform provider for CubePath |
| [cubepath.ansible](https://github.com/CubePathInc/cubepath.ansible) | Ansible collection for CubePath |

## License

MIT
