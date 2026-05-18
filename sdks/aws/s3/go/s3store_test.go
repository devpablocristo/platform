package s3store

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "sa-east-1")
	t.Setenv("REPORTS_S3_BUCKET", "reports")
	t.Setenv("REPORTS_S3_ENDPOINT", "http://localhost:4566")
	t.Setenv("REPORTS_S3_PRESIGN_ENDPOINT", "http://public.localhost:4566")
	t.Setenv("REPORTS_S3_FORCE_PATH_STYLE", "true")

	config := ConfigFromEnv("REPORTS_S3")

	if config.Region != "sa-east-1" {
		t.Fatalf("unexpected region: %q", config.Region)
	}
	if config.Bucket != "reports" {
		t.Fatalf("unexpected bucket: %q", config.Bucket)
	}
	if config.Endpoint != "http://localhost:4566" {
		t.Fatalf("unexpected endpoint: %q", config.Endpoint)
	}
	if config.PresignEndpoint != "http://public.localhost:4566" {
		t.Fatalf("unexpected presign endpoint: %q", config.PresignEndpoint)
	}
	if !config.ForcePathStyle {
		t.Fatal("expected force path style")
	}
}

func TestPutUsesConfiguredBucket(t *testing.T) {
	t.Parallel()

	api := &fakeS3API{}
	client := NewWithAPI(api, &fakePresignAPI{}, "reports")

	err := client.Put(context.Background(), PutInput{
		Key:         "exports/report.csv",
		Body:        []byte("hello"),
		ContentType: "text/csv",
		Metadata: map[string]string{
			"tenant": "acme",
		},
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if api.putInput == nil {
		t.Fatal("expected put input")
	}
	if got := *api.putInput.Bucket; got != "reports" {
		t.Fatalf("unexpected bucket: %q", got)
	}
	if got := *api.putInput.Key; got != "exports/report.csv" {
		t.Fatalf("unexpected key: %q", got)
	}
	if got := *api.putInput.ContentType; got != "text/csv" {
		t.Fatalf("unexpected content type: %q", got)
	}
	body, err := io.ReadAll(api.putInput.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected body: %q", string(body))
	}
	if got := api.putInput.Metadata["tenant"]; got != "acme" {
		t.Fatalf("unexpected metadata: %q", got)
	}
}

func TestGetReadsObjectBody(t *testing.T) {
	t.Parallel()

	api := &fakeS3API{
		getBody: []byte("payload"),
	}
	client := NewWithAPI(api, &fakePresignAPI{}, "reports")

	body, err := client.Get(context.Background(), "exports/report.csv")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestPresignGetUsesFilenameAndTTL(t *testing.T) {
	t.Parallel()

	presign := &fakePresignAPI{url: "https://example.com/object"}
	client := NewWithAPI(&fakeS3API{}, presign, "reports")

	url, err := client.PresignGet(context.Background(), "exports/report.csv", 30*time.Minute, "report.csv")
	if err != nil {
		t.Fatalf("PresignGet returned error: %v", err)
	}
	if url != "https://example.com/object" {
		t.Fatalf("unexpected url: %q", url)
	}
	if presign.input == nil || presign.input.ResponseContentDisposition == nil {
		t.Fatal("expected response content disposition")
	}
	if !strings.Contains(*presign.input.ResponseContentDisposition, "filename=report.csv") {
		t.Fatalf("unexpected content disposition: %q", *presign.input.ResponseContentDisposition)
	}
	if presign.expires != 30*time.Minute {
		t.Fatalf("unexpected ttl: %s", presign.expires)
	}
}

func TestPutRejectsMissingKey(t *testing.T) {
	t.Parallel()

	client := NewWithAPI(&fakeS3API{}, &fakePresignAPI{}, "reports")
	err := client.Put(context.Background(), PutInput{})
	if err == nil || !strings.Contains(err.Error(), "key is required") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}

type fakeS3API struct {
	putInput *s3.PutObjectInput
	getBody  []byte
	putErr   error
	getErr   error
}

func (api *fakeS3API) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	api.putInput = input
	if api.putErr != nil {
		return nil, api.putErr
	}
	return &s3.PutObjectOutput{}, nil
}

func (api *fakeS3API) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if api.getErr != nil {
		return nil, api.getErr
	}
	if input == nil {
		return nil, errors.New("missing input")
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(api.getBody)),
	}, nil
}

type fakePresignAPI struct {
	input    *s3.GetObjectInput
	putInput *s3.PutObjectInput
	expires  time.Duration
	url      string
	err      error
}

func (api *fakePresignAPI) PresignGetObject(_ context.Context, input *s3.GetObjectInput, options ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	api.input = input
	cfg := s3.PresignOptions{}
	for _, option := range options {
		option(&cfg)
	}
	api.expires = cfg.Expires
	if api.err != nil {
		return nil, api.err
	}
	return &v4.PresignedHTTPRequest{URL: api.url}, nil
}

func (api *fakePresignAPI) PresignPutObject(_ context.Context, input *s3.PutObjectInput, options ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	api.putInput = input
	cfg := s3.PresignOptions{}
	for _, option := range options {
		option(&cfg)
	}
	api.expires = cfg.Expires
	if api.err != nil {
		return nil, api.err
	}
	return &v4.PresignedHTTPRequest{URL: api.url}, nil
}

func TestPresignPutAndExtractKey(t *testing.T) {
	t.Parallel()

	presign := &fakePresignAPI{url: "https://example.com/upload"}
	client := NewWithAPI(&fakeS3API{}, presign, "reports")

	url, err := client.PresignPut(context.Background(), PutInput{
		Key:         "exports/report.csv",
		ContentType: "text/csv",
		Metadata:    map[string]string{"tenant": "acme"},
	}, 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut returned error: %v", err)
	}
	if url != "https://example.com/upload" {
		t.Fatalf("unexpected url: %q", url)
	}
	if presign.putInput == nil || presign.putInput.Metadata["tenant"] != "acme" {
		t.Fatalf("unexpected put input: %#v", presign.putInput)
	}

	key := ExtractKey("reports", "https://reports.s3.sa-east-1.amazonaws.com/exports/report.csv?x=1")
	if key != "exports/report.csv" {
		t.Fatalf("unexpected key: %q", key)
	}
}
