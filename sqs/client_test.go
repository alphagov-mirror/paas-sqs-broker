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
	"github.com/awslabs/goformation"
	goformationt "github.com/awslabs/goformation/v4/cloudformation"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
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

	Describe("CreateStack", func() {
		It("constructs the full stack when creating", func() {
			instanceID := uuid.NewV4().String()
			orgID := uuid.NewV4().String()
			params := sqs.QueueParams{}
			sqsClient.CreateStack(context.Background(), instanceID, orgID, params)
			Expect(cfnAPI.CreateStackWithContextCallCount()).To(Equal(1))
			_, input, _ := cfnAPI.CreateStackWithContextArgsForCall(0)
			Expect(*input.StackName).To(Equal(fmt.Sprintf("test-queue-prefix-%s", instanceID)))

			Expect(input.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
			Expect(input.TemplateBody).ToNot(BeNil())

			t, err := goformation.ParseYAML([]byte(*input.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			// we should see a queue resource
			// TODO WIP what is not working here?
			Expect(t.Resources[sqs.SQSResourceName]).To(BeAssignableToTypeOf(&goformationsqs.Queue{}))
			v := make([]goformationt.Resource, 0, len(t.Resources))
			for  _, value := range t.Resources {
				v = append(v, value)
			}
			Expect(v).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			queue, ok := t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())

			// fifo should be set to false because we asked for a standard queue
			Expect(queue.FifoQueue).To(BeFalse())

			// should have suitable tags set
			Expect(queue.Tags).To(And(
				ContainElement(goformationtags.Tag{
					Key:   "Name",
					Value: instanceID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Customer",
					Value: orgID,
				}),
				ContainElement(goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				}),
			))

			// we set this in provdata
			Expect(queue.ContentBasedDeduplication).To(BeTrue())

		})
	})

	Describe("GetStackStatus", func() {
		// TODO
	})
})
