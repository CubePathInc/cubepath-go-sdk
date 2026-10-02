# CubePath Go SDK

Official Go client library for the [CubePath](https://cubepath.com) cloud infrastructure API.

CubePath is a cloud infrastructure provider offering virtual private servers (VPS), bare metal servers, managed Kubernetes, managed databases, load balancers, CDN, S3 compatible object storage, video transcoding, DNS hosting, private networking, cloud alerts, and DDoS protection across multiple datacenter locations.

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
    PlanName:     "gp.nano",
    TemplateName: "debian-12",
    LocationName: "us-mia-1",
    SSHKeyIDs:    []int{12},
})

// List all VPS instances (grouped by project)
projects, err := client.VPS.List(ctx)

// Get a specific VPS
vps, err := client.VPS.Get(ctx, 12345)

// Power operations (start_vps, stop_vps, restart_vps, reset_vps)
err = client.VPS.Power(ctx, 12345, "restart_vps")

// Resize
err = client.VPS.Resize(ctx, 12345, "gp.pro")

// Destroy
err = client.VPS.Destroy(ctx, 12345, true) // true = release floating IPs

// Plans per location (Status 2 = orderable, 1 = out of stock)
plans, err := client.VPS.Plans(ctx)

// Destruction protection and moving between projects
err = client.VPS.SetProtection(ctx, 12345, true)
err = client.VPS.MoveToProject(ctx, 12345, otherProjectID)

// SSH keys (installed on the next reinstall) and private network
err = client.VPS.AddSSHKeys(ctx, 12345, []int{12, 47})
err = client.VPS.RemoveSSHKey(ctx, 12345, 47)
err = client.VPS.AttachNetwork(ctx, 12345, networkID) // restart the VPS to apply
err = client.VPS.DetachNetwork(ctx, 12345)

// Console (noVNC): connect to WebSocketURL, the ticket is the VNC password
session, err := client.VPS.VNCURL(ctx, 12345)
```

#### Availability Groups

The VPS of a group are placed on different physical nodes.

```go
groups := client.VPS.AvailabilityGroups()
group, err := groups.Create(ctx, &cubepath.CreateAvailabilityGroupRequest{
    ProjectID:    projectID,
    Name:         "web",
    LocationName: "us-mia-1",
})
err = groups.AddVPS(ctx, group.UUID, vpsID)
list, err := groups.ListByProject(ctx, projectID)
err = groups.RemoveVPS(ctx, group.UUID, vpsID)
err = groups.Delete(ctx, group.UUID) // the group must be empty
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
    ModelName:    "c1.metal.plus",
    LocationName: "us-mia-1",
    Hostname:     "db-server",
    Password:     "secure-password",
    OSName:       "debian-12",
})

// Power operations (start_metal, stop_metal, restart_metal)
err = client.Baremetal.Power(ctx, bmID, "restart_metal")

// Activate rescue mode
rescue, err := client.Baremetal.Rescue(ctx, bmID)
fmt.Printf("Username: %s, Password: %s\n", rescue.Username, rescue.Password)

// Read BMC sensors (temperatures in CELSIUS, fans in RPM; LastSeen is the last BMC poll)
sensors, err := client.Baremetal.BMCSensors(ctx, bmID)

// Reinstall progress (the server status is "deploying" while it runs) and cancel
status, err := client.Baremetal.ReinstallStatus(ctx, bmID)
err = client.Baremetal.CancelReinstall(ctx, bmID)

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

// Catalog: models with stock per location, and the systems a server can install
models, err := client.Baremetal.ListModels(ctx)
systems, err := client.Baremetal.ListOS(ctx, bmID)

// KVM console access
kvm, err := client.Baremetal.GetKVM(ctx, bmID)

// Protection, projects, SSH keys and private network
err = client.Baremetal.SetProtection(ctx, bmID, true)
err = client.Baremetal.MoveToProject(ctx, bmID, otherProjectID)
err = client.Baremetal.AddSSHKeys(ctx, bmID, []int{12})
err = client.Baremetal.RemoveSSHKey(ctx, bmID, 12)
err = client.Baremetal.AttachNetwork(ctx, bmID, networkID)
err = client.Baremetal.DetachNetwork(ctx, bmID)
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
    LocationName:   "us-mia-1",
    Version:        "1.31",
    HAControlPlane: true,
    NodePools: []cubepath.CreateNodePoolConfig{
        {Name: "workers", Plan: "gp.small", Count: 3},
    },
})

