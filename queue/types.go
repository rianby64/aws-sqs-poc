package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

const (
	maxNumberOfMessages             = 10
	waitTimeSeconds                 = 10
	retrySecondsToListen            = 5
	timeoutSecondsDefault           = 5
	nextDelayIncreaseSecondsDefault = 1 /// TODO: use this variable correctly
)

// These are the error definitions
var (
	ErrorDeleteMessageTimeout = errors.New("Timeout processing message from queue")
)

// mySQSSession jajaja
type mySQSSession interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

// queueSQS - A queue backed by SQS.
type queueSQS struct {
	SQS                      mySQSSession
	URL                      string
	TimeoutSeconds           int
	NextDelayIncreaseSeconds int64
	handlerMap               map[string]MessageHandler
}

// MessageHandler eje!!
type MessageHandler func(msg string) error

// SQSQueue ajajaja
type SQSQueue interface {
	Put(msg string, delaySeconds int64) error
	Register(name string, method MessageHandler)
}
