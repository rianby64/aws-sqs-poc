package queue

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type MockAWSSessionThen struct {
	input *sqs.SendMessageInput
}

func (a *MockAWSSessionThen) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.input = input
	return &sqs.SendMessageOutput{
		MessageId:        aws.String("messageID"),
		MD5OfMessageBody: aws.String("messageID"),
	}, nil
}

func (a *MockAWSSessionThen) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	if a.input == nil {
		return nil, errors.New("nothing sent")
	}

	messageAttributes := map[string]*sqs.MessageAttributeValue{}

	if value, ok := a.input.MessageAttributes["NextDelayRetry"]; ok && value != nil {
		messageAttributes["NextDelayRetry"] = value
	}

	if value, ok := a.input.MessageAttributes["Method"]; ok && value != nil {
		messageAttributes["Method"] = value
	}

	response := sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				MessageAttributes: messageAttributes,
				Body:              a.input.MessageBody,
				MessageId:         aws.String("messageID"),
				MD5OfBody:         aws.String("messageID"),
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

	expectedMsg := "send this and take it from then"
	q.Register("method", func(msg interface{}) error {
		return nil
	})

	go q.Listen()

	finish := make(chan bool)
	actualMsg := ""
	q.PutJSON("method", expectedMsg, 0).Then(func(msg interface{}) error {
		actualMsg = msg.(string)
		finish <- true
		return nil
	})

	<-finish

	assert.Equal(t, expectedMsg, actualMsg)

}
