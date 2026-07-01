package cloudprovider

import (
	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// ConvertEC2TagtoTag transforms the EC2 instance tags into inventory Tag
func ConvertEC2TagtoTag(ec2Tags []ec2types.Tag, instanceID string) []inventory.Tag {
	var tags []inventory.Tag
	for _, tag := range ec2Tags {
		tags = append(tags, *inventory.NewTag(aws.ToString(tag.Key), aws.ToString(tag.Value), instanceID))
	}
	return tags
}
