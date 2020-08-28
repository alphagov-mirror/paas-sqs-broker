package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

//go:generate counterfeiter -o fakes/fake_sqs_client.go . Client
type Client interface {
	DescribeStacksWithContext(aws.Context, *cloudformation.DescribeStacksInput, ...request.Option) (*cloudformation.DescribeStacksOutput, error)
	// DescribeStackEventsWithContext(aws.Context, *cloudformation.DescribeStackEventsInput, ...request.Option) (*cloudformation.DescribeStackEventsOutput, error)
	CreateStackWithContext(aws.Context, *cloudformation.CreateStackInput, ...request.Option) (*cloudformation.CreateStackOutput, error)
	UpdateStackWithContext(aws.Context, *cloudformation.UpdateStackInput, ...request.Option) (*cloudformation.UpdateStackOutput, error)
	DeleteStackWithContext(aws.Context, *cloudformation.DeleteStackInput, ...request.Option) (*cloudformation.DeleteStackOutput, error)
}

type Config struct {
	AWSRegion         string `json:"aws_region"`
	ResourcePrefix    string `json:"resource_prefix"`
	IAMUserPath       string `json:"iam_user_path"`
	DeployEnvironment string `json:"deploy_env"`
	Timeout           time.Duration
}

func NewSQSClientConfig(configJSON []byte) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal(configJSON, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

type SQSClient struct {
	bucketPrefix      string
	iamUserPath       string
	awsRegion         string
	deployEnvironment string
	timeout           time.Duration
	sqsClient         sqsiface.SQSAPI
	cfnClient         cloudformationiface.CloudFormationAPI
	iamClient         iamiface.IAMAPI
	logger            lager.Logger
	context           context.Context
}

func NewSQSClient(
	config *Config,
	sqsClient sqsiface.SQSAPI,
	cfnClient cloudformationiface.CloudFormationAPI,
	iamClient iamiface.IAMAPI,
	logger lager.Logger,
	ctx context.Context,
) *SQSClient {
	timeout := config.Timeout
	if timeout == time.Duration(0) {
		timeout = 30 * time.Second
	}

	return &SQSClient{
		bucketPrefix:      config.ResourcePrefix,
		iamUserPath:       fmt.Sprintf("/%s/", strings.Trim(config.IAMUserPath, "/")),
		awsRegion:         config.AWSRegion,
		deployEnvironment: config.DeployEnvironment,
		timeout:           timeout,
		sqsClient:         sqsClient,
		cfnClient:         cfnClient,
		iamClient:         iamClient,
		logger:            logger,
		context:           ctx,
	}
}

// TODO
func (s *SQSClient) DescribeStacksWithContext(ctx aws.Context, input *cloudformation.DescribeStacksInput, opts ...request.Option) (*cloudformation.DescribeStacksOutput, error) {
	return s.cfnClient.DescribeStacksWithContext(ctx, input, opts...)
}

func (s *SQSClient) CreateStackWithContext(ctx aws.Context, input *cloudformation.CreateStackInput, opts ...request.Option) (*cloudformation.CreateStackOutput, error) {
	return s.cfnClient.CreateStackWithContext(ctx, input, opts...)
}

func (s *SQSClient) UpdateStackWithContext(ctx aws.Context, input *cloudformation.UpdateStackInput, opts ...request.Option) (*cloudformation.UpdateStackOutput, error) {
	return s.cfnClient.UpdateStackWithContext(ctx, input, opts...)
}

func (s *SQSClient) DeleteStackWithContext(ctx aws.Context, input *cloudformation.DeleteStackInput, opts ...request.Option) (*cloudformation.DeleteStackOutput, error) {
	return s.cfnClient.DeleteStackWithContext(ctx, input, opts...)
}

type AWSClient struct {
	*cloudformation.CloudFormation
}
