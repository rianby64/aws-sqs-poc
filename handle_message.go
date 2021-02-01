package queue

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"

	// nolint: depguard
	log "github.com/sirupsen/logrus"
)

// handleMessage - Performs work on a message with configured timeout.
func (q *queueSQS) handleMessage(fn MessageHandler, m *sqs.Message) (err error) {
	params := sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.URL),
		ReceiptHandle: m.ReceiptHandle,
	}
	releaseWaitErr := make(chan error, 1)
	timeoutSeconds := q.TimeoutSeconds

	if timeoutSeconds == 0 {
		timeoutSeconds = timeoutSecondsDefault
	}

	go func() {
		if _, err = q.SQS.DeleteMessage(&params); err != nil {
			releaseWaitErr <- err // this is... a thing

			/*
				Let's consider the case when deletion returns with error...
				this means that probably the message is still in the queue,
				so in another round it will be processed.
			*/

			return
		}

		msgID := ""
		if msgID, err = q.prepareMessageID(m); err != nil {
			releaseWaitErr <- err

			return
		}

		releaseWaitErr <- nil

		/*
			What about if the message was deleted? Then the handler takes the
			responsibility to process the message and if it returns an error
			then resend it. Any further error only can be logged.
		*/

		msg := q.unmarshal(aws.StringValue(m.Body))
		if err2 := fn(msg); err2 != nil {
			log.Errorf("running handler error: %v", err2)
			q.msgIDerrs[msgID]++

			if err2 := q.resendMessage(m); err2 != nil {
				log.Errorf("resending messange to queue: %v", err2)
			}

			/*
				In conclusion. If you put releaseWait at the end of this
				function, surely it may end up in flooding the queue
			*/

			return
		}

		delete(q.msgIDerrs, msgID)
	}()

	select {
	case <-releaseWaitErr:
		log.Info("Processed message from queue")
		return err
	case <-time.After(time.Second * time.Duration(timeoutSeconds)):
		return ErrorDeleteMessageTimeout
	}
}

func (q *queueSQS) unmarshal(body string) interface{} {
	msg := msgJSON{}
	bytesMsg := []byte(body)

	if err := json.Unmarshal(bytesMsg, &msg); err != nil {
		return body
	}

	return msg.Msg
}

func (q *queueSQS) resendMessage(m *sqs.Message) error {
	delayRetry := int64(0)
	messageAttributes := m.MessageAttributes

	if delayRetryAttr, ok := messageAttributes["NextDelayRetry"]; ok && delayRetryAttr.StringValue != nil {
		if delayRetryAttr.StringValue != nil {
			delayRetryValue, err := strconv.ParseInt(*delayRetryAttr.StringValue, 10, 64)
			delayRetry = delayRetryValue

			if err != nil {
				return errors.Wrap(err, "Incorrect value of NextDelayRetry")
			}
		} else {
			return ErrorDelayRetryAttrNil
		}
	}

	method := ""
	if methodAttr, ok := messageAttributes["Method"]; ok {
		if methodAttr.StringValue != nil {
			method = *methodAttr.StringValue
		} else {
			return ErrorMethodAttrNil
		}
	}

	return q.PutString(method, *m.Body, delayRetry)
}

func (q *queueSQS) prepareMessageID(m *sqs.Message) (string, error) {
	msgID := ""
	if m.MessageId != nil {
		msgID = *m.MessageId
		if _, ok := q.msgIDerrs[msgID]; !ok {
			q.msgIDerrs[msgID] = 0
		}
	} else {
		return "", ErrorMessageIDNotFound
	}

	if q.msgIDerrs[msgID] == maxNumberOfRetries {
		delete(q.msgIDerrs, msgID)

		return "", ErrorRequestMaxRetries
	}

	return msgID, nil
}
