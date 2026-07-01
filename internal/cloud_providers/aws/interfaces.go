package cloudprovider

import (
	"context"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
)

// EC2Client defines the interface for EC2 operations.
type EC2Client interface {
	GetRegion() string
	FilterExistingInstances(ctx context.Context, instanceIDs []string) ([]string, error)
	StopClusterInstances(ctx context.Context, instanceIDs []string) error
	StartClusterInstances(ctx context.Context, instanceIDs []string) error
	GetRegionsList(ctx context.Context) ([]string, error)
	GetInstances(ctx context.Context) ([]inventory.Instance, error)
}

// STSClient defines the interface for STS operations.
type STSClient interface {
	GetAWSAccountID(ctx context.Context) string
}

// Route53Client defines the interface for Route53 operations.
type Route53Client interface {
	GetZonesWithTags(ctx context.Context) ([]DNSHostedZone, error)
	ZoneBelongsToCluster(cluster *inventory.Cluster, zoneWithTags DNSHostedZone) bool
	GetHostedZoneRecords(ctx context.Context, hostedZoneID string) ([]DNSRecordSet, error)
}

// CostExplorerClient defines the interface for CostExplorer operations.
type CostExplorerClient interface {
	GetCostAndUsageWithResources(ctx context.Context, input *CostAndUsageInput) (*CostAndUsageOutput, error)
}

// DNSHostedZone represents a DNS hosted zone with its tags, independent of any SDK.
type DNSHostedZone struct {
	ID   string
	Name string
	Tags []DNSHostedZoneTag
}

// DNSHostedZoneTag represents a tag on a hosted zone.
type DNSHostedZoneTag struct {
	Key   string
	Value string
}

// DNSRecordSet represents a DNS record, independent of any SDK.
type DNSRecordSet struct {
	Name string
}

// CostAndUsageInput wraps the parameters needed for a cost query.
type CostAndUsageInput struct {
	StartDate  string
	EndDate    string
	InstanceID string
}

// CostAndUsageOutput wraps the results of a cost query.
type CostAndUsageOutput struct {
	Results []CostResult
}

// CostResult represents a single time-period cost entry.
type CostResult struct {
	StartDate string
	Amount    string
}
