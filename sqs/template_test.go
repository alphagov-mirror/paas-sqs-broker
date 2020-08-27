package sqs_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	goformationsqs "github.com/awslabs/goformation/v4/cloudformation/sqs"
	goformationtags"github.com/awslabs/goformation/v4/cloudformation/tags"
	"github.com/alphagov/paas-sqs-broker/sqs"
)

var _ = Describe("Template", func() {
	var ()

	BeforeEach(func() {
		// logger = lager.NewLogger("sqs-service-broker-test")
	})

	It("should not have any input parameters", func() {
		t, err := sqs.QueueTemplate(sqs.QueueParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Parameters).To(BeEmpty())
	})

	Context("queue", func() {
		var queue *goformationsqs.Queue
		var dlqueue *goformationsqs.Queue
		var params sqs.QueueParams

		BeforeEach(func() {
			params = sqs.QueueParams{}
		})

		JustBeforeEach(func() {
			t, err := sqs.QueueTemplate(params)
			Expect(err).ToNot(HaveOccurred())
			Expect(t.Resources).To(ContainElement(BeAssignableToTypeOf(&goformationsqs.Queue{})))
			var ok bool
			queue, ok = t.Resources[sqs.SQSResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())
			dlqueue, ok = t.Resources[sqs.SQSDLQResourceName].(*goformationsqs.Queue)
			Expect(ok).To(BeTrue())
		})

		Context("when queueName is set", func() {
			BeforeEach(func() {
				params.QueueName = "paas-sqs-broker-a"
			})
			It("should have a queue name prefixed with broker prefix", func() {
				Expect(queue.QueueName).To(Equal("paas-sqs-broker-a"))
			})

		})


		Context("when tags are set", func() {
			BeforeEach(func() {
				params.Tags = map[string]string{
					"Service": "sqs",
					"DeployEnv": "autom8",
				}
			})
			It("should have a queue name prefixed with broker prefix", func() {
				Expect(queue.Tags).To(ConsistOf(
					goformationtags.Tag{ // auto-injected
						Key: "QueueType",
						Value: "Main",
					},
					goformationtags.Tag{
						Key: "Service",
						Value: "sqs",
					},
					goformationtags.Tag{
						Key: "DeployEnv",
						Value: "autom8",
					},
				))
				Expect(dlqueue.Tags).To(ConsistOf(
					goformationtags.Tag{ // auto-injected
						Key: "QueueType",
						Value: "Dead-Letter",
					},
					goformationtags.Tag{
						Key: "Service",
						Value: "sqs",
					},
					goformationtags.Tag{
						Key: "DeployEnv",
						Value: "autom8",
					},
				))
			})
		})

		It("should have sensible default values", func() {
			Expect(queue.ContentBasedDeduplication).To(BeFalse())
			Expect(queue.DelaySeconds).To(BeZero())
			Expect(queue.FifoQueue).To(BeFalse())
			Expect(queue.MaximumMessageSize).To(BeZero())
			Expect(queue.MessageRetentionPeriod).To(BeZero())
			Expect(queue.ReceiveMessageWaitTimeSeconds).To(BeZero())
			Expect(queue.RedrivePolicy).To(BeEmpty())
			Expect(queue.VisibilityTimeout).To(BeZero())
		})

		Context("when contentBasedDeduplication is set", func() {
			BeforeEach(func() {
				params.ContentBasedDeduplication = true
			})
			It("should set queue ContentBasedDeduplication from spec", func() {
				Expect(queue.ContentBasedDeduplication).To(BeTrue())
			})
		})
	})

	It("should have outputs for connection details", func() {
		t, err := sqs.QueueTemplate(sqs.QueueParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(t.Outputs).To(And(
			HaveKey("QueueURL"),
			HaveKey("DLQueueURL"),
		))
	})
})
