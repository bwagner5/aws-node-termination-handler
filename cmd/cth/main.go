// Copyright 2016-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := sqs.New(sess)

	qURL := "https://sqs.us-east-1.amazonaws.com/896453262834/cth"

	for range time.Tick(time.Second * 2) {
		fmt.Println("Checking for queue messages")
		messages, err := receiveQueueMessages(svc, qURL)
		if err != nil {
			fmt.Printf("Error while retrieving SQS messages: %v", err)
		}

		doSomethingWithMessages(messages)

		errs := deleteMessages(svc, qURL, messages)
		if errs != nil {
			fmt.Printf("Error deleting queue messages: %v", errs)
		}

	}
}

func receiveQueueMessages(svc *sqs.SQS, qURL string) ([]*sqs.Message, error) {
	result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &qURL,
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(20), // 20 seconds
		WaitTimeSeconds:     aws.Int64(0),
	})

	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

func doSomethingWithMessages(messages []*sqs.Message) {
	for _, message := range messages {
		fmt.Println("=========================================")
		fmt.Println(*message.Body)
		fmt.Println("=========================================")
	}
}

func deleteMessages(svc *sqs.SQS, qURL string, messages []*sqs.Message) []error {
	var errs []error
	for _, message := range messages {
		_, err := svc.DeleteMessage(&sqs.DeleteMessageInput{
			ReceiptHandle: message.ReceiptHandle,
			QueueUrl:      &qURL,
		})
		if err != nil {
			errs = append(errs, err)
		}

	}
	return errs
}

func deleteMessagesBatch(svc *sqs.SQS, qURL string, messages []*sqs.Message) []error {
	if len(messages) == 0 {
		return nil
	}
	var errs []error

	for len(messages) > 0 {
		deleteMessageEntries := []*sqs.DeleteMessageBatchRequestEntry{}
		messagesToDelete := messages
		if len(messages) > 10 {
			messagesToDelete = messages[:10]
		}
		for _, message := range messagesToDelete {
			deleteMessageEntry := sqs.DeleteMessageBatchRequestEntry{
				ReceiptHandle: message.ReceiptHandle,
				Id:            message.MessageId,
			}
			deleteMessageEntries = append(deleteMessageEntries, &deleteMessageEntry)
		}

		_, err := svc.DeleteMessageBatch(&sqs.DeleteMessageBatchInput{
			QueueUrl: &qURL,
			Entries:  deleteMessageEntries,
		})
		if err != nil {
			errs = append(errs, err)
		}
		messages = messages[len(messagesToDelete):]
	}

	return errs
}
