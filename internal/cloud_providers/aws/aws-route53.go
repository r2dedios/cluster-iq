package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type route53API interface {
	ListHostedZonesByName(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error)
	ListTagsForResource(ctx context.Context, input *route53.ListTagsForResourceInput, opts ...func(*route53.Options)) (*route53.ListTagsForResourceOutput, error)
	ListResourceRecordSets(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
}

// AWSRoute53Connection wraps the Route53 client for DNS operations.
type AWSRoute53Connection struct {
	client route53API
}

// NewAWSRoute53Connection creates a new Route53 client from a v2 AWS config.
func NewAWSRoute53Connection(cfg aws.Config) *AWSRoute53Connection {
	return &AWSRoute53Connection{
		client: route53.NewFromConfig(cfg),
	}
}

// WithRoute53 configures an AWSConnection instance for including the Route53 client
func WithRoute53() AWSConnectionOption {
	return func(conn *AWSConnection) {
		conn.Route53 = NewAWSRoute53Connection(conn.awsCfg)
	}
}

// GetZonesWithTags retrieves all hosted zones and their associated tags from AWS Route53
func (c *AWSRoute53Connection) GetZonesWithTags(ctx context.Context) ([]DNSHostedZone, error) {
	result, err := c.client.ListHostedZonesByName(ctx, &route53.ListHostedZonesByNameInput{})
	if err != nil {
		return nil, err
	}

	zonesWithTags := make([]DNSHostedZone, 0, len(result.HostedZones))

	for _, zone := range result.HostedZones {
		tags, err := c.client.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
			ResourceType: r53types.TagResourceTypeHostedzone,
			ResourceId:   zone.Id,
		})
		if err != nil || tags.ResourceTagSet == nil {
			continue
		}

		dnsZone := DNSHostedZone{
			ID:   aws.ToString(zone.Id),
			Name: aws.ToString(zone.Name),
			Tags: make([]DNSHostedZoneTag, 0, len(tags.ResourceTagSet.Tags)),
		}
		for _, tag := range tags.ResourceTagSet.Tags {
			dnsZone.Tags = append(dnsZone.Tags, DNSHostedZoneTag{
				Key:   aws.ToString(tag.Key),
				Value: aws.ToString(tag.Value),
			})
		}

		zonesWithTags = append(zonesWithTags, dnsZone)
	}

	return zonesWithTags, nil
}

// ZoneBelongsToCluster returns true if the hosted zone is associated to a cluster Ingress (routers)
func (c *AWSRoute53Connection) ZoneBelongsToCluster(cluster *inventory.Cluster, zoneWithTags DNSHostedZone) bool {
	for _, tag := range zoneWithTags.Tags {
		if strings.Contains(tag.Key, cluster.ClusterName) {
			return true
		}
	}
	return false
}

// GetHostedZoneRecords returns every record of a given HostedZone
func (c *AWSRoute53Connection) GetHostedZoneRecords(ctx context.Context, hostedZoneID string) ([]DNSRecordSet, error) {
	paginator := route53.NewListResourceRecordSetsPaginator(c.client, &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
	})

	var records []DNSRecordSet
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting the DNS registries: %w", err)
		}
		for _, rr := range page.ResourceRecordSets {
			records = append(records, DNSRecordSet{
				Name: aws.ToString(rr.Name),
			})
		}
	}

	return records, nil
}