// Get cluster details
cluster, err := client.Kubernetes.Get(ctx, "cluster-uuid")

// Download kubeconfig
kubeconfig, err := client.Kubernetes.GetKubeconfig(ctx, "cluster-uuid")

// Move cluster to another project
err = client.Kubernetes.Move(ctx, "cluster-uuid", newProjectID)

// Destruction protection
err = client.Kubernetes.SetProtection(ctx, "cluster-uuid", true)

// Metrics: [timestamp, value] series (time range "1h" to "30d")
metrics, err := client.Kubernetes.GetMetrics(ctx, "cluster-uuid", "24h")
nodeMetrics, err := client.Kubernetes.GetNodeMetrics(ctx, "cluster-uuid", "node-name", "1h")

// Node pool management
pool, err := client.Kubernetes.CreateNodePool(ctx, "cluster-uuid", &cubepath.CreateNodePoolRequest{
    Name:      "gpu-pool",
    Plan:      "gp.pro",
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
    LocationName: "us-mia-1",
    IPRange:      "10.0.0.0",
    Prefix:       24,
    ProjectID:    1,
})

err = client.Networks.MoveToProject(ctx, network.ID, otherProjectID)

// Dynamic Routes: eBGP sessions between the network gateway (ASN 64512) and a server of the
// network. The network needs at least one attached server.
peer, err := client.Networks.CreateBGPPeer(ctx, network.ID, &cubepath.CreateBGPPeerRequest{
    PeerType:   cubepath.BGPPeerTypeVPS,
    PeerTarget: "12345", // VPS id, or an IP of the network with PeerTypeIP
    RemoteASN:  65010,
})
peers, err := client.Networks.ListBGPPeers(ctx, network.ID) // with session state and received prefixes
err = client.Networks.DeleteBGPPeer(ctx, network.ID, peer.PeerID)
```

#### Floating IPs

```go
// Acquire a new IP
ip, err := client.FloatingIPs.Acquire(ctx, "IPv4", "us-mia-1")

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
    ProjectID: projectID,
    Name:    "web-servers",
    Enabled: true,
    Rules: []cubepath.FirewallRule{
        {Direction: "in", Protocol: "tcp", Port: strPtr("80")},
        {Direction: "in", Protocol: "tcp", Port: strPtr("443")},
    },
})

