package sqs_test

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/service/cloudformation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"

	"context"

	provideriface "github.com/alphagov/paas-service-broker-base/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	fakeClient "github.com/alphagov/paas-sqs-broker/sqs/fakes"
)

var _ = Describe("Provider", func() {
	var (
		fakeSQSClient *fakeClient.FakeClient
		sqsProvider   *sqs.SQSProvider
	)

	BeforeEach(func() {
		fakeSQSClient = &fakeClient.FakeClient{}
		sqsProvider = sqs.NewSQSProvider(
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

			Expect(fakeSQSClient.CreateStackCallCount()).To(Equal(1))

			ctx, instanceID, orgID, params := fakeSQSClient.CreateStackArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(instanceID).To(Equal(provisionData.InstanceID))
			Expect(orgID).To(Equal(provisionData.Details.OrganizationGUID))
			Expect(params.Tags).To(And(
				HaveKeyWithValue("Service", "sqs"),
				HaveKeyWithValue("Name", provisionData.InstanceID),
				HaveKeyWithValue("Customer", provisionData.Details.OrganizationGUID),
				HaveKeyWithValue("Environment", "test"),
			))
		})
	})

	DescribeTable("last operation fetches stack status",
		func(cloudformationStatus string, expectedServiceStatus domain.LastOperationState) {
			fakeSQSClient.GetStackStatusReturnsOnCall(0, cloudformationStatus, nil)
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
		It("errors when deleting a non-existent stack", func() {
			fakeSQSClient.DeleteStackReturnsOnCall(
				0,
				sqs.ErrStackNotFound,
			)

			deprovisionData := provideriface.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			_, _, err := sqsProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
		})
		It("deletes a cloudformation stack", func() {
			deprovisionData := provideriface.DeprovisionData{
				InstanceID: "09E1993E-62E2-4040-ADF2-4D3EC741EFE6",
			}
			operationData, isAsync, err := sqsProvider.Deprovision(context.Background(), deprovisionData)
			Expect(err).NotTo(HaveOccurred())
			Expect(operationData).To(Equal("deprovision"))
			Expect(isAsync).To(BeTrue())

			Expect(fakeSQSClient.DeleteStackCallCount()).To(Equal(1))
			ctx, instanceID := fakeSQSClient.DeleteStackArgsForCall(0)
			Expect(ctx).ToNot(BeNil())
			Expect(instanceID).To(Equal(deprovisionData.InstanceID))
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
				Expect(err).To(MatchError(sqs.ErrUpdateNotSupported))
			})
		})

	*/
})
