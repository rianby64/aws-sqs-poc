package queue

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

type MockAWSSessionThen struct {
	Locker          sync.Mutex
	input           *sqs.SendMessageInput
	MethodAttribute *sqs.MessageAttributeValue
}

func (a *MockAWSSessionThen) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.input = input
	return &sqs.SendMessageOutput{
		MessageId: aws.String("messageID"),
	}, nil
}

func (a *MockAWSSessionThen) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	if a.input == nil {
		return nil, errors.New("nothing sent")
	}
	a.Locker.Lock()

	messageAttributes := map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("Number"),
			StringValue: aws.String("10"),
		},
	}

	if a.MethodAttribute != nil {
		messageAttributes["Method"] = a.MethodAttribute
	}

	response := sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				MessageAttributes: messageAttributes,
				Body:              a.input.MessageBody,
				MessageId:         aws.String("messageID"),
			},
		},
	}

	return &response, nil
}

func (a *MockAWSSessionThen) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

func Test_Then_ok(t *testing.T) {
	s := &MockAWSSessionThen{}
	q := NewSQSQueue(s, "")

	msg := "send this and take it from then"
	q.PutJSON("method", msg, 0).Then(func(msg interface{}) error {
		return nil
	})
}
