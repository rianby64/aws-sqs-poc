package queue

import (
	"errors"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

type MockAWSSession1 struct {
	Locker sync.Mutex
	input  *sqs.SendMessageInput
}

func (a *MockAWSSession1) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.input = input
	return nil, nil
}

func (a *MockAWSSession1) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	if a.input == nil {
		return nil, errors.New("nothing sent")
	}

	response := sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				MessageAttributes: map[string]*sqs.MessageAttributeValue{
					"NextDelayRetry": {
						DataType:    aws.String("Number"),
						StringValue: aws.String("10"),
					},
				},
				Body: a.input.MessageBody,
			},
		},
	}

	return &response, nil
}

func (a *MockAWSSession1) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

func Test_Put_once_and_Handle(t *testing.T) {
	finish := make(chan bool)
	expected := "my string"

	handlerMock1 := func(msg string) error {
		assert.Equal(t, expected, msg)
		finish <- true
		return nil
	}

	mysession := MockAWSSession1{}
	queue := NewSQSQueue(&mysession)

	queue.Register("", handlerMock1)
	queue.Put(expected, 0)

	<-finish
}
