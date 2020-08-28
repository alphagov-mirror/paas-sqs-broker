package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/pivotal-cf/brokerapi"
)

var (
	// PollingInterval is the duration between calls to check state when waiting for apply/destroy to complete
	PollingInterval = time.Second * 15
)

type SQSProvider struct {
	Environment string
	client      sqs.Client
}

func NewSQSProvider(sqsClient sqs.Client, env string) *SQSProvider {
	return &SQSProvider{
		Environment: env,
		client:      sqsClient,
	}
}

func (s *SQSProvider) Provision(ctx context.Context, provisionData provideriface.ProvisionData) (dashboardURL string, operationData string, isAsync bool, err error) {
	params := sqs.QueueParams{} // TODO eww
	if err := json.Unmarshal(provisionData.Details.RawParameters, &params); err != nil {
		return "", "", false, err
	}

	params.Tags = map[string]string{
		"Name":        provisionData.InstanceID,
		"Service":     "sqs",
		"Customer":    provisionData.Details.OrganizationGUID,
		"Environment": s.Environment,
	}

	if provisionData.Plan.Name == "fifo" {
		params.FifoQueue = true
	}

	err = s.client.CreateStack(ctx, provisionData.InstanceID, provisionData.Details.OrganizationGUID, params)
	if err != nil {
		return "", "", false, err // TODO
	}
	return "", "provision", true, nil
}

func (s *SQSProvider) Deprovision(ctx context.Context, deprovisionData provideriface.DeprovisionData) (operationData string, isAsync bool, err error) {
	// stackName := s.getStackName(deprovisionData.InstanceID)
	// stack, err := s.getStack(ctx, stackName)
	// if err == ErrStackNotFound {
	// 	// resource is already deleted (or never existsed)
	// 	// so we're done here
	// 	return "", false, brokerapi.ErrInstanceDoesNotExist
	// } else if err != nil {
	// 	// failed to get stack status
	// 	return "", false, err // should this be async and checked later
	// }
	// if *stack.StackStatus == cloudformation.StackStatusDeleteComplete {
	// 	// resource already deleted
	// 	return "", false, nil
	// }
	// // trigger a delete unless we're already in a deleting state
	// if *stack.StackStatus != cloudformation.StackStatusDeleteInProgress {
	// 	_, err := s.client.DeleteStackWithContext(ctx, &cloudformation.DeleteStackInput{
	// 		StackName: aws.String(stackName),
	// 	})
	// 	if err != nil {
	// 		return "", false, err
	// 	}
	// }

	err = s.client.DeleteStack(ctx, deprovisionData.InstanceID)

	if err == sqs.ErrStackNotFound {
		// resource is already deleted (or never existed)
		// so we're done here
		return "", false, brokerapi.ErrInstanceDoesNotExist
	}

	return "deprovision", true, err
}

func (s *SQSProvider) Bind(ctx context.Context, bindData provideriface.BindData) (
	binding brokerapi.Binding, err error) {

	return brokerapi.Binding{
		IsAsync:     false,
		Credentials: brokerapi.Binding{},
	}, nil
}

func (s *SQSProvider) Unbind(ctx context.Context, unbindData provideriface.UnbindData) (
	unbinding brokerapi.UnbindSpec, err error) {

	return brokerapi.UnbindSpec{
		IsAsync: false,
	}, nil
}

var ErrUpdateNotSupported = errors.New("Updating the SQS queue is currently not supported")

func (s *SQSProvider) Update(ctx context.Context, updateData provideriface.UpdateData) (operationData string, isAsync bool, err error) {
	// _, err = r.Client.UpdateStackWithContext(ctx, &UpdateStackInput{
	// 	Capabilities:    capabilities,
	// 	TemplateBody:    aws.String(string(yaml)),
	// 	StackName:       aws.String(stack.GetStackName()),
	// 	StackPolicyBody: stackPolicy,
	// 	Parameters:      params,
	// })
	// if err != nil && !IsNoUpdateError(err) {
	// 	return err
	// }
	// return "", true, nil
	return "", false, ErrUpdateNotSupported
}

func (s *SQSProvider) LastOperation(ctx context.Context, lastOperationData provideriface.LastOperationData) (state brokerapi.LastOperationState, description string, err error) {
	status, err := s.client.GetStackStatus(ctx, lastOperationData.InstanceID)
	if err != nil {
		return "", "", err
	}
	switch status {
	case cloudformation.StackStatusDeleteFailed, cloudformation.StackStatusCreateFailed, cloudformation.StackStatusRollbackFailed, cloudformation.StackStatusUpdateRollbackFailed, cloudformation.StackStatusRollbackComplete, cloudformation.StackStatusUpdateRollbackComplete:
		return brokerapi.Failed, fmt.Sprintf("failed: %s", status), nil
	case cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete, cloudformation.StackStatusDeleteComplete:
		return brokerapi.Succeeded, "ready", nil
	default:
		return brokerapi.InProgress, "pending", nil
	}
}

func IsNotFoundError(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "ResourceNotFoundException" {
			return true
		} else if awsErr.Code() == "ValidationError" && strings.Contains(awsErr.Message(), sqs.NoExistErrMatch) {
			return true
		}
	}
	return false
}

func in(needle string, haystack []string) bool {
	for _, s := range haystack {
		if needle == s {
			return true
		}
	}
	return false
}
