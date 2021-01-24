package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// PutString sends an string to the queue
func (q *queueSQS) PutString(method, msg string, delaySeconds int64) error {
	if q.NextDelayIncreaseSeconds == 0 {
		q.NextDelayIncreaseSeconds = nextDelayIncreaseSecondsDefault
	}

	nextDelay := delaySeconds + q.NextDelayIncreaseSeconds
	messageAttributes := map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("Number"),
			StringValue: aws.String(fmt.Sprintf("%d", nextDelay)),
		},
		"Method": {
			DataType:    aws.String("string"),
			StringValue: aws.String(method),
		},
	}

	params := sqs.SendMessageInput{
		QueueUrl:          aws.String(q.URL),
		MessageBody:       aws.String(msg),
		DelaySeconds:      aws.Int64(delaySeconds),
		MessageAttributes: messageAttributes,
	}

	if _, err := q.SQS.SendMessage(&params); err != nil {
		return err
	}

	return nil
}

// PutString sends a JSON to the queue
func (q *queueSQS) PutJSON(method string, msg interface{}, delaySeconds int64) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "PutJSON error")
	}

	return q.PutString(method, string(msgBytes), delaySeconds)
}

// Register
func (q *queueSQS) Register(name string, method MessageHandler) {
	if q.handlerMap == nil {
		q.handlerMap = map[string]MessageHandler{}
	}

	q.handlerMap[name] = method
}

// NewSQSQueue jajaja
func NewSQSQueue(sqssession iSQSSession, URL string) SQSQueue {
	queue := queueSQS{
		SQS:                      sqssession,
		URL:                      URL,
		TimeoutSeconds:           timeoutSecondsDefault,
		NextDelayIncreaseSeconds: nextDelayIncreaseSecondsDefault,
		handlerMap:               map[string]MessageHandler{},
	}

	go func() {
		for {
			if err := queue.listen(); err != nil {
				log.Error(err, "terminated, retry to listen... wait")
			}

			time.Sleep(retrySecondsToListen * time.Second)
		}
	}()

	return &queue
}
