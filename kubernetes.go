package cubepath

import (
	"context"
	"encoding/json"
	"fmt"
)

// KubernetesService handles communication with the Kubernetes related methods of the CubePath API.
type KubernetesService interface {
	// Cluster management
	ListVersions(ctx context.Context) ([]KubernetesVersion, error)
	ListPlans(ctx context.Context, version string) ([]KubernetesPlan, error)
	List(ctx context.Context) ([]KubernetesCluster, error)
	Get(ctx context.Context, clusterUUID string) (*KubernetesCluster, error)
	Create(ctx context.Context, req *CreateKubernetesClusterRequest) (*KubernetesClusterResponse, error)
	Update(ctx context.Context, clusterUUID string, req *UpdateKubernetesClusterRequest) error
	Delete(ctx context.Context, clusterUUID string) error
	GetKubeconfig(ctx context.Context, clusterUUID string) (string, error)
	Move(ctx context.Context, clusterUUID string, projectID int) error
	ListLoadBalancers(ctx context.Context, clusterUUID string) ([]KubernetesLB, error)

	// Node Pools
	ListNodePools(ctx context.Context, clusterUUID string) ([]NodePool, error)
	CreateNodePool(ctx context.Context, clusterUUID string, req *CreateNodePoolRequest) (*NodePoolResponse, error)
	UpdateNodePool(ctx context.Context, clusterUUID, poolUUID string, req *UpdateNodePoolRequest) error
	DeleteNodePool(ctx context.Context, clusterUUID, poolUUID string) error
	AddNodes(ctx context.Context, clusterUUID, poolUUID string, count int) error
	RemoveNode(ctx context.Context, clusterUUID, poolUUID, vpsID string) error

	// Addons
	ListAvailableAddons(ctx context.Context) ([]KubernetesAddon, error)
	GetAddon(ctx context.Context, slug string) (*KubernetesAddon, error)
	ListInstalledAddons(ctx context.Context, clusterUUID string) ([]InstalledAddon, error)
	InstallAddon(ctx context.Context, clusterUUID, slug string, req *InstallAddonRequest) error
	UninstallAddon(ctx context.Context, clusterUUID, addonUUID string) error
}

// KubernetesVersion represents an available Kubernetes version.
type KubernetesVersion struct {
	Version   string `json:"version"`
	IsDefault bool   `json:"is_default"`
	MinCPU    int    `json:"min_cpu"`
	MinRAMMB  int    `json:"min_ram_mb"`
}

// KubernetesPlan represents a server plan compatible with Kubernetes.
type KubernetesPlan struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	CPU          int     `json:"cpu"`
	RAM          int     `json:"ram"`
	Storage      int     `json:"storage"`
	PricePerHour float64 `json:"price_per_hour"`
}

// KubernetesCluster represents a Kubernetes cluster.
type KubernetesCluster struct {
	UUID           string             `json:"uuid"`
	Name           string             `json:"name"`
	Label          string             `json:"label"`
	Status         string             `json:"status"`
	Version        string             `json:"version"`
	HAControlPlane bool               `json:"ha_control_plane"`
	APIEndpoint    string             `json:"api_endpoint"`
	PodCIDR        string             `json:"pod_cidr"`
	ServiceCIDR    string             `json:"service_cidr"`
	BillingType    string             `json:"billing_type"`
	Location       KubernetesLocation `json:"location"`
	Network        *KubernetesNetwork `json:"network,omitempty"`
	NodePools      []NodePool         `json:"node_pools"`
	WorkerCount    int                `json:"worker_count"`
	NodePoolCount  int                `json:"node_pool_count"`
	CreatedAt      string             `json:"created_at"`
}

// KubernetesLocation represents a cluster's location.
type KubernetesLocation struct {
	LocationName string `json:"location_name"`
	Description  string `json:"description"`
}

// KubernetesNetwork represents the network configuration of a cluster.
type KubernetesNetwork struct {
	Name    string `json:"name"`
	IPRange string `json:"ip_range"`
	Prefix  int    `json:"prefix"`
}

// NodePool represents a Kubernetes node pool.
type NodePool struct {
	UUID         string       `json:"uuid"`
	Name         string       `json:"name"`
	DesiredNodes int          `json:"desired_nodes"`
	MinNodes     int          `json:"min_nodes"`
	MaxNodes     int          `json:"max_nodes"`
	AutoScale    bool         `json:"auto_scale"`
	Plan         NodePoolPlan `json:"plan"`
	Nodes        []Node       `json:"nodes"`
}

// NodePoolPlan represents the server plan for a node pool.
type NodePoolPlan struct {
	Name string `json:"name"`
}

// Node represents a worker node in a node pool.
type Node struct {
	VPSName    string `json:"vps_name"`
	VPSStatus  string `json:"vps_status"`
	K8sStatus  string `json:"k8s_status"`
	FloatingIP string `json:"floating_ip"`
	PrivateIP  string `json:"private_ip"`
}

