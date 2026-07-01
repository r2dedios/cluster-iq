package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/stretchr/testify/assert"
)

type mockCostExplorerAPI struct {
	getCostAndUsageWithResourcesFn func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error)
}

func (m *mockCostExplorerAPI) GetCostAndUsageWithResources(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
	return m.getCostAndUsageWithResourcesFn(ctx, input, opts...)
}

func TestAWSCostExplorerConnection_GetCostAndUsageWithResources(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockCostExplorerAPI{
			getCostAndUsageWithResourcesFn: func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
				assert.Equal(t, "2024-01-01", aws.ToString(input.TimePeriod.Start))
				assert.Equal(t, "2024-01-15", aws.ToString(input.TimePeriod.End))
				assert.Equal(t, cetypes.GranularityDaily, input.Granularity)
				assert.Equal(t, []string{"i-123"}, input.Filter.Dimensions.Values)

				return &costexplorer.GetCostAndUsageWithResourcesOutput{
					ResultsByTime: []cetypes.ResultByTime{
						{
							TimePeriod: &cetypes.DateInterval{
								Start: aws.String("2024-01-01"),
								End:   aws.String("2024-01-02"),
							},
							Total: map[string]cetypes.MetricValue{
								"UnblendedCost": {Amount: aws.String("1.50"), Unit: aws.String("USD")},
							},
						},
						{
							TimePeriod: &cetypes.DateInterval{
								Start: aws.String("2024-01-02"),
								End:   aws.String("2024-01-03"),
							},
							Total: map[string]cetypes.MetricValue{
								"UnblendedCost": {Amount: aws.String("2.75"), Unit: aws.String("USD")},
							},
						},
					},
				}, nil
			},
		}
		conn := &AWSCostExplorerConnection{client: mock}
		result, err := conn.GetCostAndUsageWithResources(context.Background(), &CostAndUsageInput{
			StartDate:  "2024-01-01",
			EndDate:    "2024-01-15",
			InstanceID: "i-123",
		})
		assert.NoError(t, err)
		assert.Len(t, result.Results, 2)
		assert.Equal(t, "2024-01-01", result.Results[0].StartDate)
		assert.Equal(t, "1.50", result.Results[0].Amount)
		assert.Equal(t, "2024-01-02", result.Results[1].StartDate)
		assert.Equal(t, "2.75", result.Results[1].Amount)
	})

	t.Run("empty results", func(t *testing.T) {
		mock := &mockCostExplorerAPI{
			getCostAndUsageWithResourcesFn: func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
				return &costexplorer.GetCostAndUsageWithResourcesOutput{
					ResultsByTime: []cetypes.ResultByTime{},
				}, nil
			},
		}
		conn := &AWSCostExplorerConnection{client: mock}
		result, err := conn.GetCostAndUsageWithResources(context.Background(), &CostAndUsageInput{
			StartDate:  "2024-01-01",
			EndDate:    "2024-01-15",
			InstanceID: "i-123",
		})
		assert.NoError(t, err)
		assert.Empty(t, result.Results)
	})

	t.Run("nil total skipped", func(t *testing.T) {
		mock := &mockCostExplorerAPI{
			getCostAndUsageWithResourcesFn: func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
				return &costexplorer.GetCostAndUsageWithResourcesOutput{
					ResultsByTime: []cetypes.ResultByTime{
						{
							TimePeriod: &cetypes.DateInterval{Start: aws.String("2024-01-01")},
							Total:      nil,
						},
						{
							TimePeriod: &cetypes.DateInterval{Start: aws.String("2024-01-02")},
							Total: map[string]cetypes.MetricValue{
								"UnblendedCost": {Amount: aws.String("3.00")},
							},
						},
					},
				}, nil
			},
		}
		conn := &AWSCostExplorerConnection{client: mock}
		result, err := conn.GetCostAndUsageWithResources(context.Background(), &CostAndUsageInput{
			StartDate:  "2024-01-01",
			EndDate:    "2024-01-15",
			InstanceID: "i-123",
		})
		assert.NoError(t, err)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "3.00", result.Results[0].Amount)
	})

	t.Run("api error", func(t *testing.T) {
		mock := &mockCostExplorerAPI{
			getCostAndUsageWithResourcesFn: func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
				return nil, fmt.Errorf("access denied")
			},
		}
		conn := &AWSCostExplorerConnection{client: mock}
		result, err := conn.GetCostAndUsageWithResources(context.Background(), &CostAndUsageInput{
			StartDate:  "2024-01-01",
			EndDate:    "2024-01-15",
			InstanceID: "i-123",
		})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
