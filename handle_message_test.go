package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
)

type Mock4handleMessageAWSSession struct {
	TimesCalledDeleteMessage int
	TimesCalledSendMessage   int
	Receipt                  string
	ShouldDeleteMessageFail  bool
	DeleteMessageFailError   string
	DeleteMessageTimeout     int64
	LastNextDelayRetry       *string
	LastBodySent             *string
}

func (a *Mock4handleMessageAWSSession) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	a.TimesCalledSendMessage++
	a.LastNextDelayRetry = input.MessageAttributes["NextDelayRetry"].StringValue
	a.LastBodySent = input.MessageBody
	return &sqs.SendMessageOutput{
		MessageId:        aws.String("messageID"),
		MD5OfMessageBody: aws.String("messageID"),
	}, nil
}

func (a *Mock4handleMessageAWSSession) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return nil, nil
}

func (a *Mock4handleMessageAWSSession) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	a.TimesCalledDeleteMessage++
	a.Receipt = *input.ReceiptHandle

	if a.DeleteMessageTimeout > 0 {
		time.Sleep(time.Duration(a.DeleteMessageTimeout) * time.Second)
	}

	if a.ShouldDeleteMessageFail {
		return nil, errors.New(a.DeleteMessageFailError)
	}

	return nil, nil
}

