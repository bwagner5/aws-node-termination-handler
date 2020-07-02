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

package interruptionevent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-node-termination-handler/pkg/ec2metadata"
	"github.com/aws/aws-node-termination-handler/pkg/node"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/rs/zerolog/log"
)

const (
	// SQSTerminateKind is a const to define an SQS termination kind of interruption event
	SQSTerminateKind = "SQS_TERMINATE"
)

// Example SQS ASG Lifecycle Termination Event Message:
// {
//   "version": "0",
//   "id": "782d5b4c-0f6f-1fd6-9d62-ecf6aed0a470",
//   "detail-type": "EC2 Instance-terminate Lifecycle Action",
//   "source": "aws.autoscaling",
//   "account": "896453262834",
//   "time": "2020-07-01T22:19:58Z",
//   "region": "us-east-1",
//   "resources": [
//     "arn:aws:autoscaling:us-east-1:896453262834:autoScalingGroup:26e7234b-03a4-47fb-b0a9-2b241662774e:autoScalingGroupName/testt1.demo-0a20f32c.kops.sh"
//   ],
//   "detail": {
//     "LifecycleActionToken": "0befcbdb-6ecd-498a-9ff7-ae9b54447cd6",
//     "AutoScalingGroupName": "testt1.demo-0a20f32c.kops.sh",
//     "LifecycleHookName": "cluster-termination-handler",
//     "EC2InstanceId": "i-0633ac2b0d9769723",
//     "LifecycleTransition": "autoscaling:EC2_INSTANCE_TERMINATING"
//   }
// }

// ASGLifecycleTerminationEvent is a structure to hold ASG lifecycle termination events sent from EventBridge to SQS
type ASGLifecycleTerminationEvent struct {
	Version    string          `json:"version"`
	ID         string          `json:"id"`
	DetailType string          `json:"detail-type"`
	Source     string          `json:"source"`
	Account    string          `json:"account"`
	Time       string          `json:"time"`
	Region     string          `json:"region"`
	Resources  []string        `json:"resources"`
	Detail     LifecycleDetail `json:"detail"`
}

// LifecycleDetail provides the ASG lifecycle event details
type LifecycleDetail struct {
	LifecycleActionToken string
	AutoScalingGroupName string
	LifecycleHookName    string
	EC2InstanceID        string `json:"EC2InstanceId"`
	LifecycleTransition  string
}

// MonitorForSQSTerminationEvents continuously monitors SQS for events and sends interruption events to the passed in channel
func MonitorForSQSTerminationEvents(interruptionChan chan<- InterruptionEvent, cancelChan chan<- InterruptionEvent, _ *ec2metadata.Service) error {
	interruptionEvent, err := checkForSQSMessage()
	if err != nil {
		return err
	}
	if interruptionEvent != nil && interruptionEvent.Kind == SQSTerminateKind {
		log.Log().Msgf("Sending %s interruption event to the interruption channel", SQSTerminateKind)
		interruptionChan <- *interruptionEvent
	}
	return nil
}

// checkForSpotInterruptionNotice Checks EC2 instance metadata for a spot interruption termination notice
func checkForSQSMessage() (*InterruptionEvent, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := sqs.New(sess)
	asg := autoscaling.New(sess)

	qURL := "https://sqs.us-east-1.amazonaws.com/896453262834/cth"

	log.Log().Msg("Checking for queue messages")
	messages, err := receiveQueueMessages(svc, qURL)
	if err != nil {
		fmt.Printf("Error while retrieving SQS messages: %v", err)
	}
	if len(messages) == 0 {
		return nil, nil
	}

	lifecycleEvent := ASGLifecycleTerminationEvent{}
	err = json.Unmarshal([]byte(*messages[0].Body), lifecycleEvent)
	if err != nil {
		return nil, err
	}
	terminationTime, err := time.Parse(time.RFC3339, lifecycleEvent.Time)
	if err != nil {
		terminationTime = time.Now()
	}
	interruptionEvent := &InterruptionEvent{
		EventID:     fmt.Sprintf("asg-lifecycle-term-%x", lifecycleEvent.ID),
		Kind:        SQSTerminateKind,
		StartTime:   terminationTime,
		Description: fmt.Sprintf("ASG Lifecycle Termination event received. Instance will be interrupted at %s \n", terminationTime),
	}

	interruptionEvent.PreDrainTask = func(interruptionEvent InterruptionEvent, n node.Node) error {
		err := n.TaintSpotItn(interruptionEvent.EventID)
		if err != nil {
			log.Log().Msgf("Unable to taint node with taint %s:%s: %w", node.ASGLifecycleTerminationTaint, interruptionEvent.EventID, err)
		}

		_, err = asg.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
			AutoScalingGroupName:  &lifecycleEvent.Detail.AutoScalingGroupName,
			LifecycleActionResult: aws.String("CONTINUE"),
			LifecycleHookName:     &lifecycleEvent.Detail.LifecycleHookName,
			LifecycleActionToken:  &lifecycleEvent.Detail.LifecycleActionToken,
			InstanceId:            &lifecycleEvent.Detail.EC2InstanceID,
		})
		if err != nil {
			return err
		}
		errs := deleteMessages(svc, qURL, []*sqs.Message{messages[0]})
		if errs != nil {
			return errs[0]
		}
		return nil
	}
	return interruptionEvent, err

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
