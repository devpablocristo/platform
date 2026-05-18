package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devpablocristo/core/ingestion/go"
)

func TestClientExtract_OK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/extract" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(ingestion.ExtractResponse{
			Artifacts: []ingestion.NormalizedArtifact{
				{
					Kind:     "text_document",
					Version:  1,
					FullText: "hello",
					Provenance: ingestion.Provenance{
						Engine:        "test",
						EngineVersion: "1",
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	cli := New(srv.URL)
	out, err := cli.Extract(context.Background(), ingestion.ExtractRequest{
		AssetID:     "a1",
		ContentType: "text/plain",
		ByteSize:    5,
		StorageRef:  ingestion.StorageRef{Kind: "s3", Key: "k"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Artifacts) != 1 || out.Artifacts[0].FullText != "hello" {
		t.Fatalf("unexpected: %+v", out)
	}
}

func TestClientExtract_ServiceError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ingestion.ErrorBody{
			Code:    ingestion.CodeUnsupported,
			Message: "no",
		})
	}))
	t.Cleanup(srv.Close)

	cli := New(srv.URL)
	_, err := cli.Extract(context.Background(), ingestion.ExtractRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	ee, ok := err.(ingestion.ExtractError)
	if !ok || ee.Code != ingestion.CodeUnsupported {
		t.Fatalf("got %v", err)
	}
}
