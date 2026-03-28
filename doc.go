// Package cubepath provides a Go client library for the CubePath cloud API.
//
// Usage:
//
//	client, err := cubepath.NewClient("your-api-token")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List projects
//	projects, err := client.Projects.List(context.Background())
//
//	// Create a VPS
//	task, err := client.VPS.Create(ctx, 1, &cubepath.CreateVPSRequest{
//	    Name:         "my-vps",
//	    PlanName:     "gp.nano",
//	    TemplateName: "debian-12",
//	    LocationName: "us-mia-1",
//	})
//
//	// Create a Kubernetes cluster
//	cluster, err := client.Kubernetes.Create(ctx, &cubepath.CreateKubernetesClusterRequest{
//	    ProjectID:    1,
//	    Name:         "my-cluster",
//	    LocationName: "us-mia-1",
//	    NodePools: []cubepath.CreateNodePoolConfig{
//	        {Name: "default", Plan: "gp.small", Count: 3},
//	    },
//	})
package cubepath