/*
	Case 1: The handler receives a hander-function and a message.
	First it tries to delete it, then if OK it sends the message to the handler
*/
func Test_handleMessage_once(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	finish := make(chan bool)

	const expectedReceipt = "a receipt handle"
	const expectedMessage = "a message"

	handler := func(msg interface{}) error {
		finish <- true
		assert.Equal(t, expectedMessage, msg)
		return nil
	}

	queue := queueSQS{
		thens:     map[string][]MessageHandler{},
		SQS:       session,
		URL:       "",
		msgIDerrs: map[string]int{},
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	msg.MessageId = aws.String("messageID")
	msg.MD5OfBody = aws.String("messageID")
	err := queue.handleMessage(handler, &msg)

	<-finish

	assert.Nil(t, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage) // once called
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 2: The handler receives a hander-function and a message.
	First it tries to delete it, then if OK it sends the message to the handler
	But, if handler returns error then resend the message. First error
*/
func Test_handleMessage_resend(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	finish := make(chan bool)

	const expectedReceipt = "a receipt handle"
	const expectedMessage = "a message"

	handler := func(msg interface{}) error {
		go func() {
			finish <- true
		}()
		assert.Equal(t, expectedMessage, msg)
		return errors.New("intentional error") // this triggers the resend process
	}

	queue := queueSQS{
		thens:     map[string][]MessageHandler{},
		SQS:       session,
		URL:       "",
		msgIDerrs: map[string]int{},
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	msg.MessageId = aws.String("messageID")
	msg.MD5OfBody = aws.String("messageID")
	err := queue.handleMessage(handler, &msg)

	<-finish

	assert.Nil(t, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 1, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 3: The handler receives a hander-function and a message.
	First it tries to delete it, but it fails. Then returns an error
*/
func Test_handleMessage_deletion_error(t *testing.T) {
	session := &Mock4handleMessageAWSSession{
		ShouldDeleteMessageFail: true,
		DeleteMessageFailError:  "an intentional error",
	}

	const expectedReceipt = "a receipt handle"
	const expectedMessage = "a message" // it's not important, but it's OK to define it

	handler := func(msg interface{}) error {
		return nil
	}

	queue := queueSQS{
		thens:     map[string][]MessageHandler{},
		SQS:       session,
		URL:       "",
		msgIDerrs: map[string]int{},
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	msg.MessageId = aws.String("messageID")
	msg.MD5OfBody = aws.String("messageID")
	err := queue.handleMessage(handler, &msg)

	assert.NotNil(t, err)
	assert.Equal(t, session.DeleteMessageFailError, err.Error())

	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 0, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

/*
	Case 4: The handler receives a hander-function and a message.
	First it tries to delete it, but it fails. Then returns an error
*/
func Test_handleMessage_deletion_timeout(t *testing.T) {
	session := &Mock4handleMessageAWSSession{
		DeleteMessageTimeout: 2,
	}

	const expectedReceipt = "a receipt handle"
	const expectedMessage = "a message" // it's not important, but it's OK to define it

	handler := func(msg interface{}) error {
		return nil
	}

	queue := queueSQS{
		thens:          map[string][]MessageHandler{},
		SQS:            session,
		URL:            "",
		TimeoutSeconds: 1,
		msgIDerrs:      map[string]int{},
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	msg.MessageId = aws.String("messageID")
	msg.MD5OfBody = aws.String("messageID")
	err := queue.handleMessage(handler, &msg)

	assert.NotNil(t, err)
	assert.Equal(t, ErrorDeleteMessageTimeout, err)
	assert.Equal(t, 1, session.TimesCalledDeleteMessage)
	assert.Equal(t, 0, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

func Test_resendMessage_OK(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	queue := queueSQS{
		thens:                    map[string][]MessageHandler{},
		SQS:                      session,
		URL:                      "",
		TimeoutSeconds:           1,
		NextDelayIncreaseSeconds: 3,
	}

	currentRetry := int64(10)
	const expectedBody = "something to send"
	expectedRetry := fmt.Sprintf("%d", currentRetry+queue.NextDelayIncreaseSeconds)

	msg := sqs.Message{}
	msg.Body = aws.String(expectedBody)
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("number"),
			StringValue: aws.String(fmt.Sprintf("%d", currentRetry)),
		},
	}
	msg.MD5OfBody = aws.String("messageID")

	err := queue.resendMessage(&msg)
	assert.Nil(t, err)
	assert.Equal(t, expectedRetry, *session.LastNextDelayRetry)
	assert.Equal(t, expectedBody, *session.LastBodySent)
}

func Test_resendMessage_Incorrect_NextDelayRetry(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	queue := queueSQS{
		thens:                    map[string][]MessageHandler{},
		SQS:                      session,
		URL:                      "",
		TimeoutSeconds:           1,
		NextDelayIncreaseSeconds: 3,
	}

	incorrectNumber := "NaN"
	msg := sqs.Message{}
	msg.Body = aws.String("")
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"NextDelayRetry": {
			DataType:    aws.String("number"),
			StringValue: aws.String(incorrectNumber),
		},
	}
	msg.MD5OfBody = aws.String("messageID")

	expectedErrorStr := fmt.Sprintf(`Incorrect value of NextDelayRetry: strconv.ParseInt: parsing "%s": invalid syntax`, incorrectNumber)
	err := queue.resendMessage(&msg)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErrorStr, err.Error())
}

func Test_resendMessage_Nil_Method(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}
	queue := queueSQS{
		thens:                    map[string][]MessageHandler{},
		SQS:                      session,
		URL:                      "",
		TimeoutSeconds:           1,
		NextDelayIncreaseSeconds: 3,
	}

	msg := sqs.Message{}
	msg.Body = aws.String("")
	msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		"Method": {
			DataType:    aws.String("string"),
			StringValue: nil,
		},
		"NextDelayRetry": {
			DataType:    aws.String("number"),
			StringValue: aws.String("10"),
		},
	}
	msg.MD5OfBody = aws.String("messageID")

	err := queue.resendMessage(&msg)
	assert.NotNil(t, err)
	assert.Equal(t, ErrorMethodAttrNil, err)
}

/*
	Case 5: The handler receives a hander-function and a message.
	First it tries to delete it, then if OK it sends the message to the handler
	But, if handler returns error then resend the message. Let's reach the max number of retries
*/
func Test_handleMessage_resend_maxNumberOfRetries_reached(t *testing.T) {
	session := &Mock4handleMessageAWSSession{}

	const expectedReceipt = "a receipt handle"
	const expectedMessage = "a message"

	i := 0
	handler := func(msg interface{}) error {
		i++
		assert.Equal(t, expectedMessage, msg)
		return errors.New("intentional error") // this triggers the resend process
	}

	queue := queueSQS{
		thens:     map[string][]MessageHandler{},
		SQS:       session,
		URL:       "",
		msgIDerrs: map[string]int{},
	}

	msg := sqs.Message{}
	msg.Body = aws.String(expectedMessage)
	msg.ReceiptHandle = aws.String(expectedReceipt)
	msg.MessageId = aws.String("messageID")
	msg.MD5OfBody = aws.String("messageID")

	j := 0
	err := func() error {
		for {
			err := queue.handleMessage(handler, &msg)
			j++
			if err != nil {
				return err
			}
		}
	}()

	assert.NotNil(t, err)
	assert.Equal(t, i+1, j)
	assert.Equal(t, ErrorRequestMaxRetries, err)
	assert.Equal(t, maxNumberOfRetries+1, session.TimesCalledDeleteMessage)
	assert.Equal(t, maxNumberOfRetries, session.TimesCalledSendMessage)
	assert.Equal(t, expectedReceipt, session.Receipt)
}

// This test is for educational purposes
func Test_unmarshal_complex_thing(t *testing.T) {
	queue := queueSQS{
		thens: map[string][]MessageHandler{},
	}

	type complexObject struct {
		Field1 *string
		Field2 int64
		Field3 float64
		Field4 []string
		Field5 struct {
			Key    *string
			ValueA *string
			ValueB *float64
		}
	}

	params := complexObject{
		Field1: aws.String("ptr string"),
		Field2: 1,
		Field3: 1.0,
		Field4: []string{"str1", "str2"},
		Field5: struct {
			Key    *string
			ValueA *string
			ValueB *float64
		}{
			Key:    aws.String("key1"),
			ValueA: aws.String("value1"),
			ValueB: aws.Float64(float64(1.0)),
		},
	}

	paramsStr, _ := json.Marshal(msgJSON{Msg: params})

	msgBody := queue.unmarshal(string(paramsStr))
	reversed, err := json.Marshal(msgBody)
	assert.Nil(t, err)

	actual := complexObject{}
	assert.Nil(t, json.Unmarshal(reversed, &actual))
	assert.Equal(t, params, actual)
}

// This test is for educational purposes
func Test_unmarshal_2_several_structures(t *testing.T) {
	type Type1 struct {
		Type1  string  `json:"type1" validate:"string" type:"string" required:"true"`
		Field1 float64 `json:"field1" validate:"number"`
	}

	type Type2 struct {
		Type2  string   `json:"type2" type:"string" required:"true"`
		Field2 *float64 `json:"field2" validate:"number"`
	}

	type Type3 struct {
		Type3  string  `json:"type3" type:"string" required:"true"`
		Field3 *string `json:"field3" validate:"string"`
	}

	struct1 := Type1{
		Type1:  "type 1",
		Field1: 1.0,
	}

	struct2 := Type2{
		Type2:  "type 1",
		Field2: aws.Float64(1.0),
	}

	struct3 := Type3{
		Type3:  "type 1",
		Field3: aws.String("type 3"),
	}

	str1B, _ := json.Marshal(struct1)
	str1 := string(str1B)
	str2B, _ := json.Marshal(struct2)
	str2 := string(str2B)
	str3B, _ := json.Marshal(struct3)
	str3 := string(str3B)

	fmt.Println(str1, str2, str3)

	reversed1 := Type1{}
	reversed2 := Type2{}
	reversed3 := Type3{}

	func() {
		if err := json.Unmarshal(str1B, &reversed3); err == nil {
			fmt.Println("reversed 1 for 3")
			return
		}

		if err := json.Unmarshal(str1B, &reversed2); err == nil {
			fmt.Println("reversed 1 for 2")
			return
		}

		if err := json.Unmarshal(str1B, &reversed1); err == nil {
			fmt.Println("reversed 1 for 1")
			return
		}
	}()

	func() {
		if err := json.Unmarshal(str2B, &reversed3); err == nil {
			fmt.Println("reversed 2 for 3")
			return
		}

		if err := json.Unmarshal(str2B, &reversed2); err == nil {
			fmt.Println("reversed 2 for 2")
			return
		}

		if err := json.Unmarshal(str2B, &reversed1); err == nil {
			fmt.Println("reversed 2 for 1")
			return
		}
	}()

	func() {
		if err := json.Unmarshal(str3B, &reversed3); err == nil {
			fmt.Println("reversed 3 for 3")
			return
		}

		if err := json.Unmarshal(str3B, &reversed2); err == nil {
			fmt.Println("reversed 3 for 2")
			return
		}

		if err := json.Unmarshal(str3B, &reversed1); err == nil {
			fmt.Println("reversed 3 for 1")
			return
		}
	}()

}
