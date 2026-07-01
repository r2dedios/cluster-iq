package cloudprovider

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const unknownAccountIDCode = "Unknown_Account_ID"

// stsAPI defines the subset of the STS API used by AWSSTSConnection.
type stsAPI interface {
	GetCallerIdentity(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

type AWSSTSConnection struct {
	client stsAPI
}

func NewAWSSTSConnection(cfg aws.Config) *AWSSTSConnection {
	return &AWSSTSConnection{
		client: sts.NewFromConfig(cfg),
	}
}

func (c *AWSSTSConnection) GetAWSAccountID(ctx context.Context) string {
	result, err := c.client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return unknownAccountIDCode
	}

	return aws.ToString(result.Account)
}

// WithSTS configures an AWSConnection instance for including the STS client
func WithSTS() AWSConnectionOption {
	return func(conn *AWSConnection) {
		conn.STS = NewAWSSTSConnection(conn.awsCfg)
	}
}
