.PHONY: unit
unit:
	ginkgo $(COMMAND) -r --skipPackage=testing/integration $(PACKAGE)

.PHONY: test
test:
	go test -mod=vendor ./...

.PHONY: generate
generate:
	GOFLAGS=-mod=vendor counterfeiter -o sqs/fakes/fake_cfn_api.go vendor/github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface/ CloudFormationAPI
	go generate ./...
