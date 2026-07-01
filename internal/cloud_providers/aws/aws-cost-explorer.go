package cloudprovider

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type costExplorerAPI interface {
	GetCostAndUsageWithResources(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput, opts ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageWithResourcesOutput, error)
}

// AWSCostExplorerConnection represents the client object for the CostExplorer service
type AWSCostExplorerConnection struct {
	client costExplorerAPI
}

// NewAWSCostExplorerConnection creates a new CostExplorer client from a v2 AWS config.
func NewAWSCostExplorerConnection(cfg aws.Config) *AWSCostExplorerConnection {
	return &AWSCostExplorerConnection{
		client: costexplorer.NewFromConfig(cfg),
	}
}

// WithCostExplorer configures an AWSConnection instance for including the CostExplorer client
func WithCostExplorer() AWSConnectionOption {
	return func(conn *AWSConnection) {
		conn.CostExplorer = NewAWSCostExplorerConnection(conn.awsCfg)
	}
}

// GetCostAndUsageWithResources queries AWS Cost Explorer for instance-level daily costs
// and returns SDK-independent results.
func (c *AWSCostExplorerConnection) GetCostAndUsageWithResources(ctx context.Context, input *CostAndUsageInput) (*CostAndUsageOutput, error) {
	ceInput := &costexplorer.GetCostAndUsageWithResourcesInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(input.StartDate),
			End:   aws.String(input.EndDate),
		},
		Granularity: cetypes.GranularityDaily,
		Filter: &cetypes.Expression{
			Dimensions: &cetypes.DimensionValues{
				Key:    cetypes.DimensionResourceId,
				Values: []string{input.InstanceID},
			},
		},
		Metrics: []string{"UnblendedCost"},
	}

	result, err := c.client.GetCostAndUsageWithResources(ctx, ceInput)
	if err != nil {
		return nil, err
	}

	output := &CostAndUsageOutput{
		Results: make([]CostResult, 0, len(result.ResultsByTime)),
	}
	for _, r := range result.ResultsByTime {
		if r.Total == nil {
			continue
		}
		metric, ok := r.Total["UnblendedCost"]
		if !ok || metric.Amount == nil {
			continue
		}
		cr := CostResult{
			Amount: aws.ToString(metric.Amount),
		}
		if r.TimePeriod != nil {
			cr.StartDate = aws.ToString(r.TimePeriod.Start)
		}
		output.Results = append(output.Results, cr)
	}

	return output, nil
}
