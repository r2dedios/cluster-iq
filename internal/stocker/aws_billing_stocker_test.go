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

type mockCostExplorerClient struct {
	getCostAndUsageFn func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error)
}

func (m *mockCostExplorerClient) GetCostAndUsageWithResources(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
	return m.getCostAndUsageFn(ctx, input)
}

func TestAWSBillingStocker_getInstanceExpenses(t *testing.T) {
	t.Run("adds expenses to instance", func(t *testing.T) {
		mockCE := &mockCostExplorerClient{
			getCostAndUsageFn: func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
				assert.Equal(t, "i-111", input.InstanceID)
				return &cpaws.CostAndUsageOutput{
					Results: []cpaws.CostResult{
						{StartDate: "2024-01-01", Amount: "1.50"},
						{StartDate: "2024-01-02", Amount: "2.75"},
					},
				}, nil
			},
		}

		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, mockCE, "us-east-1")
		s := &AWSBillingStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instance := &inventory.Instance{InstanceID: "i-111"}
		err := s.getInstanceExpenses(instance)
		assert.NoError(t, err)
		assert.Len(t, instance.Expenses, 2)
	})

	t.Run("handles RFC3339 dates", func(t *testing.T) {
		mockCE := &mockCostExplorerClient{
			getCostAndUsageFn: func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
				return &cpaws.CostAndUsageOutput{
					Results: []cpaws.CostResult{
						{StartDate: "2024-01-01T00:00:00Z", Amount: "3.00"},
					},
				}, nil
			},
		}

		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, mockCE, "us-east-1")
		s := &AWSBillingStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instance := &inventory.Instance{InstanceID: "i-111"}
		err := s.getInstanceExpenses(instance)
		assert.NoError(t, err)
		assert.Len(t, instance.Expenses, 1)
	})

	t.Run("api error", func(t *testing.T) {
		mockCE := &mockCostExplorerClient{
			getCostAndUsageFn: func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
				return nil, fmt.Errorf("access denied")
			},
		}

		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, mockCE, "us-east-1")
		s := &AWSBillingStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instance := &inventory.Instance{InstanceID: "i-111"}
		err := s.getInstanceExpenses(instance)
		assert.Error(t, err)
	})

	t.Run("empty results", func(t *testing.T) {
		mockCE := &mockCostExplorerClient{
			getCostAndUsageFn: func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
				return &cpaws.CostAndUsageOutput{Results: []cpaws.CostResult{}}, nil
			},
		}

		account := newTestAccount(t)
		conn := cpaws.NewAWSConnectionWithClients(nil, nil, mockCE, "us-east-1")
		s := &AWSBillingStocker{
			Account: account,
			conn:    conn,
			logger:  zap.NewNop(),
		}

		instance := &inventory.Instance{InstanceID: "i-111"}
		err := s.getInstanceExpenses(instance)
		assert.NoError(t, err)
		assert.Empty(t, instance.Expenses)
	})
}

func TestAWSBillingStocker_MakeStock(t *testing.T) {
	t.Run("queries only target instances", func(t *testing.T) {
		queriedIDs := []string{}
		mockCE := &mockCostExplorerClient{
			getCostAndUsageFn: func(ctx context.Context, input *cpaws.CostAndUsageInput) (*cpaws.CostAndUsageOutput, error) {
				queriedIDs = append(queriedIDs, input.InstanceID)
				return &cpaws.CostAndUsageOutput{
					Results: []cpaws.CostResult{
						{StartDate: "2024-01-01", Amount: "1.00"},
					},
				}, nil
			},
		}

		account := newTestAccount(t)
		cluster, _ := inventory.NewCluster("test-cluster", "abc12", inventory.AWSProvider, "us-east-1", "", "owner")
		_ = account.AddCluster(cluster)
		_ = cluster.AddInstance(&inventory.Instance{InstanceID: "i-111"})
		_ = cluster.AddInstance(&inventory.Instance{InstanceID: "i-222"})
		_ = cluster.AddInstance(&inventory.Instance{InstanceID: "i-333"})

		conn := cpaws.NewAWSConnectionWithClients(nil, nil, mockCE, "us-east-1")
		s := &AWSBillingStocker{
			Account:     account,
			conn:        conn,
			logger:      zap.NewNop(),
			InstanceIDs: []string{"i-111", "i-333"},
		}

		err := s.MakeStock()
		assert.NoError(t, err)
		assert.ElementsMatch(t, []string{"i-111", "i-333"}, queriedIDs)
	})
}
