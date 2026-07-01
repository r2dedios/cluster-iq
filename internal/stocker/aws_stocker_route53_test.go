package stocker

import (
	"context"
	"fmt"
	"testing"

	cpaws "github.com/RHEcosystemAppEng/cluster-iq/internal/cloud_providers/aws"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockRoute53Client struct {
	getZonesWithTagsFn      func(ctx context.Context) ([]cpaws.DNSHostedZone, error)
	zoneBelongsToClusterFn  func(cluster *inventory.Cluster, zone cpaws.DNSHostedZone) bool
	getHostedZoneRecordsFn  func(ctx context.Context, id string) ([]cpaws.DNSRecordSet, error)
}

func (m *mockRoute53Client) GetZonesWithTags(ctx context.Context) ([]cpaws.DNSHostedZone, error) {
	return m.getZonesWithTagsFn(ctx)
}

func (m *mockRoute53Client) ZoneBelongsToCluster(cluster *inventory.Cluster, zone cpaws.DNSHostedZone) bool {
	return m.zoneBelongsToClusterFn(cluster, zone)
}

func (m *mockRoute53Client) GetHostedZoneRecords(ctx context.Context, id string) ([]cpaws.DNSRecordSet, error) {
	return m.getHostedZoneRecordsFn(ctx, id)
}

func TestGenerateConsoleLink(t *testing.T) {
	t.Run("standard domain", func(t *testing.T) {
		link := generateConsoleLink("my-cluster.example.com.")
		assert.Equal(t, "https://console-openshift-console.apps.my-cluster.example.com.", link)
	})

	t.Run("empty domain", func(t *testing.T) {
		link := generateConsoleLink("")
		assert.Equal(t, "https://console-openshift-console.apps.", link)
	})
}

func TestSearchConsoleURLinDNSRecords(t *testing.T) {
	cluster := &inventory.Cluster{ClusterName: "my-cluster"}

	t.Run("found", func(t *testing.T) {
		records := []cpaws.DNSRecordSet{
			{Name: "api.other-cluster.example.com."},
			{Name: "console-openshift-console.apps.my-cluster.example.com."},
		}
		link := searchConsoleURLinDNSRecords(records, cluster)
		assert.Equal(t, "https://console-openshift-console.apps.console-openshift-console.apps.my-cluster.example.com.", link)
	})

	t.Run("not found", func(t *testing.T) {
		records := []cpaws.DNSRecordSet{
			{Name: "api.other-cluster.example.com."},
		}
		link := searchConsoleURLinDNSRecords(records, cluster)
		assert.Equal(t, unknownConsoleLinkCode, link)
	})

	t.Run("empty records", func(t *testing.T) {
		link := searchConsoleURLinDNSRecords([]cpaws.DNSRecordSet{}, cluster)
		assert.Equal(t, unknownConsoleLinkCode, link)
	})

	t.Run("nil records", func(t *testing.T) {
		link := searchConsoleURLinDNSRecords(nil, cluster)
		assert.Equal(t, unknownConsoleLinkCode, link)
	})

	t.Run("returns first match", func(t *testing.T) {
		records := []cpaws.DNSRecordSet{
			{Name: "api.my-cluster.example.com."},
			{Name: "console-openshift-console.apps.my-cluster.example.com."},
		}
		link := searchConsoleURLinDNSRecords(records, cluster)
		assert.Contains(t, link, "api.my-cluster.example.com.")
	})
}

func TestAWSStocker_getConsoleLinkOfCluster(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mockR53 := &mockRoute53Client{
			getHostedZoneRecordsFn: func(ctx context.Context, id string) ([]cpaws.DNSRecordSet, error) {
				return []cpaws.DNSRecordSet{
					{Name: "console-openshift-console.apps.my-cluster.example.com."},
				}, nil
			},
		}
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, mockR53, nil, "us-east-1")
		s := &AWSStocker{Account: account, conn: conn, logger: zap.NewNop()}

		cluster := &inventory.Cluster{ClusterName: "my-cluster"}
		zone := cpaws.DNSHostedZone{ID: "/hostedzone/Z1", Name: "example.com."}

		link := s.getConsoleLinkOfCluster(cluster, zone)
		assert.Contains(t, link, "my-cluster.example.com.")
		assert.Contains(t, link, "https://")
	})

	t.Run("api error returns unknown", func(t *testing.T) {
		mockR53 := &mockRoute53Client{
			getHostedZoneRecordsFn: func(ctx context.Context, id string) ([]cpaws.DNSRecordSet, error) {
				return nil, fmt.Errorf("access denied")
			},
		}
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, mockR53, nil, "us-east-1")
		s := &AWSStocker{Account: account, conn: conn, logger: zap.NewNop()}

		cluster := &inventory.Cluster{ClusterName: "my-cluster"}
		zone := cpaws.DNSHostedZone{ID: "/hostedzone/Z1"}

		link := s.getConsoleLinkOfCluster(cluster, zone)
		assert.Equal(t, unknownConsoleLinkCode, link)
	})
}

