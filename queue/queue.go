package queue

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
)

// Put sends something to the queue
func (q *queueSQS) Put(msg string, delaySeconds int64) error {
	if q.NextDelayIncreaseSeconds == 0 {
		q.NextDelayIncreaseSeconds = nextDelayIncreaseSecondsDefault
	}

	nextDelay := delaySeconds + q.NextDelayIncreaseSeconds
	messageAttributes := map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("Number"),
			StringValue: aws.String(fmt.Sprintf("%d", nextDelay)),
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

// Register
func (q *queueSQS) Register(name string, method MessageHandler) {
	q.handlerMap[name] = method
}

// NewSQSQueue jajaja
func NewSQSQueue(sqssession mySQSSession, URL string) SQSQueue {
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
