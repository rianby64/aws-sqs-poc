package queue

import (
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
)

type Mock4ReceiveMessageAWSSession struct {
	CalledReceiveMessage bool
	Locker               sync.Mutex
	Waiter               sync.WaitGroup
}

func (a *Mock4ReceiveMessageAWSSession) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return nil, nil
}

func (a *Mock4ReceiveMessageAWSSession) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	a.CalledReceiveMessage = true
	a.Locker.Lock()
	a.Waiter.Done()

	response := sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{},
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
	/*
		session := &Mock4ReceiveMessageAWSSession{}
		session.Waiter.Add(1) // wait for the call of ReceiveMessage

		handler := func(str string) error {
			return nil
		}

		queue := queueSQS{
			SQS: session,
		}

		go queue.listen(handler)
		session.Waiter.Wait()

		assert.True(t, session.CalledReceiveMessage)
	*/
}
