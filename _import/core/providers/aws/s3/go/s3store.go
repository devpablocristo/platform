package s3store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const defaultPresignTTL = time.Hour

// Config representa la configuración mínima de S3.
type Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	PresignEndpoint string
	ForcePathStyle  bool
}

// PutInput representa un put object mínimo y reusable.
type PutInput struct {
	Key         string
	Body        []byte
	ContentType string
	Metadata    map[string]string
}

type objectAPI interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type presignAPI interface {
	PresignGetObject(context.Context, *s3.GetObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignPutObject(context.Context, *s3.PutObjectInput, ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// Client envuelve operaciones comunes de S3 con bucket fijo.
type Client struct {
	api     objectAPI
	presign presignAPI
	bucket  string
}

// ConfigFromEnv carga configuración desde env.
func ConfigFromEnv(prefix string) Config {
	prefix = normalizeEnvPrefix(prefix)
	if prefix == "" {
		return Config{
			Region:          firstNonEmpty(os.Getenv("AWS_REGION"), "us-east-1"),
			Bucket:          firstNonEmpty(os.Getenv("S3_BUCKET")),
			Endpoint:        firstNonEmpty(os.Getenv("S3_ENDPOINT")),
			PresignEndpoint: firstNonEmpty(os.Getenv("S3_PRESIGN_ENDPOINT")),
			ForcePathStyle:  parseBool(os.Getenv("S3_FORCE_PATH_STYLE")),
		}
	}
	return Config{
		Region:          firstNonEmpty(os.Getenv(prefix+"AWS_REGION"), os.Getenv("AWS_REGION"), "us-east-1"),
		Bucket:          firstNonEmpty(os.Getenv(prefix+"BUCKET"), os.Getenv("S3_BUCKET")),
		Endpoint:        firstNonEmpty(os.Getenv(prefix+"ENDPOINT"), os.Getenv("S3_ENDPOINT")),
		PresignEndpoint: firstNonEmpty(os.Getenv(prefix+"PRESIGN_ENDPOINT"), os.Getenv("S3_PRESIGN_ENDPOINT")),
		ForcePathStyle:  parseBool(firstNonEmpty(os.Getenv(prefix+"FORCE_PATH_STYLE"), os.Getenv("S3_FORCE_PATH_STYLE"))),
	}
}

// New crea un cliente S3 desde configuración local.
func New(ctx context.Context, config Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return NewFromAWSConfig(awsCfg, config), nil
}

// NewFromAWSConfig crea un cliente S3 reutilizando un aws.Config existente.
func NewFromAWSConfig(awsCfg awssdk.Config, config Config) *Client {
	api := newS3Client(awsCfg, strings.TrimSpace(config.Endpoint), config.ForcePathStyle)
	presignAPI := api
	if endpoint := strings.TrimSpace(config.PresignEndpoint); endpoint != "" && endpoint != strings.TrimSpace(config.Endpoint) {
		presignAPI = newS3Client(awsCfg, endpoint, config.ForcePathStyle)
	}
	return NewWithAPI(api, s3.NewPresignClient(presignAPI), config.Bucket)
}

func newS3Client(awsCfg awssdk.Config, endpoint string, forcePathStyle bool) *s3.Client {
	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		if endpoint != "" {
			options.BaseEndpoint = awssdk.String(endpoint)
		}
		options.UsePathStyle = forcePathStyle
	})
}

// NewWithAPI crea un cliente desde implementaciones inyectadas.
func NewWithAPI(api objectAPI, presign presignAPI, bucket string) *Client {
	return &Client{
		api:     api,
		presign: presign,
		bucket:  strings.TrimSpace(bucket),
	}
}

// Bucket devuelve el bucket fijo del cliente.
func (c *Client) Bucket() string {
	if c == nil {
		return ""
	}
	return c.bucket
}

// Put sube un objeto al bucket configurado.
func (c *Client) Put(ctx context.Context, input PutInput) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("s3 client is nil")
	}
	if strings.TrimSpace(c.bucket) == "" {
		return fmt.Errorf("s3 bucket is required")
	}
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return fmt.Errorf("s3 key is required")
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := c.api.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      awssdk.String(c.bucket),
		Key:         awssdk.String(key),
		Body:        bytes.NewReader(input.Body),
		ContentType: awssdk.String(contentType),
		Metadata:    cloneMap(input.Metadata),
	})
	if err != nil {
		return fmt.Errorf("put s3 object: %w", err)
	}
	return nil
}

