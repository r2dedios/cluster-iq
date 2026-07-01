package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/assert"
)

type mockRoute53API struct {
	listHostedZonesByNameFn func(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error)
	listTagsForResourceFn   func(ctx context.Context, input *route53.ListTagsForResourceInput, opts ...func(*route53.Options)) (*route53.ListTagsForResourceOutput, error)
	listResourceRecordSetsFn func(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
}

func (m *mockRoute53API) ListHostedZonesByName(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
	return m.listHostedZonesByNameFn(ctx, input, opts...)
}

func (m *mockRoute53API) ListTagsForResource(ctx context.Context, input *route53.ListTagsForResourceInput, opts ...func(*route53.Options)) (*route53.ListTagsForResourceOutput, error) {
	return m.listTagsForResourceFn(ctx, input, opts...)
}

func (m *mockRoute53API) ListResourceRecordSets(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	return m.listResourceRecordSetsFn(ctx, input, opts...)
}

func TestAWSRoute53Connection_GetZonesWithTags(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockRoute53API{
			listHostedZonesByNameFn: func(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
				return &route53.ListHostedZonesByNameOutput{
					HostedZones: []r53types.HostedZone{
						{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
					},
				}, nil
			},
			listTagsForResourceFn: func(ctx context.Context, input *route53.ListTagsForResourceInput, opts ...func(*route53.Options)) (*route53.ListTagsForResourceOutput, error) {
				return &route53.ListTagsForResourceOutput{
					ResourceTagSet: &r53types.ResourceTagSet{
						Tags: []r53types.Tag{
							{Key: aws.String("kubernetes.io/cluster/my-cluster-abc123"), Value: aws.String("owned")},
						},
					},
				}, nil
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		zones, err := conn.GetZonesWithTags(context.Background())
		assert.NoError(t, err)
		assert.Len(t, zones, 1)
		assert.Equal(t, "/hostedzone/Z1", zones[0].ID)
		assert.Equal(t, "example.com.", zones[0].Name)
		assert.Len(t, zones[0].Tags, 1)
		assert.Equal(t, "kubernetes.io/cluster/my-cluster-abc123", zones[0].Tags[0].Key)
		assert.Equal(t, "owned", zones[0].Tags[0].Value)
	})

	t.Run("empty zones", func(t *testing.T) {
		mock := &mockRoute53API{
			listHostedZonesByNameFn: func(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
				return &route53.ListHostedZonesByNameOutput{
					HostedZones: []r53types.HostedZone{},
				}, nil
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		zones, err := conn.GetZonesWithTags(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, zones)
	})

	t.Run("tag error skips zone", func(t *testing.T) {
		mock := &mockRoute53API{
			listHostedZonesByNameFn: func(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
				return &route53.ListHostedZonesByNameOutput{
					HostedZones: []r53types.HostedZone{
						{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
						{Id: aws.String("/hostedzone/Z2"), Name: aws.String("other.com.")},
					},
				}, nil
			},
			listTagsForResourceFn: func(ctx context.Context, input *route53.ListTagsForResourceInput, opts ...func(*route53.Options)) (*route53.ListTagsForResourceOutput, error) {
				if aws.ToString(input.ResourceId) == "/hostedzone/Z1" {
					return nil, fmt.Errorf("access denied")
				}
				return &route53.ListTagsForResourceOutput{
					ResourceTagSet: &r53types.ResourceTagSet{
						Tags: []r53types.Tag{
							{Key: aws.String("env"), Value: aws.String("prod")},
						},
					},
				}, nil
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		zones, err := conn.GetZonesWithTags(context.Background())
		assert.NoError(t, err)
		assert.Len(t, zones, 1)
		assert.Equal(t, "other.com.", zones[0].Name)
	})

	t.Run("list zones error", func(t *testing.T) {
		mock := &mockRoute53API{
			listHostedZonesByNameFn: func(ctx context.Context, input *route53.ListHostedZonesByNameInput, opts ...func(*route53.Options)) (*route53.ListHostedZonesByNameOutput, error) {
				return nil, fmt.Errorf("network error")
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		zones, err := conn.GetZonesWithTags(context.Background())
		assert.Error(t, err)
		assert.Nil(t, zones)
	})
}

func TestAWSRoute53Connection_ZoneBelongsToCluster(t *testing.T) {
	conn := &AWSRoute53Connection{}

	t.Run("match", func(t *testing.T) {
		cluster := &inventory.Cluster{ClusterName: "my-cluster"}
		zone := DNSHostedZone{
			Tags: []DNSHostedZoneTag{
				{Key: "kubernetes.io/cluster/my-cluster-abc123", Value: "owned"},
			},
		}
		assert.True(t, conn.ZoneBelongsToCluster(cluster, zone))
	})

	t.Run("no match", func(t *testing.T) {
		cluster := &inventory.Cluster{ClusterName: "my-cluster"}
		zone := DNSHostedZone{
			Tags: []DNSHostedZoneTag{
				{Key: "kubernetes.io/cluster/other-cluster-xyz", Value: "owned"},
			},
		}
		assert.False(t, conn.ZoneBelongsToCluster(cluster, zone))
	})

	t.Run("empty tags", func(t *testing.T) {
		cluster := &inventory.Cluster{ClusterName: "my-cluster"}
		zone := DNSHostedZone{Tags: []DNSHostedZoneTag{}}
		assert.False(t, conn.ZoneBelongsToCluster(cluster, zone))
	})
}

func TestAWSRoute53Connection_GetHostedZoneRecords(t *testing.T) {
	t.Run("success single page", func(t *testing.T) {
		mock := &mockRoute53API{
			listResourceRecordSetsFn: func(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
				return &route53.ListResourceRecordSetsOutput{
					ResourceRecordSets: []r53types.ResourceRecordSet{
						{Name: aws.String("console-openshift-console.apps.my-cluster.example.com.")},
						{Name: aws.String("api.my-cluster.example.com.")},
					},
					IsTruncated: false,
				}, nil
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		records, err := conn.GetHostedZoneRecords(context.Background(), "/hostedzone/Z1")
		assert.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "console-openshift-console.apps.my-cluster.example.com.", records[0].Name)
		assert.Equal(t, "api.my-cluster.example.com.", records[1].Name)
	})

	t.Run("paginated", func(t *testing.T) {
		callCount := 0
		mock := &mockRoute53API{
			listResourceRecordSetsFn: func(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
				callCount++
				if callCount == 1 {
					return &route53.ListResourceRecordSetsOutput{
						ResourceRecordSets: []r53types.ResourceRecordSet{
							{Name: aws.String("record1.example.com.")},
						},
						IsTruncated:    true,
						NextRecordName: aws.String("record2.example.com."),
					}, nil
				}
				return &route53.ListResourceRecordSetsOutput{
					ResourceRecordSets: []r53types.ResourceRecordSet{
						{Name: aws.String("record2.example.com.")},
					},
					IsTruncated: false,
				}, nil
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		records, err := conn.GetHostedZoneRecords(context.Background(), "/hostedzone/Z1")
		assert.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, 2, callCount)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockRoute53API{
			listResourceRecordSetsFn: func(ctx context.Context, input *route53.ListResourceRecordSetsInput, opts ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
				return nil, fmt.Errorf("throttled")
			},
		}
		conn := &AWSRoute53Connection{client: mock}
		records, err := conn.GetHostedZoneRecords(context.Background(), "/hostedzone/Z1")
		assert.Error(t, err)
		assert.Nil(t, records)
	})
}
