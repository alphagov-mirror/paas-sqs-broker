package provider_test

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"context"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/provider"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Provider", func() {
	var (
		fakeSQSClient *fakeClient.FakeClient
		sqsProvider   *provider.SQSProvider
	)

	BeforeEach(func() {
		fakeSQSClient = &fakeClient.FakeClient{}
		sqsProvider = provider.NewSQSProvider(fakeSQSClient)
	})

	Describe("Provision", func() {
		It("creates a cloudformation stack", func() {
			provisionData := provideriface.ProvisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
				Plan: brokerapi.ServicePlan{
					Name: "standard",
					ID:   "uuid-2",
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
			Expect(*input.TemplateBody).ToNot(BeEmpty())
			Expect(input.StackName).To(Equal(aws.String(fmt.Sprintf("paas-sqs-broker-%s", provisionData.InstanceID))))
			Expect(input.Parameters).To(ConsistOf(
				&cloudformation.Parameter{
					ParameterKey:   aws.String("QueueName"),
					ParameterValue: aws.String(fmt.Sprintf("paas-sqs-broker-%s", provisionData.InstanceID)),
				},
				&cloudformation.Parameter{
					ParameterKey:   aws.String("FifoQueue"),
					ParameterValue: aws.String("false"),
				},
			))
		})
	})

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

		Describe("LastOperation", func() {
			It("returns success unconditionally", func() {
				state, description, err := sqsProvider.LastOperation(context.Background(), provideriface.LastOperationData{})
				Expect(err).NotTo(HaveOccurred())
				Expect(description).To(Equal("Last operation polling not required. All operations are synchronous."))
				Expect(state).To(Equal(brokerapi.Succeeded))
			})
		})
	*/
})
