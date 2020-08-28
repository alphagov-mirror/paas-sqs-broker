package broker_test

import (
	"context"
	"net/http"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	aws_sqs "github.com/aws/aws-sdk-go/service/sqs"
	cfn "github.com/aws/aws-sdk-go/service/cloudformation"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-service-broker-base/broker"
	brokertesting "github.com/alphagov/paas-service-broker-base/testing"
	"github.com/alphagov/paas-sqs-broker/provider"
	"github.com/alphagov/paas-sqs-broker/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/pivotal-cf/brokerapi"
	uuid "github.com/satori/go.uuid"
)

const (
	ASYNC_ALLOWED = true
)

type BindingResponse struct {
	Credentials map[string]interface{} `json:"credentials"`
}

var _ = Describe("Broker", func() {
	var (
		instanceID string
		serviceID  = "uuid-1"
		planID     = "uuid-2"
	)

	BeforeEach(func() {
		instanceID = uuid.NewV4().String()
	})

	It("should return a 410 response when trying to delete a non-existent instance", func() {
		_, brokerTester := initialise()

		res := brokerTester.Deprovision(instanceID, serviceID, planID, ASYNC_ALLOWED)
		Expect(res.Code).To(Equal(http.StatusGone))
	})

	It("should manage the lifecycle of an SQS queue", func() {
		By("initialising")
		Expect(true).To(Equal(true))
		sqsClientConfig, brokerTester := initialise()
		_, _, _ = sqsClientConfig, brokerTester, instanceID // TODO

		By("Provisioning")

		// defer helpers.DeprovisionService(brokerTester, instanceID, serviceID, planID)

		By("Binding an app")

		// defer helpers.Unbind(brokerTester, instanceID, serviceID, planID, binding1ID)

		By("Asserting the credentials returned work for both reading and writing")

		By("Binding an app as a read-only user")

		// defer helpers.Unbind(brokerTester, instanceID, serviceID, planID, binding2ID)

		By("Asserting that read-only credentials can read, but fail to write to a file")

		By("Asserting the first user's credentials still work for reading and writing")
	})

})

func initialise() (*sqs.Config, brokertesting.BrokerTester) {
	file, err := os.Open("../../fixtures/config.json")
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	config, err := broker.NewConfig(file)
	Expect(err).ToNot(HaveOccurred())

	config.API.Locket.SkipVerify = true
	config.API.Locket.Address = mockLocket.ListenAddress
	config.API.Locket.CACertFile = path.Join(locketFixtures.Filepath, "locket-server.cert.pem")
	config.API.Locket.ClientCertFile = path.Join(locketFixtures.Filepath, "locket-client.cert.pem")
	config.API.Locket.ClientKeyFile = path.Join(locketFixtures.Filepath, "locket-client.key.pem")

	sqsClientConfig, err := sqs.NewSQSClientConfig(config.Provider)
	Expect(err).ToNot(HaveOccurred())

	logger := lager.NewLogger("sqs-service-broker-test")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, config.API.LagerLogLevel))

	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(sqsClientConfig.AWSRegion)}))
	sqsClient := sqs.NewSQSClient(sqsClientConfig, aws_sqs.New(sess), cfn.New(sess), iam.New(sess), logger, context.Background())

	sqsProvider := provider.NewSQSProvider(sqsClient, "test")

	serviceBroker, err := broker.New(config, sqsProvider, logger)
	Expect(err).ToNot(HaveOccurred())
	brokerAPI := broker.NewAPI(serviceBroker, logger, config)

	return sqsClientConfig, brokertesting.New(brokerapi.BrokerCredentials{
		Username: "username",
		Password: "password",
	}, brokerAPI)
}
