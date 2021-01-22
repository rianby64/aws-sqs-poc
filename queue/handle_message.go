package queue

import (
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// handleMessage - Performs work on a message with configured timeout.
func (q *queueSQS) handleMessage(fn MessageHandler, m *sqs.Message) error {
	releaseWait := make(chan bool, 1)
	params := sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.URL),
		ReceiptHandle: m.ReceiptHandle,
	}

	go func() {
		defer func() {
			releaseWait <- true
		}()

		if _, err := q.SQS.DeleteMessage(&params); err != nil {
			log.Errorf("deleting message from queue: %v", err)
			return
		}

		if err := fn(aws.StringValue(m.Body)); err != nil {
			q.resendMessage(m)
			log.Errorf("running queue handler error: %v", err)
		}
	}()

	select {
	case <-releaseWait:
		log.Info("Processed message from queue")
		return nil
	case <-time.After(time.Second * time.Duration(q.TimeoutSeconds)):
		return errors.New("Timeout processing message from queue")
	}
}

func (q *queueSQS) resendMessage(m *sqs.Message) {
	delayRetry := int64(0)
	messageAttributes := m.MessageAttributes
	if delayRetryAttr, ok := messageAttributes["NextDelayRetry"]; ok && delayRetryAttr.StringValue != nil {
		delayRetryValue, err := strconv.ParseInt(*delayRetryAttr.StringValue, 10, 64)
		if err != nil {
			log.Error(errors.Wrap(err, "NextDelayRetry incorrect"))

			return
		}
		delayRetry = delayRetryValue
	}

	q.Put(*m.Body, delayRetry)
}