// KubernetesAddon represents an available Kubernetes addon.
type KubernetesAddon struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	HelmRepoName     string `json:"helm_repo_name"`
	HelmRepoURL      string `json:"helm_repo_url"`
	HelmChart        string `json:"helm_chart"`
	DefaultVersion   string `json:"default_version"`
	Namespace        string `json:"namespace"`
	IconURL          string `json:"icon_url"`
	DocumentationURL string `json:"documentation_url"`
	Keywords         string `json:"keywords"`
	MinK8sVersion    string `json:"min_k8s_version"`
}

// InstalledAddon represents an addon installed on a cluster.
type InstalledAddon struct {
	UUID             string `json:"uuid"`
	Status           string `json:"status"`
	InstalledVersion string `json:"installed_version"`
	Addon            struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"addon"`
	InstalledAt string `json:"installed_at"`
}

// KubernetesLB represents a load balancer associated with a cluster.
type KubernetesLB struct {
	UUID   string `json:"uuid"`
	Name   string `json:"name"`
	Status string `json:"status"`
	IP     string `json:"floating_ip_address"`
}

// KubernetesClusterResponse represents the response from creating a cluster.
type KubernetesClusterResponse struct {
	Detail string `json:"detail,omitempty"`
	UUID   string `json:"uuid,omitempty"`
}

// NodePoolResponse represents the response from creating a node pool.
type NodePoolResponse struct {
	Detail string `json:"detail,omitempty"`
	UUID   string `json:"uuid,omitempty"`
}

// CreateKubernetesClusterRequest represents a request to create a Kubernetes cluster.
type CreateKubernetesClusterRequest struct {
	ProjectID      int                    `json:"project_id"`
	Name           string                 `json:"name"`
	LocationName   string                 `json:"location_name"`
	Version        string                 `json:"version,omitempty"`
	HAControlPlane bool                   `json:"ha_control_plane"`
	NodePools      []CreateNodePoolConfig `json:"node_pools"`
	Network        *ClusterNetworkConfig  `json:"network,omitempty"`
}

// CreateNodePoolConfig represents a node pool configuration for cluster creation.
type CreateNodePoolConfig struct {
	Name  string `json:"name"`
	Plan  string `json:"plan"`
	Count int    `json:"count"`
}

// ClusterNetworkConfig represents network configuration for cluster creation.
type ClusterNetworkConfig struct {
	NetworkID   int    `json:"network_id,omitempty"`
	NodeCIDR    string `json:"node_cidr,omitempty"`
	PodCIDR     string `json:"pod_cidr,omitempty"`
	ServiceCIDR string `json:"service_cidr,omitempty"`
}

// UpdateKubernetesClusterRequest represents a request to update a cluster.
type UpdateKubernetesClusterRequest struct {
	Name  *string `json:"name,omitempty"`
	Label *string `json:"label,omitempty"`
}

// CreateNodePoolRequest represents a request to create a node pool.
type CreateNodePoolRequest struct {
	Name      string            `json:"name"`
	Plan      string            `json:"plan"`
	Count     int               `json:"count"`
	AutoScale bool              `json:"auto_scale"`
	Labels    map[string]string `json:"labels,omitempty"`
	Taints    []NodeTaint       `json:"taints,omitempty"`
}

// NodeTaint represents a Kubernetes node taint.
type NodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// UpdateNodePoolRequest represents a request to update a node pool.
type UpdateNodePoolRequest struct {
	Name         *string           `json:"name,omitempty"`
	DesiredNodes *int              `json:"desired_nodes,omitempty"`
	MinNodes     *int              `json:"min_nodes,omitempty"`
	MaxNodes     *int              `json:"max_nodes,omitempty"`
	AutoScale    *bool             `json:"auto_scale,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Taints       []NodeTaint       `json:"taints,omitempty"`
}

// InstallAddonRequest represents a request to install an addon.
type InstallAddonRequest struct {
	CustomValues map[string]interface{} `json:"custom_values,omitempty"`
}

type kubernetesService struct {
	client *Client
}

// Cluster management

