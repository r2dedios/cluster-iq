package cloudagent

import (
	"context"
	"fmt"
	"testing"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/actions"
	cpaws "github.com/RHEcosystemAppEng/cluster-iq/internal/cloud_providers/aws"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockEC2Client struct {
	getRegionFn             func() string
	filterExistingFn        func(ctx context.Context, ids []string) ([]string, error)
	stopClusterInstancesFn  func(ctx context.Context, ids []string) error
	startClusterInstancesFn func(ctx context.Context, ids []string) error
	getRegionsListFn        func(ctx context.Context) ([]string, error)
	getInstancesFn          func(ctx context.Context) ([]inventory.Instance, error)
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

func newTestExecutor(t *testing.T, ec2 *mockEC2Client) *AWSExecutor {
	t.Helper()
	account, err := inventory.NewAccount("acc-123", "test-account", inventory.AWSProvider, "user", "pass")
	assert.NoError(t, err)
	conn := cpaws.NewAWSConnectionWithClients(ec2, nil, nil, "us-east-1")
	return &AWSExecutor{
		account: account,
		conn:    conn,
		logger:  zap.NewNop(),
	}
}

func TestAWSExecutor_PowerOnCluster(t *testing.T) {
	t.Run("delegates to StartClusterInstances", func(t *testing.T) {
		var calledWith []string
		mock := &mockEC2Client{
			startClusterInstancesFn: func(ctx context.Context, ids []string) error {
				calledWith = ids
				return nil
			},
		}
		exec := newTestExecutor(t, mock)

		err := exec.PowerOnCluster([]string{"i-111", "i-222"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"i-111", "i-222"}, calledWith)
	})

	t.Run("empty instances returns error", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})
		err := exec.PowerOnCluster([]string{})
		assert.Error(t, err)
	})

	t.Run("ec2 error propagated", func(t *testing.T) {
		mock := &mockEC2Client{
			startClusterInstancesFn: func(ctx context.Context, ids []string) error {
				return fmt.Errorf("ec2 error")
			},
		}
		exec := newTestExecutor(t, mock)

		err := exec.PowerOnCluster([]string{"i-111"})
		assert.Error(t, err)
	})
}

func TestAWSExecutor_PowerOffCluster(t *testing.T) {
	t.Run("delegates to StopClusterInstances", func(t *testing.T) {
		var calledWith []string
		mock := &mockEC2Client{
			stopClusterInstancesFn: func(ctx context.Context, ids []string) error {
				calledWith = ids
				return nil
			},
		}
		exec := newTestExecutor(t, mock)

		err := exec.PowerOffCluster([]string{"i-111"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"i-111"}, calledWith)
	})

	t.Run("empty instances returns error", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})
		err := exec.PowerOffCluster([]string{})
		assert.Error(t, err)
	})
}

func TestAWSExecutor_DeleteCluster(t *testing.T) {
	t.Run("returns not implemented error", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})
		target := actions.ActionTarget{
			ClusterID:   "cluster-123",
			ClusterName: "my-cluster",
			InfraID:     "my-cluster-abc12",
			Region:      "us-east-1",
		}
		err := exec.DeleteCluster(target)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}

func TestAWSExecutor_ProcessAction(t *testing.T) {
	t.Run("dispatches PowerOn", func(t *testing.T) {
		started := false
		mock := &mockEC2Client{
			startClusterInstancesFn: func(ctx context.Context, ids []string) error {
				started = true
				return nil
			},
		}
		exec := newTestExecutor(t, mock)

		target := *actions.NewActionTarget("acc-123", "us-east-1", "cluster-1", []string{"i-111"})
		action := actions.NewPowerOnClusterAction(target, "test", nil)

		err := exec.ProcessAction(action)
		assert.NoError(t, err)
		assert.True(t, started)
	})

	t.Run("dispatches PowerOff", func(t *testing.T) {
		stopped := false
		mock := &mockEC2Client{
			stopClusterInstancesFn: func(ctx context.Context, ids []string) error {
				stopped = true
				return nil
			},
		}
		exec := newTestExecutor(t, mock)

		target := *actions.NewActionTarget("acc-123", "us-east-1", "cluster-1", []string{"i-111"})
		action := actions.NewPowerOffClusterAction(target, "test", nil)

		err := exec.ProcessAction(action)
		assert.NoError(t, err)
		assert.True(t, stopped)
	})

	t.Run("dispatches DeleteCluster", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})

		target := *actions.NewActionTarget("acc-123", "us-east-1", "cluster-1", []string{"i-111"})
		action := actions.NewDeleteClusterAction(target, "test", nil)

		err := exec.ProcessAction(action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("scan returns error", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})

		target := actions.ActionTarget{Region: "us-east-1"}
		action := actions.NewInstantAction(actions.Scan, target, actions.StatusPending, "test", nil, true)

		err := exec.ProcessAction(action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scanner service")
	})

	t.Run("unknown operation returns error", func(t *testing.T) {
		exec := newTestExecutor(t, &mockEC2Client{})

		target := actions.ActionTarget{Region: "us-east-1"}
		action := actions.NewInstantAction("UnknownOp", target, actions.StatusPending, "test", nil, true)

		err := exec.ProcessAction(action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot identify")
	})
}
