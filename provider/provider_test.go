package provider_test

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	goformation "github.com/awslabs/goformation/v4"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags "github.com/awslabs/goformation/v4/cloudformation/tags"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi/domain"

	"context"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
)

var _ = Describe("Provider", func() {
	var (
		fakeSQSClient *fakeClient.FakeClient
		sqsProvider   *provider.SQSProvider
	)

	BeforeEach(func() {
		fakeSQSClient = &fakeClient.FakeClient{}
		sqsProvider = provider.NewSQSProvider(
			fakeSQSClient,
			"test", // environment
		)
	})

	Describe("Provision", func() {
		It("creates a cloudformation stack", func() {
			provisionData := provideriface.ProvisionData{
				InstanceID: "a5da1b66-da42-4c83-b806-f287bc589ab3",
				Plan: domain.ServicePlan{
					Name: "standard",
					ID:   "uuid-2",
				},
				Details: domain.ProvisionDetails{
					OrganizationGUID: "27b72d3f-9401-4b45-a7e7-40b17819954f",
					RawParameters: json.RawMessage(`{
						"contentBasedDeduplication": true
					}`),
				},
			}
			dashboardURL, operationData, isAsync, err := sqsProvider.Provision(context.Background(), provisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(dashboardURL).To(Equal(""))
			Expect(operationData).To(Equal("provision"))
			Expect(isAsync).To(BeTrue())

			Expect(fakeSQSClient.CreateStackWithContextCallCount()).To(Equal(1))

			ctx, input, _ := fakeSQSClient.CreateStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.Capabilities).To(ConsistOf(
				aws.String("CAPABILITY_NAMED_IAM"),
			))
			Expect(input.TemplateBody).ToNot(BeNil())
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("paas-sqs-broker-%s", provisionData.InstanceID))))

			t, err := goformation.ParseYAML([]byte(*input.TemplateBody))
			Expect(err).ToNot(HaveOccurred())

			// we should see a queue resource
			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			queue, ok := t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())

			// fifo should be set to false because we asked for a standard queue
			Expect(queue.FifoQueue).To(BeFalse())

			// should have suitable tags set
			Expect(queue.Tags).To(ContainElements(
				goformationtags.Tag{
					Key:   "Name",
					Value: provisionData.InstanceID,
				},
				goformationtags.Tag{
					Key:   "Service",
					Value: "sqs",
				},
				goformationtags.Tag{
					Key:   "Customer",
					Value: provisionData.Details.OrganizationGUID,
				},
				goformationtags.Tag{
					Key:   "Environment",
					Value: "test",
				},
			))

			// the queue resource should have had values from provisionData passed through
			queueName := fmt.Sprintf("paas-sqs-broker-%s", provisionData.InstanceID)
			Expect(queue.QueueName).To(Equal(queueName))

			// we set this in provdata
			Expect(queue.ContentBasedDeduplication).To(BeTrue())
		})
	})

	DescribeTable("last operation fetches stack status",
		func(cloudformationStatus string, expectedServiceStatus domain.LastOperationState) {
			fakeSQSClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformationStatus),
					},
				},
			}, nil)
			lastOperationData := provideriface.LastOperationData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			state, _, err := sqsProvider.LastOperation(context.Background(), lastOperationData)
			Expect(err).NotTo(HaveOccurred())

			Expect(state).To(Equal(expectedServiceStatus))
		},

		Entry("delete failed",
			cloudformation.StackStatusDeleteFailed,
			domain.Failed,
		),
		Entry("create failed",
			cloudformation.StackStatusCreateFailed,
			domain.Failed,
		),
		Entry("rollback failed",
			cloudformation.StackStatusRollbackFailed,
			domain.Failed,
		),
		Entry("update rollback failed",
			cloudformation.StackStatusUpdateRollbackFailed,
			domain.Failed,
		),
		Entry("rollback complete",
			cloudformation.StackStatusRollbackComplete,
			domain.Failed,
		),
		Entry("update rollback complete",
			cloudformation.StackStatusUpdateRollbackComplete,
			domain.Failed,
		),
		Entry("create complete",
			cloudformation.StackStatusCreateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusUpdateComplete,
			domain.Succeeded,
		),
		Entry("update complete",
			cloudformation.StackStatusDeleteComplete,
			domain.Succeeded,
		),
		Entry("create in progress",
			cloudformation.StackStatusCreateInProgress,
			domain.InProgress,
		),
		Entry("update in progress",
			cloudformation.StackStatusUpdateInProgress,
			domain.InProgress,
		),
		Entry("delete in progress",
			cloudformation.StackStatusDeleteInProgress,
			domain.InProgress,
		),
		Entry("rollback in progress",
			cloudformation.StackStatusRollbackInProgress,
			domain.InProgress,
		),
	)

	Describe("Deprovision", func() {
		It("deletes a cloudformation stack", func() {
			fakeSQSClient.DescribeStacksWithContextReturnsOnCall(0, &cloudformation.DescribeStacksOutput{
				Stacks: []*cloudformation.Stack{
					{
						StackName:   aws.String("some stack"),
						StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
					},
				},
			}, nil)

			deprovisionData := provideriface.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			operationData, isAsync, err := sqsProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(operationData).To(Equal("deprovision"))
			Expect(isAsync).To(BeTrue())

			Expect(fakeSQSClient.DeleteStackWithContextCallCount()).To(Equal(1))
			ctx, input, _ := fakeSQSClient.DeleteStackWithContextArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("paas-sqs-broker-%s", deprovisionData.InstanceID))))
		})
	})

	/*
		Describe("Bind", func() {
		})

		Describe("Unbind", func() {
		})

		Describe("Update", func() {
			It("does not support updating a bucket", func() {
				updateData := provideriface.UpdateData{
					InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				}

				_, _, err := sqsProvider.Update(context.Background(), updateData)
				Expect(err).To(MatchError(provider.ErrUpdateNotSupported))
			})
		})

	*/
})