func (s *kubernetesService) ListVersions(ctx context.Context) ([]KubernetesVersion, error) {
	var versions []KubernetesVersion
	if err := s.client.get(ctx, "/kubernetes/versions", &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (s *kubernetesService) ListPlans(ctx context.Context, version string) ([]KubernetesPlan, error) {
	path := "/kubernetes/plans"
	if version != "" {
		path += "?version=" + version
	}
	var plans []KubernetesPlan
	if err := s.client.get(ctx, path, &plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *kubernetesService) List(ctx context.Context) ([]KubernetesCluster, error) {
	var clusters []KubernetesCluster
	if err := s.client.get(ctx, "/kubernetes/", &clusters); err != nil {
		return nil, err
	}
	return clusters, nil
}

func (s *kubernetesService) Get(ctx context.Context, clusterUUID string) (*KubernetesCluster, error) {
	var cluster KubernetesCluster
	if err := s.client.get(ctx, fmt.Sprintf("/kubernetes/%s", clusterUUID), &cluster); err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (s *kubernetesService) Create(ctx context.Context, req *CreateKubernetesClusterRequest) (*KubernetesClusterResponse, error) {
	var result KubernetesClusterResponse
	if err := s.client.post(ctx, "/kubernetes/", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *kubernetesService) Update(ctx context.Context, clusterUUID string, req *UpdateKubernetesClusterRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/kubernetes/%s", clusterUUID), req, nil)
}

func (s *kubernetesService) Delete(ctx context.Context, clusterUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/kubernetes/%s", clusterUUID))
}

func (s *kubernetesService) GetKubeconfig(ctx context.Context, clusterUUID string) (string, error) {
	data, err := s.client.getRaw(ctx, fmt.Sprintf("/kubernetes/%s/kubeconfig", clusterUUID))
	if err != nil {
		return "", err
	}

	// Try to extract kubeconfig from JSON wrapper
	var result struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	if err := json.Unmarshal(data, &result); err == nil && result.Kubeconfig != "" {
		return result.Kubeconfig, nil
	}

	// Return raw response as string
	return string(data), nil
}

func (s *kubernetesService) Move(ctx context.Context, clusterUUID string, projectID int) error {
	body := map[string]interface{}{
		"project_id": projectID,
	}
	return s.client.post(ctx, fmt.Sprintf("/kubernetes/%s/move", clusterUUID), body, nil)
}

func (s *kubernetesService) ListLoadBalancers(ctx context.Context, clusterUUID string) ([]KubernetesLB, error) {
	var lbs []KubernetesLB
	if err := s.client.get(ctx, fmt.Sprintf("/kubernetes/%s/loadbalancers", clusterUUID), &lbs); err != nil {
		return nil, err
	}
	return lbs, nil
}

// Node Pools

func (s *kubernetesService) ListNodePools(ctx context.Context, clusterUUID string) ([]NodePool, error) {
	var pools []NodePool
	if err := s.client.get(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/", clusterUUID), &pools); err != nil {
		return nil, err
	}
	return pools, nil
}

func (s *kubernetesService) CreateNodePool(ctx context.Context, clusterUUID string, req *CreateNodePoolRequest) (*NodePoolResponse, error) {
	var result NodePoolResponse
	if err := s.client.post(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/", clusterUUID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *kubernetesService) UpdateNodePool(ctx context.Context, clusterUUID, poolUUID string, req *UpdateNodePoolRequest) error {
	return s.client.patch(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/%s", clusterUUID, poolUUID), req, nil)
}

func (s *kubernetesService) DeleteNodePool(ctx context.Context, clusterUUID, poolUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/%s", clusterUUID, poolUUID))
}

func (s *kubernetesService) AddNodes(ctx context.Context, clusterUUID, poolUUID string, count int) error {
	body := map[string]interface{}{
		"count": count,
	}
	return s.client.post(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/%s/nodes", clusterUUID, poolUUID), body, nil)
}

func (s *kubernetesService) RemoveNode(ctx context.Context, clusterUUID, poolUUID, vpsID string) error {
	return s.client.del(ctx, fmt.Sprintf("/kubernetes/%s/node-pools/%s/nodes/%s", clusterUUID, poolUUID, vpsID))
}

// Addons

func (s *kubernetesService) ListAvailableAddons(ctx context.Context) ([]KubernetesAddon, error) {
	var addons []KubernetesAddon
	if err := s.client.get(ctx, "/kubernetes/addons", &addons); err != nil {
		return nil, err
	}
	return addons, nil
}

func (s *kubernetesService) GetAddon(ctx context.Context, slug string) (*KubernetesAddon, error) {
	var addon KubernetesAddon
	if err := s.client.get(ctx, fmt.Sprintf("/kubernetes/addons/%s", slug), &addon); err != nil {
		return nil, err
	}
	return &addon, nil
}

func (s *kubernetesService) ListInstalledAddons(ctx context.Context, clusterUUID string) ([]InstalledAddon, error) {
	var installed []InstalledAddon
	if err := s.client.get(ctx, fmt.Sprintf("/kubernetes/%s/addons", clusterUUID), &installed); err != nil {
		return nil, err
	}
	return installed, nil
}

func (s *kubernetesService) InstallAddon(ctx context.Context, clusterUUID, slug string, req *InstallAddonRequest) error {
	body := map[string]interface{}{}
	if req != nil && len(req.CustomValues) > 0 {
		body["custom_values"] = req.CustomValues
	}
	return s.client.post(ctx, fmt.Sprintf("/kubernetes/%s/addons/%s/install", clusterUUID, slug), body, nil)
}

func (s *kubernetesService) UninstallAddon(ctx context.Context, clusterUUID, addonUUID string) error {
	return s.client.del(ctx, fmt.Sprintf("/kubernetes/%s/addons/%s", clusterUUID, addonUUID))
}
