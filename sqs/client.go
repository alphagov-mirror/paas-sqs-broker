package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
)

var (
	// ErrStackNotFound returned when stack does not exist, or has been deleted
	ErrStackNotFound = fmt.Errorf("STACK_NOT_FOUND")
	// NoExistErrMatch is a string to match if stack does not exist
	NoExistErrMatch = "does not exist"
)

//go:generate counterfeiter -o fakes/fake_sqs_client.go . Client
type Client interface {
	DescribeStacksWithContext(context.Context, *cloudformation.DescribeStacksInput, ...request.Option) (*cloudformation.DescribeStacksOutput, error)
	// DescribeStackEventsWithContext(aws.Context, *cloudformation.DescribeStackEventsInput, ...request.Option) (*cloudformation.DescribeStackEventsOutput, error)
	CreateStackWithContext(context.Context, *cloudformation.CreateStackInput, ...request.Option) (*cloudformation.CreateStackOutput, error)
	UpdateStackWithContext(context.Context, *cloudformation.UpdateStackInput, ...request.Option) (*cloudformation.UpdateStackOutput, error)
	DeleteStack(ctx context.Context, instanceID string) error
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
	resourcePrefix    string
	iamUserPath       string
	awsRegion         string
	deployEnvironment string
	timeout           time.Duration
	cfnClient         cloudformationiface.CloudFormationAPI
	logger            lager.Logger
	context           context.Context
}

func NewSQSClient(
	ctx context.Context,
	config *Config,
	cfnClient cloudformationiface.CloudFormationAPI,
	logger lager.Logger,
) *SQSClient {
	timeout := config.Timeout
	if timeout == time.Duration(0) {
		timeout = 30 * time.Second
	}

	return &SQSClient{
		resourcePrefix:    config.ResourcePrefix,
		iamUserPath:       fmt.Sprintf("/%s/", strings.Trim(config.IAMUserPath, "/")),
		awsRegion:         config.AWSRegion,
		deployEnvironment: config.DeployEnvironment,
		timeout:           timeout,
		cfnClient:         cfnClient,
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

func (s *SQSClient) DeleteStack(ctx context.Context, instanceID string) error {
	stackName := s.getStackName(instanceID)
	_, err := s.getStack(ctx, stackName)
	if err != nil {
		return err
	}
	s.cfnClient.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	return nil
}

func (s *SQSClient) getStack(ctx context.Context, stackName string) (*cloudformation.Stack, error) {
	describeOutput, err := s.cfnClient.DescribeStacksWithContext(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		if IsNotFoundError(err) {
			return nil, ErrStackNotFound
		}
		return nil, err
	}
	if describeOutput == nil {
		return nil, fmt.Errorf("describeOutput was nil, potential issue with AWS Client")
	}
	if len(describeOutput.Stacks) == 0 {
		return nil, fmt.Errorf("describeOutput contained no Stacks, potential issue with AWS Client")
	}
	if len(describeOutput.Stacks) > 1 {
		return nil, fmt.Errorf("describeOutput contained multiple Stacks which is unexpected when calling with StackName, potential issue with AWS Client")
	}
	state := describeOutput.Stacks[0]
	if state.StackStatus == nil {
		return nil, fmt.Errorf("describeOutput contained a nil StackStatus, potential issue with AWS Client")
	}
	return state, nil
}

func (s *SQSClient) getStackName(instanceID string) string {
	return fmt.Sprintf("%s%s", s.resourcePrefix, instanceID)
}

func IsNotFoundError(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "ResourceNotFoundException" {
			return true
		} else if awsErr.Code() == "ValidationError" && strings.Contains(awsErr.Message(), NoExistErrMatch) {
			return true
		}
	}
	return false
}

type AWSClient struct {
	*cloudformation.CloudFormation
}
