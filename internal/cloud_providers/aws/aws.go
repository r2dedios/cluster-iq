package cloudprovider

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	// Default Region for AWS CLI
	DefaultAWSRegion = "eu-west-1"
)

// AWSConnection defines the connection with AWS APIs and its various
// services. It can be customized based on the user's requirements.
// To add a new service to AWSConnection, include the corresponding
// "With<SERVICE>()" method available in this package.
//
// Currently supported services:
// * EC2 (computing)
// * Route53 (DNS)
// * STS (SecurityTokenService)
// * CostExplorer (billing data)
type AWSConnection struct {
	awsCfg       aws.Config
	EC2          EC2Client
	Route53      Route53Client
	STS          STSClient
	CostExplorer CostExplorerClient
	accountID    string
	user         string
	password     string
	region       string
	unmanaged    bool
}

// AWSConnectionOption defines the options for creating different sets of AWS services connections
type AWSConnectionOption func(*AWSConnection)

// NewAWSConnection creates a connection with AWS APIs. Based on the AWSConnectionOptions, it will create different clients for every available service
func NewAWSConnection(ctx context.Context, user string, password string, region string, opts ...AWSConnectionOption) (*AWSConnection, error) {
	if region == "" {
		region = DefaultAWSRegion
	}

	conn := &AWSConnection{
		user:     user,
		password: password,
		region:   region,
	}

	if err := conn.loadConfig(ctx); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(conn)
	}

	if conn.STS != nil {
		accountID := conn.STS.GetAWSAccountID(ctx)
		if accountID == unknownAccountIDCode {
			return nil, fmt.Errorf("credential validation failed: could not retrieve AWS account ID")
		}
		conn.accountID = accountID
	}

	return conn, nil
}

// loadConfig creates the AWS config with static credentials and the configured region
func (conn *AWSConnection) loadConfig(ctx context.Context) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(conn.user, conn.password, "")),
		awsconfig.WithRegion(conn.region),
	)
	if err != nil {
		return fmt.Errorf("cannot load AWS config for Account: %w", err)
	}

	conn.awsCfg = cfg
	return nil
}

// NewAWSConnectionWithClients creates an AWSConnection with pre-injected clients,
// bypassing AWS credential loading. SetRegion and Connect become no-ops on the
// returned connection, so injected clients are never replaced.
func NewAWSConnectionWithClients(ec2 EC2Client, route53 Route53Client, costExplorer CostExplorerClient, region string) *AWSConnection {
	return &AWSConnection{
		EC2:          ec2,
		Route53:      route53,
		CostExplorer: costExplorer,
		region:       region,
		unmanaged:    true,
	}
}

// GetRegion returns the current region configured for the AWS Connection
func (conn AWSConnection) GetRegion() string {
	return conn.region
}

// SetRegion configures a new Region for the AWS Connection and refreshes the service clients for the new target region
func (conn *AWSConnection) SetRegion(ctx context.Context, region string) error {
	conn.region = region
	if conn.unmanaged {
		return nil
	}
	return conn.Connect(ctx)
}

// GetAccountID returns the accountID obtained from AWS for the account on the current AWSConnection
func (conn *AWSConnection) GetAccountID() string {
	return conn.accountID
}

// Connect establish or refresh the AWS service clients for the AWSConnection
// object. This is needed because some clients needs to be re-created when
// switching to a different region
func (conn *AWSConnection) Connect(ctx context.Context) error {
	if conn.unmanaged {
		return nil
	}

	if err := conn.loadConfig(ctx); err != nil {
		return err
	}

	if conn.EC2 != nil {
		WithEC2()(conn)
	}

	if conn.Route53 != nil {
		WithRoute53()(conn)
	}

	if conn.STS != nil {
		WithSTS()(conn)
	}

	if conn.CostExplorer != nil {
		WithCostExplorer()(conn)
	}

	return nil
}
