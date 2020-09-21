package sqs

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

const (
	ParamDelaySeconds                  = "DelaySeconds"
	ParamMaximumMessageSize            = "MaximumMessageSize"
	ParamMessageRetentionPeriod        = "MessageRetentionPeriod"
	ParamReceiveMessageWaitTimeSeconds = "ReceiveMessageWaitTimeSeconds"
	ParamRedriveMaxReceiveCount        = "RedriveMaxReceiveCount"
	ParamVisibilityTimeout             = "VisibilityTimeout"
)

const (
	ConditionShouldNotUseDLQ = "ShouldNotUseDLQ"
)

const (
	ResourcePrimaryQueue   = "PrimaryQueue"
	ResourceSecondaryQueue = "SecondaryQueue"
)

const (
	OutputPrimaryQueueURL   = "PrimaryQueueURL"
	OutputPrimaryQueueARN   = "PrimaryQueueARN"
	OutputSecondaryQueueURL = "SecondaryQueueURL"
	OutputSecondaryQueueARN = "SecondaryQueueARN"
)

const (
	ExtFIFO     = ".fifo"
	ExtStandard = ""
)

// A QueueTemplateBuilder is responsible for building the
// CloudFormation YAML template.  You can configure it to control
// exactly how the template is built.
type QueueTemplateBuilder struct {
	QueueName string
	FIFOQueue bool
	Tags      map[string]string
}

// PrimaryQueueName builds the name for the primary queue
func (params *QueueTemplateBuilder) PrimaryQueueName() string {
	return fmt.Sprintf("%s-pri%s", params.QueueName, params.ext())
}

// SecondaryQueueName builds the name for the secondary queue
func (params *QueueTemplateBuilder) SecondaryQueueName() string {
	return fmt.Sprintf("%s-sec%s", params.QueueName, params.ext())
}

// ext returns the suffix for the queue names. this is important because
// FIFO queues require a sepecific suffix
func (params *QueueTemplateBuilder) ext() string {
	if params.FIFOQueue {
		return ExtFIFO
	} else {
		return ExtStandard
	}
}

// Build returns a cloudformation Template for provisioning an SQS
// queue
func (params *QueueTemplateBuilder) Build() (string, error) {
	t, err := template.New("queue-template").Parse(queueTemplateFormat)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)

	err = t.Execute(buf, params)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// QueueParams is the set of actual CloudFormation template
// parameters that can be passed to the stack.  If it comes from user
// configuration such as:
//     cf create-service foo -c '{"my-config": "bar"}'`
// then it should be in QueueParams so that CloudFormation
// can keep track of its value across updates.
type QueueParams struct {
	// DelaySeconds The time in seconds for which the delivery of all messages
	// in the queue is delayed. You can specify an integer value of 0 to 900
	// (15 minutes).
	DelaySeconds *int `json:"delay_seconds,omitempty"`
	// MaximumMessageSize is the limit of how many bytes that a message can
	// contain before Amazon SQS rejects it. You can specify an integer value
	// from 1,024 bytes (1 KiB) to 262,144 bytes (256 KiB). The default value
	// is 262,144 (256 KiB).
	MaximumMessageSize *int `json:"maximum_message_size,omitempty"`
	// MessageRetentionPeriod The number of seconds
	// that Amazon SQS retains a message. You can
	// specify an integer value from 60 seconds (1
	// minute) to 1,209,600 seconds (14 days). The
	// default value is 345,600 seconds (4 days).
	MessageRetentionPeriod *int `json:"message_retention_period,omitempty"`
	// ReceiveMessageWaitTimeSeconds Specifies the
	// duration, in seconds, that the ReceiveMessage
	// action call waits until a message is in the
	// queue in order to include it in the response,
	// rather than returning an empty response if a
	// message isn't yet available. You can specify an
	// integer from 1 to 20. Short polling is used as
	// the default or when you specify 0 for this
	// property.
	ReceiveMessageWaitTimeSeconds *int `json:"receive_message_wait_time_seconds,omitempty"`
	// RedriveMaxReceiveCount  The number of times a
	// message is delivered to the source queue before
	// being moved to the dead-letter queue.
	RedriveMaxReceiveCount *int `json:"redrive_max_receive_count,omitempty"`
	// VisibilityTimeout The length of time during
	// which a message will be unavailable after a
	// message is delivered from the queue. This blocks
	// other components from receiving the same message
	// and gives the initial component time to process
	// and delete the message from the queue.
	// Values must be from 0 to 43,200 seconds (12 hours). If you don't specify a value, AWS CloudFormation uses the default value of 30 seconds.
	VisibilityTimeout *int `json:"visibility_timeout,omitempty"`
}

// CreateParams returns a set of cloudformation.Parameter suitable for
// passing to CreateStackWithContext().
func (params *QueueParams) CreateParams() []*cloudformation.Parameter {
	stackParams := []*cloudformation.Parameter{}
	if params.DelaySeconds != nil {
		stackParams = append(stackParams, mkParameter(ParamDelaySeconds, *params.DelaySeconds))
	}
	if params.MaximumMessageSize != nil {
		stackParams = append(stackParams, mkParameter(ParamMaximumMessageSize, *params.MaximumMessageSize))
	}
	if params.MessageRetentionPeriod != nil {
		stackParams = append(stackParams, mkParameter(ParamMessageRetentionPeriod, *params.MessageRetentionPeriod))
	}
	if params.ReceiveMessageWaitTimeSeconds != nil {
		stackParams = append(stackParams, mkParameter(ParamReceiveMessageWaitTimeSeconds, *params.ReceiveMessageWaitTimeSeconds))
	}
	if params.RedriveMaxReceiveCount != nil {
		stackParams = append(stackParams, mkParameter(ParamRedriveMaxReceiveCount, *params.RedriveMaxReceiveCount))
	}
	if params.VisibilityTimeout != nil {
		stackParams = append(stackParams, mkParameter(ParamVisibilityTimeout, *params.VisibilityTimeout))
	}
	return stackParams
}

func mkParameter(name string, value int) *cloudformation.Parameter {
	return &cloudformation.Parameter{
		ParameterKey:   aws.String(name),
		ParameterValue: aws.String(strconv.Itoa(value)),
	}
}

// UpdateParams returns a set of cloudformation.Parameter suitable for
// passing to UpdateStackWithContext().  In particular, if a parameter
// is nil, UpdateParams will return a cloudformation.Parameter with
// UsePreviousValue set to true.
func (params *QueueParams) UpdateParams() []*cloudformation.Parameter {
	return []*cloudformation.Parameter{
		mkOptionalParameter(ParamDelaySeconds, params.DelaySeconds),
		mkOptionalParameter(ParamMaximumMessageSize, params.MaximumMessageSize),
		mkOptionalParameter(ParamMessageRetentionPeriod, params.MessageRetentionPeriod),
		mkOptionalParameter(ParamReceiveMessageWaitTimeSeconds, params.ReceiveMessageWaitTimeSeconds),
		mkOptionalParameter(ParamRedriveMaxReceiveCount, params.RedriveMaxReceiveCount),
		mkOptionalParameter(ParamVisibilityTimeout, params.VisibilityTimeout),
	}
}

func mkOptionalParameter(name string, value *int) *cloudformation.Parameter {
	if value == nil {
		return &cloudformation.Parameter{
			ParameterKey:     aws.String(name),
			UsePreviousValue: aws.Bool(true),
		}
	} else {
		return &cloudformation.Parameter{
			ParameterKey:   aws.String(name),
			ParameterValue: aws.String(strconv.Itoa(*value)),
		}
	}
}
