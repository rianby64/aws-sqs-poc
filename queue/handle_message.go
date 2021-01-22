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
	params := sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.URL),
		ReceiptHandle: m.ReceiptHandle,
	}
	releaseWait := make(chan bool, 1)
	timeoutSeconds := q.TimeoutSeconds

	if timeoutSeconds == 0 {
		timeoutSeconds = timeoutSecondsDefault
	}

	go func() {
		if _, err = q.SQS.DeleteMessage(&params); err != nil {
			releaseWait <- true // this is... a thing

			/*
				Let's consider the case when deletion returns with error...
				this means that probably the message is still in the queue,
				so in another round it will be processed.
			*/

			return
		}

		releaseWait <- true

		/*
			What about if the message was deleted? Then the handler takes the
			responsability to process the message and if it returns an error
			then resend it. Any further error only can be logged.
		*/

		if err := fn(aws.StringValue(m.Body)); err != nil {
			log.Errorf("running handler error: %v", err)

			if err := q.resendMessage(m); err != nil {
				log.Errorf("resending messange to queue: %v", err)
			}

			/*
				In conclusion. If you put releaseWait at the end of this
				function, surely it may end up in flooding the queue
			*/
		}
	}()

	select {
	case <-releaseWait:
		log.Info("Processed message from queue")
		return err
	case <-time.After(time.Second * time.Duration(timeoutSeconds)):
		return ErrorDeleteMessageTimeout
	}
}

func (q *queueSQS) resendMessage(m *sqs.Message) error {
	delayRetry := int64(0)
	messageAttributes := m.MessageAttributes
	if delayRetryAttr, ok := messageAttributes["NextDelayRetry"]; ok && delayRetryAttr.StringValue != nil {
		delayRetryValue, err := strconv.ParseInt(*delayRetryAttr.StringValue, 10, 64)

		if err != nil {
			return errors.Wrap(err, "Incorrect value of NextDelayRetry")
		}

		delayRetry = delayRetryValue
	}

	return q.Put(*m.Body, delayRetry)
}
