package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

type mockEC2API struct {
	describeInstancesFn func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	describeRegionsFn   func(ctx context.Context, input *ec2.DescribeRegionsInput, opts ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
	stopInstancesFn     func(ctx context.Context, input *ec2.StopInstancesInput, opts ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
	startInstancesFn    func(ctx context.Context, input *ec2.StartInstancesInput, opts ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
}

func (m *mockEC2API) DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.describeInstancesFn(ctx, input, opts...)
}

func (m *mockEC2API) DescribeRegions(ctx context.Context, input *ec2.DescribeRegionsInput, opts ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return m.describeRegionsFn(ctx, input, opts...)
}

func (m *mockEC2API) StopInstances(ctx context.Context, input *ec2.StopInstancesInput, opts ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
	return m.stopInstancesFn(ctx, input, opts...)
}

func (m *mockEC2API) StartInstances(ctx context.Context, input *ec2.StartInstancesInput, opts ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
	return m.startInstancesFn(ctx, input, opts...)
}

func TestAWSEC2Connection_FilterExistingInstances(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		conn := &AWSEC2Connection{client: &mockEC2API{}}
		result, err := conn.FilterExistingInstances(context.Background(), []string{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("found instances", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstancesFn: func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				return &ec2.DescribeInstancesOutput{
					Reservations: []ec2types.Reservation{
						{
							Instances: []ec2types.Instance{
								{InstanceId: aws.String("i-111")},
								{InstanceId: aws.String("i-222")},
							},
						},
					},
				}, nil
			},
		}
		conn := &AWSEC2Connection{client: mock}
		result, err := conn.FilterExistingInstances(context.Background(), []string{"i-111", "i-222", "i-333"})
		assert.NoError(t, err)
		assert.Equal(t, []string{"i-111", "i-222"}, result)
	})

	t.Run("api error", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstancesFn: func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				return nil, fmt.Errorf("api error")
			},
		}
		conn := &AWSEC2Connection{client: mock}
		result, err := conn.FilterExistingInstances(context.Background(), []string{"i-111"})
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAWSEC2Connection_GetRegionsList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockEC2API{
			describeRegionsFn: func(ctx context.Context, input *ec2.DescribeRegionsInput, opts ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
				return &ec2.DescribeRegionsOutput{
					Regions: []ec2types.Region{
						{RegionName: aws.String("us-east-1")},
						{RegionName: aws.String("eu-west-1")},
					},
				}, nil
			},
		}
		conn := &AWSEC2Connection{client: mock}
		regions, err := conn.GetRegionsList(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, []string{"us-east-1", "eu-west-1"}, regions)
	})

	t.Run("error", func(t *testing.T) {
		mock := &mockEC2API{
			describeRegionsFn: func(ctx context.Context, input *ec2.DescribeRegionsInput, opts ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
				return nil, fmt.Errorf("network error")
			},
		}
		conn := &AWSEC2Connection{client: mock}
		regions, err := conn.GetRegionsList(context.Background())
		assert.Error(t, err)
		assert.Nil(t, regions)
	})
}

func TestAWSEC2Connection_StopClusterInstances(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		mock := &mockEC2API{
			describeInstancesFn: func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				return &ec2.DescribeInstancesOutput{}, nil
			},
		}
		conn := &AWSEC2Connection{client: mock}
		err := conn.StopClusterInstances(context.Background(), []string{})
		assert.NoError(t, err)
	})

	t.Run("stops running instances", func(t *testing.T) {
		callCount := 0
		mock := &mockEC2API{
			describeInstancesFn: func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				callCount++
				if callCount == 1 {
					return &ec2.DescribeInstancesOutput{
						Reservations: []ec2types.Reservation{
							{Instances: []ec2types.Instance{{InstanceId: aws.String("i-111")}}},
						},
					}, nil
				}
				return &ec2.DescribeInstancesOutput{
					Reservations: []ec2types.Reservation{
						{Instances: []ec2types.Instance{{InstanceId: aws.String("i-111")}}},
					},
				}, nil
			},
			stopInstancesFn: func(ctx context.Context, input *ec2.StopInstancesInput, opts ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error) {
				assert.Equal(t, []string{"i-111"}, input.InstanceIds)
				return &ec2.StopInstancesOutput{}, nil
			},
		}
		conn := &AWSEC2Connection{client: mock}
		err := conn.StopClusterInstances(context.Background(), []string{"i-111"})
		assert.NoError(t, err)
	})
}

func TestAWSEC2Connection_StartClusterInstances(t *testing.T) {
	t.Run("starts stopped instances", func(t *testing.T) {
		callCount := 0
		mock := &mockEC2API{
			describeInstancesFn: func(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				callCount++
				if callCount == 1 {
					return &ec2.DescribeInstancesOutput{
						Reservations: []ec2types.Reservation{
							{Instances: []ec2types.Instance{{InstanceId: aws.String("i-222")}}},
						},
					}, nil
				}
				return &ec2.DescribeInstancesOutput{
					Reservations: []ec2types.Reservation{
						{Instances: []ec2types.Instance{{InstanceId: aws.String("i-222")}}},
					},
				}, nil
			},
			startInstancesFn: func(ctx context.Context, input *ec2.StartInstancesInput, opts ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error) {
				assert.Equal(t, []string{"i-222"}, input.InstanceIds)
				return &ec2.StartInstancesOutput{}, nil
			},
		}
		conn := &AWSEC2Connection{client: mock}
		err := conn.StartClusterInstances(context.Background(), []string{"i-222"})
		assert.NoError(t, err)
	})
}
