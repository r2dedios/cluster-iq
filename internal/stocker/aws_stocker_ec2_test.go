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

type mockEC2Client struct {
	getRegionFn              func() string
	filterExistingFn         func(ctx context.Context, ids []string) ([]string, error)
	stopClusterInstancesFn   func(ctx context.Context, ids []string) error
	startClusterInstancesFn  func(ctx context.Context, ids []string) error
	getRegionsListFn         func(ctx context.Context) ([]string, error)
	getInstancesFn           func(ctx context.Context) ([]inventory.Instance, error)
}

func (m *mockEC2Client) GetRegion() string {
	if m.getRegionFn != nil {
		return m.getRegionFn()
	}
	return "us-east-1"
}

func (m *mockEC2Client) FilterExistingInstances(ctx context.Context, ids []string) ([]string, error) {
	if m.filterExistingFn != nil {
		return m.filterExistingFn(ctx, ids)
	}
	return nil, nil
}

func (m *mockEC2Client) StopClusterInstances(ctx context.Context, ids []string) error {
	if m.stopClusterInstancesFn != nil {
		return m.stopClusterInstancesFn(ctx, ids)
	}
	return nil
}

func (m *mockEC2Client) StartClusterInstances(ctx context.Context, ids []string) error {
	if m.startClusterInstancesFn != nil {
		return m.startClusterInstancesFn(ctx, ids)
	}
	return nil
}

func (m *mockEC2Client) GetRegionsList(ctx context.Context) ([]string, error) {
	if m.getRegionsListFn != nil {
		return m.getRegionsListFn(ctx)
	}
	return nil, nil
}

func (m *mockEC2Client) GetInstances(ctx context.Context) ([]inventory.Instance, error) {
	if m.getInstancesFn != nil {
		return m.getInstancesFn(ctx)
	}
	return nil, nil
}

func newTestAccount(t *testing.T) *inventory.Account {
	t.Helper()
	acc, err := inventory.NewAccount("acc-123", "test-account", inventory.AWSProvider, "user", "pass")
	assert.NoError(t, err)
	return acc
}

func TestAWSStocker_processInstances(t *testing.T) {
	t.Run("groups instances into clusters", func(t *testing.T) {
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instances := []inventory.Instance{
			{
				InstanceID: "i-111",
				Tags: []inventory.Tag{
					{Key: "kubernetes.io/cluster/my-cluster-abc12", Value: "owned"},
					{Key: "Name", Value: "master-0"},
				},
			},
			{
				InstanceID: "i-222",
				Tags: []inventory.Tag{
					{Key: "kubernetes.io/cluster/my-cluster-abc12", Value: "owned"},
					{Key: "Name", Value: "worker-0"},
				},
			},
		}

		s.processInstances(instances)

		assert.Len(t, account.Clusters, 1)
		for _, cluster := range account.Clusters {
			assert.Equal(t, "my-cluster", cluster.ClusterName)
			assert.Len(t, cluster.Instances, 2)
		}
	})

	t.Run("multiple clusters", func(t *testing.T) {
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instances := []inventory.Instance{
			{
				InstanceID: "i-111",
				Tags: []inventory.Tag{
					{Key: "kubernetes.io/cluster/cluster-a-aaa11", Value: "owned"},
				},
			},
			{
				InstanceID: "i-222",
				Tags: []inventory.Tag{
					{Key: "kubernetes.io/cluster/cluster-b-bbb22", Value: "owned"},
				},
			},
		}

		s.processInstances(instances)

		assert.Len(t, account.Clusters, 2)
	})

	t.Run("skips non-openshift instances when flag set", func(t *testing.T) {
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account:                  account,
			conn:                     conn,
			skipNoOpenShiftInstances: true,
			logger:                   zap.NewNop(),
		}

		instances := []inventory.Instance{
			{
				InstanceID: "i-111",
				Tags: []inventory.Tag{
					{Key: "Name", Value: "standalone-vm"},
				},
			},
		}

		s.processInstances(instances)

		assert.Empty(t, account.Clusters)
	})

	t.Run("includes non-openshift instances when flag not set", func(t *testing.T) {
		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account:                  account,
			conn:                     conn,
			skipNoOpenShiftInstances: false,
			logger:                   zap.NewNop(),
		}

		instances := []inventory.Instance{
			{
				InstanceID: "i-111",
				Tags: []inventory.Tag{
					{Key: "Name", Value: "standalone-vm"},
				},
			},
		}

		s.processInstances(instances)

		assert.Len(t, account.Clusters, 1)
	})
}

func TestAWSStocker_processRegion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		account := newTestAccount(t)
		mockEC2 := &mockEC2Client{
			getInstancesFn: func(ctx context.Context) ([]inventory.Instance, error) {
				return []inventory.Instance{
					{
						InstanceID: "i-111",
						Tags: []inventory.Tag{
							{Key: "kubernetes.io/cluster/my-cluster-abc12", Value: "owned"},
						},
					},
				}, nil
			},
		}
		conn := cpaws.NewAWSConnectionWithClients(mockEC2, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		err := s.processRegion("us-east-1")
		assert.NoError(t, err)
		assert.Len(t, account.Clusters, 1)
	})

	t.Run("get instances error", func(t *testing.T) {
		account := newTestAccount(t)
		mockEC2 := &mockEC2Client{
			getInstancesFn: func(ctx context.Context) ([]inventory.Instance, error) {
				return nil, fmt.Errorf("api error")
			},
		}
		conn := cpaws.NewAWSConnectionWithClients(mockEC2, nil, nil, "us-east-1")
		s := &AWSStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		err := s.processRegion("us-east-1")
		assert.Error(t, err)
		assert.Empty(t, account.Clusters)
	})
}
