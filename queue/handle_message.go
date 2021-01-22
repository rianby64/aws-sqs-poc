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
func (q *queueSQS) handleMessage(fn MessageHandler, m *sqs.Message) (err error) {
	timeoutSeconds := q.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = timeoutSecondsDefault
	}
	releaseWait := make(chan bool, 1)
	params := sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.URL),
		ReceiptHandle: m.ReceiptHandle,
	}

	go func() {
		if _, err = q.SQS.DeleteMessage(&params); err != nil {
			log.Errorf("deleting message from queue: %v", err)
			releaseWait <- true

			return
		}

		releaseWait <- true

		if err := fn(aws.StringValue(m.Body)); err != nil {
			log.Errorf("running handler error: %v", err)

			if err := q.resendMessage(m); err != nil {
				log.Errorf("resending messange to queue: %v", err)
			}
		}
	}()

	select {
	case <-releaseWait:
		log.Info("Processed message from queue")
		return err
	case <-time.After(time.Second * time.Duration(timeoutSeconds)):
		return errors.New("Timeout processing message from queue")
	}
}

func (q *queueSQS) resendMessage(m *sqs.Message) error {
	delayRetry := int64(0)
	messageAttributes := m.MessageAttributes
	if delayRetryAttr, ok := messageAttributes["NextDelayRetry"]; ok && delayRetryAttr.StringValue != nil {
		delayRetryValue, err := strconv.ParseInt(*delayRetryAttr.StringValue, 10, 64)

		if err != nil {
			return errors.Wrap(err, "NextDelayRetry incorrect")
		}

		delayRetry = delayRetryValue
	}

	return q.Put(*m.Body, delayRetry)
}