// Replace the groups of a VPS (at most 10, in priority order; an empty list removes them)
res, err := client.Firewall.AssignToVPS(ctx, vpsID, &cubepath.VPSFirewallGroupsRequest{
    FirewallGroupIDs: []int{group.ID},
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

// Import a BIND zone file into a zone, or create a zone from one
result, err := client.DNS.ImportZoneFile(ctx, zone.UUID, zoneFile)
fmt.Println(result.Imported, result.Skipped, result.Errors)
result, err = client.DNS.CreateZoneFromFile(ctx, "example.org", projectID, zoneFile)

// Create a zone with the records found in public DNS
result, err = client.DNS.CreateZoneFromScan(ctx, "example.net", projectID)

// GeoDNS regions (Pro and Business plans) and moving a zone
regions, err := client.DNS.ListRegions(ctx)
err = client.DNS.MoveZoneToProject(ctx, zone.UUID, otherProjectID)

// Health checks on A/AAAA records (Pro and Business plans): an unhealthy target is left out
// of the answers until it recovers
check, err := client.DNS.SetHealthCheck(ctx, zone.UUID, record.UUID, &cubepath.DNSHealthCheckRequest{
    Name:      "web",
    CheckType: cubepath.DNSHealthCheckHTTPS,
    Path:      "/health",
})
checks, err := client.DNS.ListHealthChecks(ctx, zone.UUID)
err = client.DNS.DeleteHealthCheck(ctx, zone.UUID, record.UUID)
```

### Load Balancers

```go
// Create a load balancer
lb, err := client.LoadBalancer.Create(ctx, &cubepath.CreateLoadBalancerRequest{
    Name:         "web-lb",
    PlanName:     "lb.small",
    LocationName: "us-mia-1",
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

// Add up to 50 targets at once
targets, err := client.LoadBalancer.AddTargets(ctx, lb.UUID, listener.UUID, []cubepath.AddTargetRequest{
    {TargetType: "vps", TargetUUID: "123", Weight: 100},
    {TargetType: "vps", TargetUUID: "124", Weight: 100},
})

// Protection and moving between projects
err = client.LoadBalancer.SetProtection(ctx, lb.UUID, true)
err = client.LoadBalancer.MoveToProject(ctx, lb.UUID, otherProjectID)
```

### CDN

```go
// Create a CDN zone
zone, err := client.CDN.CreateZone(ctx, &cubepath.CreateCDNZoneRequest{
    Name:     "my-cdn",
    PlanName: "cdn.starter",
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

// Query metrics, optionally filtered
metrics, err := client.CDN.GetMetrics(ctx, zone.UUID, cubepath.CDNMetricSummary, &cubepath.CDNMetricsParams{
    Minutes:     60,
    StatusRange: "5xx",
    Country:     "ES,FR",
})

// Purge the cache: some paths (a trailing * purges a prefix) or everything
purge, err := client.CDN.PurgeCache(ctx, zone.UUID, &cubepath.CDNPurgeRequest{
    Paths: []string{"/assets/app.css", "/images/*"},
})
purges, err := client.CDN.ListPurges(ctx, zone.UUID) // progress per location

// Token Auth: enable it (the secret is returned once), then sign URLs
enabled := true
updated, err := client.CDN.UpdateZone(ctx, zone.UUID, &cubepath.UpdateCDNZoneRequest{TokenAuthEnabled: &enabled})
signed, err := client.CDN.SignURL(ctx, zone.UUID, &cubepath.CDNSignURLRequest{
    Path:      "/videos/clip.mp4",
    ExpiresIn: 3600,
})
fmt.Println(signed.SignedURL)
secret, err := client.CDN.RotateTokenSecret(ctx, zone.UUID) // invalidates every signed URL
```

Available metric types (`CDNMetric*` constants): `summary`, `requests`, `bandwidth`, `cache`, `status-codes`, `top-urls`, `top-countries`, `top-asn`, `top-user-agents`, `blocked`, `pops`, `file-extensions`.

### Object Storage

S3 compatible buckets. Buckets and access keys are created asynchronously: they start as
`pending` and are `active` a few seconds later.

```go
// Tiers, with endpoint, prices and free tier
tiers, err := client.ObjectStorage.ListTiers(ctx)

// Create a bucket
bucket, err := client.ObjectStorage.CreateBucket(ctx, &cubepath.CreateObjectStorageBucketRequest{
    Name: "my-backups",
    Tier: "infrequent_access",
})
detail, err := client.ObjectStorage.GetBucket(ctx, bucket.UUID) // poll until Status == "active"

// Create an access key for S3 clients (the secret is only returned here)
key, err := client.ObjectStorage.CreateKey(ctx, &cubepath.CreateObjectStorageKeyRequest{
    Name:        "backup-job",
    Tier:        "infrequent_access",
    Permission:  "read_write", // or "read_only"
    BucketUUIDs: []string{bucket.UUID}, // omit for every bucket of the project
})
fmt.Println(key.AccessKeyID, key.SecretAccessKey, key.Endpoint, key.Region)

// Versioning and deletion protection
enabled := "enabled"
err = client.ObjectStorage.UpdateBucket(ctx, bucket.UUID, &cubepath.UpdateObjectStorageBucketRequest{Versioning: &enabled})

// Month usage and cost
usage, err := client.ObjectStorage.GetUsage(ctx, &cubepath.ObjectStorageUsageOptions{Period: "2026-09"})

// Delete (force purges the bucket content first)
err = client.ObjectStorage.DeleteKey(ctx, key.UUID)
err = client.ObjectStorage.DeleteBucket(ctx, bucket.UUID, true)
```

Lifecycle rules delete objects in the background, permanently. `PutBucketLifecycle` replaces
every rule; the change is applied asynchronously (seconds, up to about 12 minutes after a previous
change of the same bucket) and objects go within 48 hours of their due date. In a versioned
bucket an expiration only adds a delete marker: add a noncurrent version rule to free space.

```go
days, prefix := 30, "logs/"
change, err := client.ObjectStorage.PutBucketLifecycle(ctx, bucket.UUID, []cubepath.ObjectStorageLifecycleRule{{
    ID:         "logs-30d",
    Enabled:    true,
    Filter:     &cubepath.ObjectStorageLifecycleFilter{Prefix: &prefix},
    Expiration: &cubepath.ObjectStorageLifecycleExpiration{Days: &days},
}})
lifecycle, err := client.ObjectStorage.GetBucketLifecycle(ctx, bucket.UUID) // poll until lifecycle.Applied()
_, err = client.ObjectStorage.DeleteBucketLifecycle(ctx, bucket.UUID)
```

Serve a bucket publicly through the CDN by adding it as an origin of a CDN zone:

```go
origin, err := client.CDN.CreateOrigin(ctx, zone.UUID, &cubepath.CreateCDNOriginRequest{
    Name:                    "assets",
    ObjectStorageBucketUUID: bucket.UUID,
})
```

Deleting that origin stops serving the bucket.

### Managed Databases

Managed MySQL, PostgreSQL and Valkey. Operations run in the background: poll `Get` until the
status is `active` again (the API answers 409 while another operation is running).

```go
plans, err := client.ManagedDatabases.ListPlans(ctx, "postgresql") // per location, price per node

created, err := client.ManagedDatabases.Create(ctx, &cubepath.CreateManagedDatabaseRequest{
    ProjectID: projectID,
    Name:      "app-db",
    Engine:    "postgresql",
    Version:   "17.5.0",
    PlanUUID:  plans[0].Plans[0].UUID,
    Replicas:  2,
})
db, err := client.ManagedDatabases.Get(ctx, created.UUID) // poll until Status == "active"

creds, err := client.ManagedDatabases.GetCredentials(ctx, created.UUID)
fmt.Println(creds.URI)

// Logical databases and users (the user password is only returned here)
_, err = client.ManagedDatabases.CreateDatabase(ctx, created.UUID, "app")
user, err := client.ManagedDatabases.CreateUser(ctx, created.UUID, &cubepath.CreateManagedDatabaseUserRequest{Username: "app"})

// Scaling, configuration and credential rotation
replicas := 3
_, err = client.ManagedDatabases.Scale(ctx, created.UUID, &cubepath.ScaleManagedDatabaseRequest{Replicas: &replicas})
cfg, err := client.ManagedDatabases.GetConfig(ctx, created.UUID)
_, err = client.ManagedDatabases.UpdateConfig(ctx, created.UUID, map[string]interface{}{"work_mem": 8192})
err = client.ManagedDatabases.RotateCredentials(ctx, created.UUID)

err = client.ManagedDatabases.Delete(ctx, created.UUID)
```

### DDoS Mitigation

Protection profiles, country/ASN/prefix list filters and traffic capture apply to IPs with
Premium protection; firewall rules on the scrubbing platform work on any IP of the organization.

```go
ips, err := client.DDoSMitigation.ListIPs(ctx, nil)

// Firewall rules (a subnet creates one rule per IP)
err = client.DDoSMitigation.CreateFirewallRule(ctx, &cubepath.CreateDDoSFirewallRuleRequest{
    Network:  "203.0.113.10",
    Protocol: cubepath.DDoSProtocolUDP,
    DstPort:  27015,
    Action:   cubepath.DDoSActionDrop,
})
rules, err := client.DDoSMitigation.ListFirewallRules(ctx, "203.0.113.10")
err = client.DDoSMitigation.DeleteFirewallRule(ctx, rules[0].ID)

// Protection profile: start from the current one and change what you need
profile := cubepath.DefaultDDoSProtectionProfile()
profile.CountryMode = 1 // block the listed countries
err = client.DDoSMitigation.UpsertProfile(ctx, "203.0.113.10", profile)
err = client.DDoSMitigation.SetProfileCountries(ctx, "203.0.113.10", []string{"BR"})

// Prefix lists
list, err := client.DDoSMitigation.CreatePrefixList(ctx, "partners", "")
err = client.DDoSMitigation.AddPrefixListEntry(ctx, list.UUID, "198.51.100.0/24")

// Sampled traffic and pass/drop statistics
stats, err := client.DDoSMitigation.GetTrafficStats(ctx, &cubepath.DDoSTrafficStatsRequest{
    StartTime: time.Now().Add(-time.Hour),
    EndTime:   time.Now(),
    Interval:  "5m",
})
```

### Cloud Alerts

Alerts watch a metric of a VPS, baremetal server or availability group and notify a channel
(or create/destroy a VPS) when a threshold is crossed.

```go
channel, err := client.CloudAlerts.CreateNotificationChannel(ctx, &cubepath.CreateNotificationChannelRequest{
    Name:   "ops",
    Type:   cubepath.NotificationChannelSlack,
    Config: map[string]interface{}{"webhook_url": "https://hooks.slack.com/services/..."},
})

alert, err := client.CloudAlerts.Create(ctx, &cubepath.CreateCloudAlertRequest{
    ProjectID:  projectID,
    Name:       "high cpu",
    TargetType: cubepath.CloudAlertTargetVPS,
    TargetID:   "12345",
    MetricType: cubepath.CloudAlertMetricCPU,
    Operator:   cubepath.CloudAlertOperatorGreaterThan,
    Threshold:  90,
    Actions: []cubepath.CloudAlertActionRequest{
        {ActionType: cubepath.CloudAlertActionNotify, NotificationChannelID: channel.ID},
    },
})

disabled := "disabled"
_, err = client.CloudAlerts.Update(ctx, alert.ID, &cubepath.UpdateCloudAlertRequest{Status: &disabled})
events, err := client.CloudAlerts.History(ctx, alert.ID, 50)
```

### Video Transcoder

Jobs read a video from a URL or an S3 compatible bucket and write the outputs to your S3
compatible bucket (for example a CubePath Object Storage bucket).

```go
job, err := client.Transcoder.CreateJob(ctx, &cubepath.CreateTranscodeJobRequest{
    Input: cubepath.TranscodeInput{Source: "url", URL: "https://example.com/video.mp4"},
    Output: cubepath.TranscodeDestination{S3: cubepath.TranscodeS3{
        Endpoint:  key.Endpoint,
        Region:    key.Region,
        Bucket:    "videos",
        Path:      "encoded/",
        AccessKey: key.AccessKeyID,
        SecretKey: key.SecretAccessKey,
    }},
    Outputs: []cubepath.TranscodeOutputSpec{
        {Type: "file", Params: map[string]interface{}{"codec": "h264", "height": 720, "container": "mp4"}},
        {Type: "hls"},
    },
})
job, err = client.Transcoder.GetJob(ctx, job.UUID) // poll until completed, failed or canceled
outputs, err := client.Transcoder.GetJobOutputs(ctx, job.UUID)
page, err := client.Transcoder.ListJobs(ctx, &cubepath.TranscodeJobListOptions{Limit: 100})
```

### Other Services

#### Projects

```go
project, err := client.Projects.Create(ctx, &cubepath.CreateProjectRequest{
    Name:        "production",
    Description: "Production environment",
})
projects, err := client.Projects.List(ctx)
err = client.Projects.Update(ctx, project.ID, "prod")
```

#### SSH Keys

```go
key, err := client.SSHKeys.Create(ctx, &cubepath.CreateSSHKeyRequest{
    Name:   "deploy-key",
    SSHKey: "ssh-ed25519 AAAA...",
})
keys, err := client.SSHKeys.List(ctx)
key, err = client.SSHKeys.Update(ctx, key.ID, "deploy-key-2") // rename
```

#### Pricing

```go
pricing, err := client.Pricing.Get(ctx)
```

#### DDoS Attacks

```go
attacks, err := client.DDoS.ListAttacks(ctx) // empty when there are none
details, err := client.DDoS.GetAttackDetails(ctx, attacks[0].AttackID)
graph, err := client.DDoS.GetAttackTrafficGraph(ctx, attacks[0].AttackID)
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
