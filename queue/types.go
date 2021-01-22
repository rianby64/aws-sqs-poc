package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	maxNumberOfMessages      = 10
	waitTimeSeconds          = 10
	nextDelayIncreaseSeconds = 1
	retrySecondsToListen     = 5
	timeoutSecondsDefault    = 5
)

// mySQSSession jajaja
type mySQSSession interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

// queueSQS - A queue backed by SQS.
type queueSQS struct {
	SQS            mySQSSession
	URL            string
	TimeoutSeconds int
}

// MessageHandler eje!!
type MessageHandler func(msg string) error

// SQSQueue ajajaja
type SQSQueue interface {
	Put(msg string, delaySeconds int64) error
}
