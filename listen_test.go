package queue

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type Mock4ReceiveMessageAWSSession struct {
	CalledReceiveMessage    bool
	Locker                  sync.Mutex
	Waiter                  sync.WaitGroup
	ReceiveMessageResponses []*sqs.Message
	ReceiveMessageError     error
	DeleteMessageError      error
}

func (a *Mock4ReceiveMessageAWSSession) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return nil, nil
}

func (a *Mock4ReceiveMessageAWSSession) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	a.CalledReceiveMessage = true
	a.Locker.Lock()
	a.Waiter.Done()

	response := sqs.ReceiveMessageOutput{
		Messages: a.ReceiveMessageResponses,
	}

	return &response, a.ReceiveMessageError
}

func (a *Mock4ReceiveMessageAWSSession) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, a.DeleteMessageError
}

/*
	Case 1: queue.listen(handler) calls ReceiveMessage at least once
*/
func Test_listen_calls_ReceiveMessage(t *testing.T) {
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{},
	}
	session.Waiter.Add(1) // wait for the call of ReceiveMessage

	handler := func(str interface{}) error {
		return nil
	}

	queue := queueSQS{
		SQS:        session,
		handlerMap: map[string]MessageHandler{},
	}

	queue.Register("", handler)

	go func() {
		if err := queue.listen(); err != nil {
			t.Error(err)
		}
	}()

	session.Waiter.Wait()

	assert.True(t, session.CalledReceiveMessage)
}

/*
	Case 2: queue.listen(handler) calls ReceiveMessage at least once
*/
func Test_listen_calls_ReceiveMessage_with_two_responses(t *testing.T) {
	msg1 := "message 1"
	msg2 := "message 2"
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{
			{
				Body:      aws.String(msg1),
				MessageId: aws.String("message id 1"),
			},
			{
				Body:      aws.String(msg2),
				MessageId: aws.String("message id 2"),
			},
		},
	}
	session.Waiter.Add(3) // wait for the call of ReceiveMessage and the handler twice

	i := 0
	handler := func(str interface{}) error {
		if msg1 == str || msg2 == str {
			i++
			session.Waiter.Done()
		}

		return nil
	}

	queue := queueSQS{
		SQS:        session,
		handlerMap: map[string]MessageHandler{},
		msgIDerrs:  map[string]int{},
	}

	queue.Register("", handler)

	go func() {
		if err := queue.listen(); err != nil {
			t.Error(err)
		}
	}()

	session.Waiter.Wait()

	assert.True(t, session.CalledReceiveMessage)
	assert.Equal(t, 2, i)
}

/*
	Case 3: queue.listen(handler) ReceiveMessage returns an error, so listens returns it too
*/
func Test_listen_calls_ReceiveMessage_error_listen_error_too(t *testing.T) {
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{},
		ReceiveMessageError:     errors.New("an intentional error"),
	}
	session.Waiter.Add(1)

	queue := queueSQS{
		SQS:        session,
		handlerMap: map[string]MessageHandler{},
	}

	err := queue.listen()
	session.Waiter.Wait()

	assert.NotNil(t, err)
	assert.Equal(t, "SQS.ReceiveMessage error: an intentional error", err.Error())
}

/*
	Case 4: queue.matchHandler matches nothing
*/
func Test_matchHandler_not_found(t *testing.T) {
	queue := queueSQS{
		handlerMap: map[string]MessageHandler{},
	}

	msg := &sqs.Message{}
	handler, err := queue.matchHandler(msg)

	assert.Nil(t, handler)
	assert.NotNil(t, err)
	assert.Equal(t, ErrorHandlerNotFound, err)
}

/*
	Case 5: queue.matchHandler matches one named handler
*/
func Test_matchHandler_match_named_handler(t *testing.T) {
	method := "named_handler"
	called := false
	namedHandler := func(msg interface{}) error {
		called = true
		return nil
	}

	queue := queueSQS{
		handlerMap: map[string]MessageHandler{
			method: namedHandler,
		},
	}

	msg := &sqs.Message{}
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"Method": {
			DataType:    aws.String("string"),
			StringValue: aws.String(method),
		},
	}
	handler, err := queue.matchHandler(msg)

	assert.NotNil(t, handler)
	assert.Nil(t, err)
	assert.Nil(t, handler(""))
	assert.True(t, true, called)
}

/*
	Case 6: listen fails because of matchHandler error
*/
func Test_listen_matchHandler_err(t *testing.T) {
	msg := &sqs.Message{}
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"Method": {
			DataType:    aws.String("string"),
			StringValue: aws.String("unmatchable_method"),
		},
	}
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{msg},
	}

	session.Waiter.Add(1)

	queue := queueSQS{
		SQS: session,
	}

	err := queue.listen()
	session.Waiter.Wait()

	assert.NotNil(t, err)
	assert.Equal(t, ErrorHandlerNotFound, err)
}

/*
	Case 7: listen fails because of handleMessage by DeleteMessage error
*/
func Test_listen_handleMessage_err(t *testing.T) {
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{{}},
		DeleteMessageError:      errors.New("intentional error"),
	}

	session.Waiter.Add(1)

	queue := queueSQS{
		SQS: session,
	}
	queue.Register("", func(msg interface{}) error {
		return nil
	})

	err := queue.listen()
	session.Waiter.Wait()

	assert.NotNil(t, err)
	assert.Equal(t, "handling queue message: intentional error", err.Error())
}
