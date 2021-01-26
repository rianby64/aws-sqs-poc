package queue

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"

	// nolint: depguard
	log "github.com/sirupsen/logrus"
)

// listen - A worker loop that reads and processes queue messages.
func (q *queueSQS) listen() error {
	params := sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.URL),
		MaxNumberOfMessages: aws.Int64(maxNumberOfMessages),
		MessageAttributeNames: []*string{
			aws.String("All"), // Required
		},
		WaitTimeSeconds:   aws.Int64(waitTimeSeconds),
		VisibilityTimeout: aws.Int64(waitTimeSeconds), // (1) check footnote
	}

	log.Info("Starting the listen process")

	for {
		resp, err := q.SQS.ReceiveMessage(&params)

		if err != nil {
			return errors.Wrap(err, "SQS.ReceiveMessage error")
		}

		if len(resp.Messages) > 0 {
			for _, msg := range resp.Messages {
				handler, err := q.matchHandler(msg)
				if err != nil {
					return err
				}

				if err := q.handleMessage(handler, msg); err != nil {
					return errors.Wrap(err, "handling queue message")
				}
			}
		}
	}
}

func (q *queueSQS) matchHandler(msg *sqs.Message) (MessageHandler, error) {
	methodName := ""
	messageAttributes := msg.MessageAttributes

	if methodNameAttr, ok := messageAttributes["Method"]; ok {
		methodName = *methodNameAttr.StringValue
	}

	if handler, ok := q.handlerMap[methodName]; ok {
		return handler, nil
	}

	return nil, ErrorHandlerNotFound
}

/*
(1) The VisibilityTimeout ensures that only once the message will be available to one instace
	Suppose we've two or more instances listening to the queue. If a message appears in the queue
	then, different listeners can grab the same message at the same moment. AWS ensures that a read
	operation will happen atomically. And, once the message has been delivered to one listener, this message
	should become unavailable to all other listeners, and that's the purpose of VisibilityTimeout.
	Even if the value of VisibilityTimeout equals zero, the atomicity property still works.
*/
