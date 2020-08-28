package sqs_test

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-sqs-broker/sqs"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("Client", func() {
	var (
		cfnAPI          *fakeClient.FakeCloudFormationAPI
		sqsClientConfig *sqs.Config
		logger          lager.Logger
		sqsClient       sqs.Client
	)

	BeforeEach(func() {
		cfnAPI = &fakeClient.FakeCloudFormationAPI{}
		logger = lager.NewLogger("sqs-service-broker-test")
		sqsClientConfig = &sqs.Config{
			AWSRegion:         "eu-west-2",
			ResourcePrefix:    "test-queue-prefix-",
			IAMUserPath:       "/test-iam-path/",
			DeployEnvironment: "test-env",
			Timeout:           2 * time.Second,
		}
		sqsClient = sqs.NewSQSClient(
			context.Background(),
			sqsClientConfig,
			cfnAPI,
			logger,
		)
	})

	BeforeEach(func() {
		// logger = lager.NewLogger("sqs-service-broker-test")
	})

	Describe("DeleteStack", func() {
		It("constructs the full stack name when deleting", func() {
			instanceID := uuid.NewV4().String()
			cfnAPI.DescribeStacksWithContextReturnsOnCall(
				0,
				&cloudformation.DescribeStacksOutput{Stacks: []*cloudformation.Stack{
					{StackStatus: aws.String("groovy")},
				}},
				nil,
			)
			sqsClient.DeleteStack(context.Background(), instanceID)
			Expect(cfnAPI.DeleteStackWithContextCallCount()).To(Equal(1))
			_, input, _ := cfnAPI.DeleteStackWithContextArgsForCall(0)
			Expect(*input.StackName).To(Equal(fmt.Sprintf("test-queue-prefix-%s", instanceID)))
		})

		It("returns an error when the stack doesn't exist", func() {
			instanceID := uuid.NewV4().String()
			cfnAPI.DescribeStacksWithContextReturnsOnCall(
				0,
				&cloudformation.DescribeStacksOutput{},
				&fakeClient.MockAWSError{
					C: "ValidationError",
					M: "Stack with id test-queue-prefix-09E1993E-62E2-4040-ADF2-4D3EC741EFE6 does not exist",
				},
			)
			err := sqsClient.DeleteStack(context.Background(), instanceID)
			Expect(err).To(HaveOccurred())
		})
	})

	It("Does a thing", func() {
		Expect(true).To(Equal(true))
	})
})