// Get descarga un objeto completo del bucket configurado.
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	if c == nil || c.api == nil {
		return nil, fmt.Errorf("s3 client is nil")
	}
	if strings.TrimSpace(c.bucket) == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("s3 key is required")
	}

	output, err := c.api.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awssdk.String(c.bucket),
		Key:    awssdk.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3 object: %w", err)
	}
	defer output.Body.Close()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("read s3 object body: %w", err)
	}
	return body, nil
}

// PresignGet genera una URL firmada de descarga.
func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration, filename string) (string, error) {
	if c == nil || c.presign == nil {
		return "", fmt.Errorf("s3 presign client is nil")
	}
	if strings.TrimSpace(c.bucket) == "" {
		return "", fmt.Errorf("s3 bucket is required")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("s3 key is required")
	}
	if ttl <= 0 {
		ttl = defaultPresignTTL
	}

	input := &s3.GetObjectInput{
		Bucket: awssdk.String(c.bucket),
		Key:    awssdk.String(key),
	}
	if value := contentDisposition(filename); value != "" {
		input.ResponseContentDisposition = awssdk.String(value)
	}

	request, err := c.presign.PresignGetObject(ctx, input, func(options *s3.PresignOptions) {
		options.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("presign s3 object: %w", err)
	}
	return request.URL, nil
}

// PresignPut genera una URL firmada para subir un objeto.
func (c *Client) PresignPut(ctx context.Context, input PutInput, ttl time.Duration) (string, error) {
	if c == nil || c.presign == nil {
		return "", fmt.Errorf("s3 presign client is nil")
	}
	if strings.TrimSpace(c.bucket) == "" {
		return "", fmt.Errorf("s3 bucket is required")
	}
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return "", fmt.Errorf("s3 key is required")
	}
	if ttl <= 0 {
		ttl = defaultPresignTTL
	}
	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	request, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      awssdk.String(c.bucket),
		Key:         awssdk.String(key),
		ContentType: awssdk.String(contentType),
		Metadata:    cloneMap(input.Metadata),
	}, func(options *s3.PresignOptions) {
		options.Expires = ttl
	})
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}
	return request.URL, nil
}

// ExtractKey admite keys crudas, `s3://bucket/key` o URLs S3.
func ExtractKey(bucket, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "s3://") || strings.HasPrefix(value, "s3:/") {
		cleaned := strings.TrimPrefix(strings.TrimPrefix(value, "s3://"), "s3:/")
		parts := strings.SplitN(cleaned, "/", 2)
		if len(parts) == 2 {
			return stripQueryParams(parts[1])
		}
		return ""
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return stripQueryParams(value)
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return stripQueryParams(value)
	}
	host := parsed.Hostname()
	path := strings.TrimPrefix(parsed.Path, "/")
	if bucket != "" && strings.HasPrefix(path, bucket+"/") {
		return strings.TrimPrefix(path, bucket+"/")
	}
	if strings.HasPrefix(host, bucket+".s3.") || strings.HasPrefix(host, bucket+".s3-") {
		return path
	}
	if strings.HasPrefix(path, bucket+"/") {
		return strings.TrimPrefix(path, bucket+"/")
	}
	return path
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

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func cloneMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cloned[key] = value
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func contentDisposition(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return ""
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": filename})
}

func stripQueryParams(value string) string {
	if idx := strings.IndexByte(value, '?'); idx != -1 {
		return value[:idx]
	}
	return value
}
