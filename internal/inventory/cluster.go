package inventory

import (
	"fmt"
	"strings"
	"time"
)

const minInstances int = 3

// Cluster is the object to store Openshift Clusters and its properties
type Cluster struct {
	// ID is the uniq key to idenfity every cluster independently of which account it belongs
	// Its built as "name+infra_id+account"
	ID string `db:"id" json:"id"`

	// Cluster's Name. Must be unique per Account
	Name string `db:"name" json:"name"`

	// InfraID is the infrastructure ID generated by openshift-installer during the installation.Cluster (Could be undefined)
	InfraID string `db:"infra_id" json:"infra_id"`

	// Infrastructure provider identifier.
	Provider CloudProvider `db:"provider" json:"provider"`

	// Defines the status of the cluster if its infrastructure is running or not
	Status InstanceState `db:"state" json:"status"`

	// The region of the infrastructure provider in which the cluster is deployed
	Region string `db:"region" json:"region"`

	// Account name which this cluster belongs to
	AccountName string `db:"account_name" json:"accountName"`

	// Openshift Console URL. Might not be accesible if its protected
	ConsoleLink string `db:"console_link" json:"consoleLink"`

	// Instances count
	InstanceCount int `db:"instance_count" json:"instanceCount"`

	// Last scan timestamp of the account
	LastScanTimestamp time.Time `db:"last_scan_timestamp" json:"lastScanTimestamp"`

	// Cluster's instance (nodes) lists
	Instances []Instance
}

// NewCluster creates a new cluster instance
func NewCluster(name string, infraID string, provider CloudProvider, region string, accountName string, consoleLink string) *Cluster {
	id, err := GenerateClusterID(name, infraID, accountName)
	if err != nil {
		fmt.Println("Can't generate Cluster Object: ", err.Error())
		return nil
	}
	return &Cluster{
		ID:                id,
		Name:              name,
		InfraID:           infraID,
		Provider:          provider,
		Status:            Unknown,
		Region:            region,
		AccountName:       accountName,
		ConsoleLink:       consoleLink,
		InstanceCount:     0,
		LastScanTimestamp: time.Now(),
		Instances:         make([]Instance, 0),
	}
}

// isClusterStopped checks if the Cluster is Stopped
func (c Cluster) isClusterStopped() bool {
	if c.Status == Stopped {
		return true
	}
	return false
}

// isClusterRunning checks if the Cluster is Running
func (c Cluster) isClusterRunning() bool {
	if c.Status == Running {
		return true
	}
	return false
}

// UpdateStatus evaluate the status of the cluster checking how many of the
// nodes are in Running or Stopped status. As Openshift needs at lease 3 nodes
// running to be considered correctly Running (3 master nodes), but we cant'
// figure out which Instance is a master node, if at least 3 of the Cluster
// instances are running, Cluster will be considered as Running also.
// If the instances count is less than minInstances, Cluster would be
// considered as Unknown status
// TODO: find out a more trustable approach to evaluate cluster status
func (c *Cluster) UpdateStatus() {
	c.InstanceCount = len(c.Instances)

	// Check minimun instances
	if c.InstanceCount < minInstances {
		c.Status = Unknown
		return
	}

	count := 0
	for _, instance := range c.Instances {
		if instance.State == Running {
			count++
		}
		if count >= minInstances {
			c.Status = Running
			return
		}
	}

	c.Status = Stopped
}

// AddInstance add a new instance to a cluster
func (c *Cluster) AddInstance(instance Instance) {
	c.Instances = append(c.Instances, instance)
	c.UpdateStatus()
}

// PrintCluster prints cluster info
func (c Cluster) PrintCluster() {
	fmt.Printf("\tCluster: %s[%s] -- [%s](Instances: %d)\n", c.Name, c.ID, c.ConsoleLink, c.InstanceCount)
	for _, instance := range c.Instances {
		instance.PrintInstance()
	}
	fmt.Printf("\n")
}

// Obtain the required parameters for generate a ClusterID. If any key parameter is missing, it will return a non-nil error
func GenerateClusterID(name string, infraID string, accountName string) (string, error) {
	if name == "" || accountName == "" {
		return "", fmt.Errorf("Can't generate ClusterID. Some key parameters are missing")
	}
	args := []string{name, infraID, accountName}
	id := strings.Join(args, "-")
	return id, nil
}
