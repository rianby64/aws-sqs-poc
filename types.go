package queue

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
)

const (
	maxNumberOfMessages             = 10
	waitTimeSeconds                 = 10
	maxNumberOfRetries              = 5
	retrySecondsToListen            = 5
	timeoutSecondsDefault           = 5
	nextDelayIncreaseSecondsDefault = 1
)

// These are the error definitions
var (
	ErrorMethodAttrNil        = errors.New("methodAttr value is nil")
	ErrorDeleteMessageTimeout = errors.New("timeout processing message from queue")
	ErrorHandlerNotFound      = errors.New("handler not found in the register map")
	ErrorMessageIDNotFound    = errors.New("response has no messageID value")
	ErrorRequestMaxRetries    = errors.New("drop request from Queue as it failed maxNumberOfRetries times")
)

// iSQSSession represents the interface to connect to a Queue
type iSQSSession interface {
	SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
	ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
}

// queueSQS - A queue backed by SQS.
type queueSQS struct {
	SQS                      iSQSSession
	URL                      string
	TimeoutSeconds           int
	NextDelayIncreaseSeconds int64
	handlerMap               map[string]MessageHandler
	msgIDerrs                map[string]int
	thens                    map[string][]MessageHandler
}

// MessageHandler receives from the queue the message. Use Register to define the handler
type MessageHandler func(msg interface{}) error

type msgJSON struct {
	Msg interface{} `json:"msg"`
}

// SQSQueue defines the special SQS-Queue that accepts handlers via Register
type SQSQueue interface {
	PutString(method, msg string, delaySeconds int64) *sqsResponseThenable
	PutJSON(method string, msg interface{}, delaySeconds int64) *sqsResponseThenable
	Register(name string, method MessageHandler)
	Listen()
}

type sqsResponseThenable struct {
	queue     *queueSQS
	messageID string
	Error     error
}
