package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/slack-go/slack"

	"github.com/rianby64/aws-sqs-poc/queue"
)

// MsgOptionText jajaja
type MsgOptionText struct {
	Text   string
	Escape bool
}

// PostMessageArgs jajajaj
type PostMessageArgs struct {
	MsgOptionText MsgOptionText
	ChannelID     string
}

// NewAWSSession - Creates a new AWS session.
func NewAWSSession() (*session.Session, error) {
	credsValue := credentials.Value{
		AccessKeyID:     "--",
		SecretAccessKey: "--",
	}

	creds := credentials.NewStaticCredentialsFromCreds(credsValue)
	_, err := creds.Get()

	if err != nil {
		return nil, err
	}

	return session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-1"),
	})
}

// main it is
func main() {
	slackClient := slack.New("--")

	i := 0
	myhandler := func(msg interface{}) error {
		params, ok := msg.(PostMessageArgs)
		if !ok {
			return errors.New("cannot cast msg into PostMessageArgs")
		}

		options := slack.MsgOptionText(params.MsgOptionText.Text, params.MsgOptionText.Escape)
		_, _, err := slackClient.PostMessage(params.ChannelID, options)
		i++
		fmt.Println("Sending the message", options, slackClient, i)
		return err //errors.New("this is the error")
	}

	awssession, _ := NewAWSSession()
	sqssession := sqs.New(awssession)
	myq := queue.NewSQSQueue(sqssession, "https://sqs.us-east-1.amazonaws.com/490043543248/my-queue-test")
	myq.Register("", myhandler)

	channelID := "C01JG8ALHHV"
	msg := "This is my message to send ogo ogo ogo"

	optionsJSON, _ := json.Marshal(PostMessageArgs{
		MsgOptionText: MsgOptionText{Text: msg, Escape: false},
		ChannelID:     channelID,
	})

	toSend := string(optionsJSON)

	if err := myq.Put(toSend, 0); err != nil {
		fmt.Println(err)
	}

	fmt.Println(channelID)
	time.Sleep(5 * time.Hour)

}
