package cloudprovider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
)

type mockSTSAPI struct {
	getCallerIdentityFn func(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSAPI) GetCallerIdentity(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return m.getCallerIdentityFn(ctx, input, opts...)
}

func TestAWSSTSConnection_GetAWSAccountID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedID := "123456789012"
		conn := &AWSSTSConnection{
			client: &mockSTSAPI{
				getCallerIdentityFn: func(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return &sts.GetCallerIdentityOutput{
						Account: aws.String(expectedID),
					}, nil
				},
			},
		}
		result := conn.GetAWSAccountID(context.Background())
		assert.Equal(t, expectedID, result)
	})

	t.Run("error returns unknown", func(t *testing.T) {
		conn := &AWSSTSConnection{
			client: &mockSTSAPI{
				getCallerIdentityFn: func(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return nil, fmt.Errorf("access denied")
				},
			},
		}
		result := conn.GetAWSAccountID(context.Background())
		assert.Equal(t, unknownAccountIDCode, result)
	})

	t.Run("nil account returns empty", func(t *testing.T) {
		conn := &AWSSTSConnection{
			client: &mockSTSAPI{
				getCallerIdentityFn: func(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return &sts.GetCallerIdentityOutput{
						Account: nil,
					}, nil
				},
			},
		}
		result := conn.GetAWSAccountID(context.Background())
		assert.Equal(t, "", result)
	})
}
