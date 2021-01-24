package queue

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

type Mock4ReceiveMessageAWSSession struct {
	CalledReceiveMessage    bool
	Locker                  sync.Mutex
	Waiter                  sync.WaitGroup
	ReceiveMessageResponses []*sqs.Message
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

	return &response, nil
}

func (a *Mock4ReceiveMessageAWSSession) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

/*
	Case 1: queue.listen(handler) calls ReceiveMessage at least once
*/
func Test_listen_calls_ReceiveMessage(t *testing.T) {
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{},
	}
	session.Waiter.Add(1) // wait for the call of ReceiveMessage

	handler := func(str string) error {
		return nil
	}

	queue := queueSQS{
		SQS:        session,
		handlerMap: map[string]MessageHandler{},
	}

	queue.Register("", handler)

	go queue.listen()
	session.Waiter.Wait()

	assert.True(t, session.CalledReceiveMessage)
}

/*
	Case 1: queue.listen(handler) calls ReceiveMessage at least once
*/
func Test_listen_calls_ReceiveMessage_with_two_responses(t *testing.T) {
	msg1 := "message 1"
	msg2 := "message 2"
	session := &Mock4ReceiveMessageAWSSession{
		ReceiveMessageResponses: []*sqs.Message{
			{
				Body: aws.String(msg1),
			},
			{
				Body: aws.String(msg2),
			},
		},
	}
	session.Waiter.Add(3) // wait for the call of ReceiveMessage and the handler twice

	i := 0
	handler := func(str string) error {
		if msg1 == str || msg2 == str {
			i++
			session.Waiter.Done()
		}

		return nil
	}

	queue := queueSQS{
		SQS:        session,
		handlerMap: map[string]MessageHandler{},
	}

	queue.Register("", handler)

	go queue.listen()
	session.Waiter.Wait()

	assert.True(t, session.CalledReceiveMessage)
	assert.Equal(t, 2, i)
}
