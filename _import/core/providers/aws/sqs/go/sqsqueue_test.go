package sqsqueue

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "sa-east-1")
	t.Setenv("WORKER_SQS_ENDPOINT", "http://localhost:4566")
	t.Setenv("WORKER_SQS_QUEUE_URL", "https://sqs.local/queue")

	config := ConfigFromEnv("WORKER_SQS")

	if config.Region != "sa-east-1" {
		t.Fatalf("unexpected region: %q", config.Region)
	}
	if config.Endpoint != "http://localhost:4566" {
		t.Fatalf("unexpected endpoint: %q", config.Endpoint)
	}
	if config.QueueURL != "https://sqs.local/queue" {
		t.Fatalf("unexpected queue url: %q", config.QueueURL)
	}
}

func TestSendUsesConfiguredQueue(t *testing.T) {
	t.Parallel()

	api := &fakeQueueAPI{}
	client := NewWithAPI(api, "https://sqs.local/queue")

	messageID, err := client.Send(context.Background(), "hello", SendOptions{
		DelaySeconds:         5,
		MessageGroupID:       "group-1",
		MessageDeduplication: "dedupe-1",
		Attributes: map[string]string{
			"tenant": "acme",
		},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if messageID != "mid-123" {
		t.Fatalf("unexpected message id: %q", messageID)
	}
	if api.input == nil {
		t.Fatal("expected send message input")
	}
	if got := aws.ToString(api.input.QueueUrl); got != "https://sqs.local/queue" {
		t.Fatalf("unexpected queue url: %q", got)
	}
	if got := aws.ToString(api.input.MessageBody); got != "hello" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := aws.ToString(api.input.MessageGroupId); got != "group-1" {
		t.Fatalf("unexpected group id: %q", got)
	}
	if got := aws.ToString(api.input.MessageDeduplicationId); got != "dedupe-1" {
		t.Fatalf("unexpected dedupe id: %q", got)
	}
	if got := aws.ToString(api.input.MessageAttributes["tenant"].StringValue); got != "acme" {
		t.Fatalf("unexpected tenant attribute: %q", got)
	}
}

func TestSendJSONMarshalsPayload(t *testing.T) {
	t.Parallel()

	api := &fakeQueueAPI{}
	client := NewWithAPI(api, "https://sqs.local/queue")

	_, err := client.SendJSON(context.Background(), map[string]any{"kind": "report", "ok": true}, SendOptions{})
	if err != nil {
		t.Fatalf("SendJSON returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(aws.ToString(api.input.MessageBody)), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["kind"] != "report" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestSendRejectsMissingQueueURL(t *testing.T) {
	t.Parallel()

	client := NewWithAPI(&fakeQueueAPI{}, "")
	_, err := client.Send(context.Background(), "hello", SendOptions{})
	if err == nil || !strings.Contains(err.Error(), "queue url is required") {
		t.Fatalf("expected missing queue url error, got %v", err)
	}
}

type fakeQueueAPI struct {
	input *sqs.SendMessageInput
	err   error
}

func (api *fakeQueueAPI) SendMessage(_ context.Context, input *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	api.input = input
	if api.err != nil {
		return nil, api.err
	}
	return &sqs.SendMessageOutput{MessageId: aws.String("mid-123")}, nil
}
