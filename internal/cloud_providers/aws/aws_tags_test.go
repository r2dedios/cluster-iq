package cloudprovider

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestConvertEC2TagtoTag(t *testing.T) {
	t.Run("normal tags", func(t *testing.T) {
		ec2Tags := []ec2types.Tag{
			{Key: aws.String("Name"), Value: aws.String("my-instance")},
			{Key: aws.String("env"), Value: aws.String("prod")},
		}
		tags := ConvertEC2TagtoTag(ec2Tags, "i-123")
		assert.Len(t, tags, 2)
		assert.Equal(t, "Name", tags[0].Key)
		assert.Equal(t, "my-instance", tags[0].Value)
		assert.Equal(t, "i-123", tags[0].InstanceID)
		assert.Equal(t, "env", tags[1].Key)
		assert.Equal(t, "prod", tags[1].Value)
	})

	t.Run("empty slice", func(t *testing.T) {
		tags := ConvertEC2TagtoTag([]ec2types.Tag{}, "i-123")
		assert.Empty(t, tags)
	})

	t.Run("nil slice", func(t *testing.T) {
		tags := ConvertEC2TagtoTag(nil, "i-123")
		assert.Empty(t, tags)
	})

	t.Run("nil key and value", func(t *testing.T) {
		ec2Tags := []ec2types.Tag{
			{Key: nil, Value: nil},
		}
		tags := ConvertEC2TagtoTag(ec2Tags, "i-123")
		assert.Len(t, tags, 1)
		assert.Equal(t, "", tags[0].Key)
		assert.Equal(t, "", tags[0].Value)
	})
}
