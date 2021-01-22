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
	nextDelay := delaySeconds + nextDelayIncreaseSecondsDefault
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

// NewSQSQueue jajaja
func NewSQSQueue(sqssession mySQSSession, handler MessageHandler) SQSQueue {
	queue := queueSQS{
		SQS:            sqssession,
		URL:            "https://sqs.us-east-1.amazonaws.com/490043543248/my-queue-test",
		TimeoutSeconds: 1,
	}

	go func() {
		for {
			if err := queue.listen(handler); err != nil {
				log.Error(err, "terminated, retry to listen... wait")
			}

			time.Sleep(retrySecondsToListen * time.Second)
		}
	}()

	return &queue
}
