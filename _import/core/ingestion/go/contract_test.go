package ingestion

import (
	"encoding/json"
	"testing"
)

func TestExtractRequest_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	in := ExtractRequest{
		AssetID:      "id-1",
		ContentType:  "application/pdf",
		ByteSize:     42,
		StorageRef:   StorageRef{Kind: "s3", Bucket: "b", Key: "raw/x"},
		LocaleHint:   "es",
		CorrelationID: "job-9",
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out ExtractRequest
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}
