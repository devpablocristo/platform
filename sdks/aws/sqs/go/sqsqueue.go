package sqsqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Config representa la configuración mínima de una cola SQS.
type Config struct {
	Region   string
	Endpoint string
	QueueURL string
}

// SendOptions define opciones de envío reutilizables.
type SendOptions struct {
	DelaySeconds         int32
	MessageGroupID       string
	MessageDeduplication string
	Attributes           map[string]string
}

type queueAPI interface {
	SendMessage(context.Context, *sqs.SendMessageInput, ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// Client envuelve envíos comunes a una cola fija.
type Client struct {
	api      queueAPI
	queueURL string
}

// ConfigFromEnv carga configuración desde env.
func ConfigFromEnv(prefix string) Config {
	prefix = normalizeEnvPrefix(prefix)
	if prefix == "" {
		return Config{
			Region:   firstNonEmpty(os.Getenv("AWS_REGION"), "us-east-1"),
			Endpoint: firstNonEmpty(os.Getenv("SQS_ENDPOINT")),
			QueueURL: firstNonEmpty(os.Getenv("SQS_QUEUE_URL")),
		}
	}
	return Config{
		Region:   firstNonEmpty(os.Getenv(prefix+"AWS_REGION"), os.Getenv("AWS_REGION"), "us-east-1"),
		Endpoint: firstNonEmpty(os.Getenv(prefix+"ENDPOINT"), os.Getenv("SQS_ENDPOINT")),
		QueueURL: firstNonEmpty(os.Getenv(prefix+"QUEUE_URL"), os.Getenv("SQS_QUEUE_URL")),
	}
}

// New crea un cliente SQS desde la configuración local.
func New(ctx context.Context, config Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return NewFromAWSConfig(awsCfg, config), nil
}

// NewFromAWSConfig crea un cliente SQS reutilizando un aws.Config existente.
func NewFromAWSConfig(awsCfg awssdk.Config, config Config) *Client {
	api := sqs.NewFromConfig(awsCfg, func(options *sqs.Options) {
		if strings.TrimSpace(config.Endpoint) != "" {
			options.BaseEndpoint = awssdk.String(strings.TrimSpace(config.Endpoint))
		}
	})
	return NewWithAPI(api, config.QueueURL)
}

// NewWithAPI crea un cliente desde una implementación inyectada.
func NewWithAPI(api queueAPI, queueURL string) *Client {
	return &Client{
		api:      api,
		queueURL: strings.TrimSpace(queueURL),
	}
}

// QueueURL devuelve la cola fija configurada.
func (c *Client) QueueURL() string {
	if c == nil {
		return ""
	}
	return c.queueURL
}

// Send envía un mensaje raw a la cola configurada.
func (c *Client) Send(ctx context.Context, body string, options SendOptions) (string, error) {
	if c == nil || c.api == nil {
		return "", fmt.Errorf("sqs client is nil")
	}
	if strings.TrimSpace(c.queueURL) == "" {
		return "", fmt.Errorf("sqs queue url is required")
	}

	input := &sqs.SendMessageInput{
		QueueUrl:          awssdk.String(c.queueURL),
		MessageBody:       awssdk.String(body),
		DelaySeconds:      options.DelaySeconds,
		MessageAttributes: toMessageAttributes(options.Attributes),
	}
	if value := strings.TrimSpace(options.MessageGroupID); value != "" {
		input.MessageGroupId = awssdk.String(value)
	}
	if value := strings.TrimSpace(options.MessageDeduplication); value != "" {
		input.MessageDeduplicationId = awssdk.String(value)
	}

	output, err := c.api.SendMessage(ctx, input)
	if err != nil {
		return "", fmt.Errorf("send sqs message: %w", err)
	}
	return firstNonEmpty(awssdk.ToString(output.MessageId)), nil
}

// SendJSON serializa payload a JSON y luego envía el mensaje.
func (c *Client) SendJSON(ctx context.Context, payload any, options SendOptions) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal sqs payload: %w", err)
	}
	return c.Send(ctx, string(body), options)
}

func toMessageAttributes(values map[string]string) map[string]types.MessageAttributeValue {
	if len(values) == 0 {
		return nil
	}
	attributes := make(map[string]types.MessageAttributeValue, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		attributes[key] = types.MessageAttributeValue{
			DataType:    awssdk.String("String"),
			StringValue: awssdk.String(value),
		}
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

func normalizeEnvPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	return strings.TrimSuffix(prefix, "_") + "_"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
