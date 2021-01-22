package queue

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

type Mock4handleMessageAWSSession struct {
	TimesCalledDeleteMessage int
	TimesCalledSendMessage   int
	Receipt                  string
	ShouldDeleteMessageFail  bool
	DeleteMessageFailError   string
	DeleteMessageTimeout     int64
	LastNextDelayRetry       *string
	LastBodySent             *string
}

func (a *Mock4handleMessageAWSSession) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.TimesCalledSendMessage++
	a.LastNextDelayRetry = input.MessageAttributes["NextDelayRetry"].StringValue
	a.LastBodySent = input.MessageBody
	return nil, nil
}

func (a *Mock4handleMessageAWSSession) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return nil, nil
}

func (a *Mock4handleMessageAWSSession) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	a.TimesCalledDeleteMessage++
	a.Receipt = *input.ReceiptHandle

	if a.DeleteMessageTimeout > 0 {
		time.Sleep(time.Duration(a.DeleteMessageTimeout) * time.Second)
	}

	if a.ShouldDeleteMessageFail {
		return nil, errors.New(a.DeleteMessageFailError)
	}

	return nil, nil
}

/*
	Case 1: The handler receives a hander-function and a message.
	First it tries to delete it, then if OK it sends the message to the handler
*/
func Test_handleMessage_once(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	finish := make(chan bool)
	expectedReceipt := "a receipt handle"
	expectedMessage := "a message"

	handler := func(msg string) error {
		finish <- true
		assert.Equal(t, expectedMessage, msg)
		return nil
	}

	queue := queueSQS{
		SQS: session,
		URL: "",
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	err := queue.handleMessage(handler, &msg)

	<-finish

	assert.Nil(t, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage) // once called
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 2: The handler receives a hander-function and a message.
	First it tries to delete it, then if OK it sends the message to the handler
	But, if handler returns error then resend the message. First error
*/
func Test_handleMessage_resend(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	finish := make(chan bool)
	expectedReceipt := "a receipt handle"
	expectedMessage := "a message"

	handler := func(msg string) error {
		go func() {
			finish <- true
		}()
		assert.Equal(t, expectedMessage, msg)
		return errors.New("intentional error") // this triggers the resend process
	}

	queue := queueSQS{
		SQS: session,
		URL: "",
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	err := queue.handleMessage(handler, &msg)

	<-finish

	assert.Nil(t, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 1, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 3: The handler receives a hander-function and a message.
	First it tries to delete it, but it fails. Then returns an error
*/
func Test_handleMessage_deletion_error(t *testing.T) {
	session := &Mock4handleMessageAWSSession{
		ShouldDeleteMessageFail: true,
		DeleteMessageFailError:  "an intentional error",
	}

	expectedReceipt := "a receipt handle"
	expectedMessage := "a message" // it's not important, but it's OK to define it

	handler := func(msg string) error {
		return nil
	}

	queue := queueSQS{
		SQS: session,
		URL: "",
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	err := queue.handleMessage(handler, &msg)

	assert.NotNil(t, err)
	assert.Equal(t, session.DeleteMessageFailError, err.Error())

	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 0, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 4: The handler receives a hander-function and a message.
	First it tries to delete it, but it fails. Then returns an error
*/
func Test_handleMessage_deletion_timeout(t *testing.T) {
	session := &Mock4handleMessageAWSSession{
		DeleteMessageTimeout: 2,
	}

	expectedReceipt := "a receipt handle"
	expectedMessage := "a message" // it's not important, but it's OK to define it

	handler := func(msg string) error {
		return nil
	}

	queue := queueSQS{
		SQS:            session,
		URL:            "",
		TimeoutSeconds: 1,
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	err := queue.handleMessage(handler, &msg)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorDeleteMessageTimeout, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 0, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

func Test_resendMessage_OK(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	queue := queueSQS{
		SQS:            session,
		URL:            "",
		TimeoutSeconds: 1,
	}

	currentRetry := 10
	expectedBody := "something to send"
	expectedRetry := fmt.Sprintf("%d", currentRetry+queue.TimeoutSeconds)

	msg := sqs.Message{}
	msg.Body = aws.String(expectedBody)
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("number"),
			StringValue: aws.String(fmt.Sprintf("%d", currentRetry)),
		},
	}

	queue.resendMessage(&msg)
	assert.Equal(t, expectedRetry, *session.LastNextDelayRetry)
	assert.Equal(t, expectedBody, *session.LastBodySent)
}
