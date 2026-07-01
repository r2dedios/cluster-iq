package cloudprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ec2API defines the subset of the EC2 API used by AWSEC2Connection.
type ec2API interface {
	DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeRegions(ctx context.Context, input *ec2.DescribeRegionsInput, opts ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
	StopInstances(ctx context.Context, input *ec2.StopInstancesInput, opts ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
	StartInstances(ctx context.Context, input *ec2.StartInstancesInput, opts ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
}

// AWSEC2Connection represents the EC2 client for AWS
type AWSEC2Connection struct {
	client ec2API
	region string
}

// NewAWSEC2Connection creates a new EC2 client instance based on an aws.Config
func NewAWSEC2Connection(cfg aws.Config) *AWSEC2Connection {
	return &AWSEC2Connection{
		client: ec2.NewFromConfig(cfg),
		region: cfg.Region,
	}
}

// WithEC2 configures an AWSConnection instance for including the EC2 client
func WithEC2() AWSConnectionOption {
	return func(conn *AWSConnection) {
		conn.EC2 = NewAWSEC2Connection(conn.awsCfg)
	}
}

// GetRegion returns the configured region for the AWSEC2Connection
func (c AWSEC2Connection) GetRegion() string {
	return c.region
}

func (c *AWSEC2Connection) FilterExistingInstances(ctx context.Context, instanceIDs []string) ([]string, error) {
	if len(instanceIDs) == 0 {
		return nil, nil
	}

	input := &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-id"),
				Values: instanceIDs,
			},
		},
	}

	result, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to filter existing instances: %w", err)
	}

	var existingIDs []string
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			existingIDs = append(existingIDs, aws.ToString(instance.InstanceId))
		}
	}

	return existingIDs, nil
}

// StopClusterInstances stops EC2 instances from the provided instances list.
func (c *AWSEC2Connection) StopClusterInstances(ctx context.Context, instanceIDs []string) error {
	existingIDs, err := c.FilterExistingInstances(ctx, instanceIDs)
	if err != nil {
		return err
	}

	if len(existingIDs) == 0 {
		return nil
	}
	input := &ec2.DescribeInstancesInput{
		InstanceIds: existingIDs,
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{string(ec2types.InstanceStateNameRunning)},
			},
		},
	}

	result, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe instances: %w", err)
	}

	if result == nil || len(result.Reservations) == 0 {
		return nil
	}

	var runningInstanceIDs []string
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			runningInstanceIDs = append(runningInstanceIDs, aws.ToString(instance.InstanceId))
		}
	}

	stopInput := &ec2.StopInstancesInput{
		InstanceIds: runningInstanceIDs,
	}

	_, err = c.client.StopInstances(ctx, stopInput)
	if err != nil {
		return fmt.Errorf("error stopping instances: %w", err)
	}

	return nil
}

// StartClusterInstances starts EC2 instances from the provided instances list.
func (c *AWSEC2Connection) StartClusterInstances(ctx context.Context, instanceIDs []string) error {
	existingIDs, err := c.FilterExistingInstances(ctx, instanceIDs)
	if err != nil {
		return err
	}

	if len(existingIDs) == 0 {
		return nil
	}
	input := &ec2.DescribeInstancesInput{
		InstanceIds: existingIDs,
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{string(ec2types.InstanceStateNameStopped)},
			},
		},
	}

	result, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to describe instances: %w", err)
	}

	if result == nil || len(result.Reservations) == 0 {
		return nil
	}

	var stoppedInstanceIDs []string
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			stoppedInstanceIDs = append(stoppedInstanceIDs, aws.ToString(instance.InstanceId))
		}
	}

	startInput := &ec2.StartInstancesInput{
		InstanceIds: stoppedInstanceIDs,
	}

	_, err = c.client.StartInstances(ctx, startInput)
	if err != nil {
		return fmt.Errorf("error starting instances: %w", err)
	}

	return nil
}

// GetRegionsList returns a list of the available AWS regions as a string array
func (c *AWSEC2Connection) GetRegionsList(ctx context.Context) ([]string, error) {
	regions, err := c.client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}

	regionList := make([]string, 0)

	for _, region := range regions.Regions {
		regionList = append(regionList, aws.ToString(region.RegionName))
	}

	return regionList, nil
}

// GetInstances gets the list of EC2 instances and returns them as an Array of Inventory.Instances
// Using paginated requests for more efficiency
func (c *AWSEC2Connection) GetInstances(ctx context.Context) ([]inventory.Instance, error) {
	input := &ec2.DescribeInstancesInput{}

	paginator := ec2.NewDescribeInstancesPaginator(c.client, input)

	var instances []inventory.Instance
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting EC2 instances reservations: %w", err)
		}
		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				newInstance, err := EC2InstanceToInventoryInstance(instance)
				if err != nil {
					continue
				}
				instances = append(instances, *newInstance)
			}
		}
	}

	return instances, nil
}

// EC2InstanceToInventoryInstance converts an EC2 instance into an inventory.Instance
func EC2InstanceToInventoryInstance(ec2instance ec2types.Instance) (*inventory.Instance, error) {
	id := aws.ToString(ec2instance.InstanceId)
	tags := ConvertEC2TagtoTag(ec2instance.Tags, id)
	name := inventory.GetInstanceNameFromTags(tags)
	instanceType := string(ec2instance.InstanceType)
	availabilityZone := aws.ToString(ec2instance.Placement.AvailabilityZone)
	status := inventory.AsResourceStatus(string(ec2instance.State.Name))
	creationTimestamp := getInstanceCreationTimestamp(ec2instance)

	instance, err := inventory.NewInstance(
		id,
		name,
		inventory.AWSProvider,
		instanceType,
		availabilityZone,
		status,
		tags,
		creationTimestamp,
	)
	if err != nil {
		return nil, err
	}

	return instance, nil
}

// getInstanceCreationTimestamp retrieves the creation timestamp of an EC2 instance.
func getInstanceCreationTimestamp(instance ec2types.Instance) time.Time {
	if instance.RootDeviceName != nil {
		for _, mapping := range instance.BlockDeviceMappings {
			if mapping.DeviceName != nil && *mapping.DeviceName == *instance.RootDeviceName {
				if mapping.Ebs != nil && mapping.Ebs.AttachTime != nil {
					return *mapping.Ebs.AttachTime
				}
			}
		}
	}

	if instance.LaunchTime != nil {
		return *instance.LaunchTime
	}

	return time.Time{}
}
