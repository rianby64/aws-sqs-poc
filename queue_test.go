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
	Locker          sync.Mutex
	input           *sqs.SendMessageInput
	MethodAttribute *sqs.MessageAttributeValue
}

func (a *MockAWSSession1) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.input = input
	return nil, nil
}

func (a *MockAWSSession1) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
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

	handlerMock1 := func(msg interface{}) error {
		assert.Equal(t, expected, msg)
		finish <- true
		return nil
	}

	mysession := MockAWSSession1{
		Locker: sync.Mutex{},
	}

	queue := NewSQSQueue(&mysession, "")

	queue.Register("", handlerMock1)
	go queue.Listen()

	if err := queue.PutString("", expected, 0); err != nil {
		t.Error(err)
	}

	<-finish
}

func Test_PutJSON_once_and_Handle(t *testing.T) {
	finish := make(chan bool)
	expected := map[string]interface{}{
		"key": "my string",
	}

	handlerMock1 := func(msg interface{}) error {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			err := errors.New("incorrect message")
			t.Error(err)
			t.Fail()

			return err
		}

		assert.Equal(t, expected["key"], msgMap["key"])
		finish <- true

		return nil
	}

	mysession := MockAWSSession1{
		MethodAttribute: &sqs.MessageAttributeValue{
			DataType:    aws.String("string"),
			StringValue: aws.String("method_name"),
		},
	}
	queue := NewSQSQueue(&mysession, "")

	queue.Register("method_name", handlerMock1)
	go queue.Listen()

	if err := queue.PutJSON("method_name", expected, 0); err != nil {
		t.Error(err)
	}

	<-finish
}