func TestAWSStocker_FindOpenshiftConsoleURLs(t *testing.T) {
	t.Run("assigns console link to matching cluster", func(t *testing.T) {
		mockR53 := &mockRoute53Client{
			getZonesWithTagsFn: func(ctx context.Context) ([]cpaws.DNSHostedZone, error) {
				return []cpaws.DNSHostedZone{
					{
						ID:   "/hostedzone/Z1",
						Name: "example.com.",
						Tags: []cpaws.DNSHostedZoneTag{
							{Key: "kubernetes.io/cluster/my-cluster-abc12", Value: "owned"},
						},
					},
				}, nil
			},
			zoneBelongsToClusterFn: func(cluster *inventory.Cluster, zone cpaws.DNSHostedZone) bool {
				return cluster.ClusterName == "my-cluster"
			},
			getHostedZoneRecordsFn: func(ctx context.Context, id string) ([]cpaws.DNSRecordSet, error) {
				return []cpaws.DNSRecordSet{
					{Name: "console-openshift-console.apps.my-cluster.example.com."},
				}, nil
			},
		}

		account := newTestAccount(t)
		cluster, _ := inventory.NewCluster("my-cluster", "abc12", inventory.AWSProvider, "us-east-1", "", "owner")
		_ = account.AddCluster(cluster)

		conn := cpaws.NewAWSConnectionWithClients(nil, mockR53, nil, "us-east-1")
		s := &AWSStocker{Account: account, conn: conn, logger: zap.NewNop()}

		err := s.FindOpenshiftConsoleURLs()
		assert.NoError(t, err)
		assert.Contains(t, account.Clusters[cluster.ClusterID].ConsoleLink, "my-cluster.example.com.")
	})

	t.Run("get zones error", func(t *testing.T) {
		mockR53 := &mockRoute53Client{
			getZonesWithTagsFn: func(ctx context.Context) ([]cpaws.DNSHostedZone, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, mockR53, nil, "us-east-1")
		s := &AWSStocker{Account: account, conn: conn, logger: zap.NewNop()}

		err := s.FindOpenshiftConsoleURLs()
		assert.Error(t, err)
	})

	t.Run("no matching zone leaves console link empty", func(t *testing.T) {
		mockR53 := &mockRoute53Client{
			getZonesWithTagsFn: func(ctx context.Context) ([]cpaws.DNSHostedZone, error) {
				return []cpaws.DNSHostedZone{
					{ID: "/hostedzone/Z1", Name: "other.com."},
				}, nil
			},
			zoneBelongsToClusterFn: func(cluster *inventory.Cluster, zone cpaws.DNSHostedZone) bool {
				return false
			},
		}

		account := newTestAccount(t)
		cluster, _ := inventory.NewCluster("my-cluster", "abc12", inventory.AWSProvider, "us-east-1", "", "owner")
		_ = account.AddCluster(cluster)

		conn := cpaws.NewAWSConnectionWithClients(nil, mockR53, nil, "us-east-1")
		s := &AWSStocker{Account: account, conn: conn, logger: zap.NewNop()}

		err := s.FindOpenshiftConsoleURLs()
		assert.NoError(t, err)
		assert.Equal(t, "", account.Clusters[cluster.ClusterID].ConsoleLink)
	})
}
