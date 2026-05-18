package dynamodbtable

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "sa-east-1")
	t.Setenv("AUDIT_DYNAMODB_ENDPOINT", "http://localhost:4566")
	t.Setenv("AUDIT_DYNAMODB_TABLE", "audit-events")

	config := ConfigFromEnv("AUDIT_DYNAMODB")
	if config.Region != "sa-east-1" {
		t.Fatalf("unexpected region: %q", config.Region)
	}
	if config.Endpoint != "http://localhost:4566" {
		t.Fatalf("unexpected endpoint: %q", config.Endpoint)
	}
	if config.Table != "audit-events" {
		t.Fatalf("unexpected table: %q", config.Table)
	}
}

func TestPutJSONAndGetJSON(t *testing.T) {
	t.Parallel()

	api := &fakeItemAPI{}
	client := NewWithAPI(api, "audit-events")

	input := struct {
		PK   string `dynamodbav:"pk"`
		SK   string `dynamodbav:"sk"`
		Kind string `dynamodbav:"kind"`
	}{
		PK:   "tenant#acme",
		SK:   "evt#1",
		Kind: "report.ready",
	}
	if err := client.PutJSON(context.Background(), input); err != nil {
		t.Fatalf("PutJSON returned error: %v", err)
	}

	api.getItem = api.putInput.Item
	var out struct {
		PK   string `dynamodbav:"pk"`
		SK   string `dynamodbav:"sk"`
		Kind string `dynamodbav:"kind"`
	}
	found, err := client.GetJSON(context.Background(), struct {
		PK string `dynamodbav:"pk"`
		SK string `dynamodbav:"sk"`
	}{PK: "tenant#acme", SK: "evt#1"}, &out)
	if err != nil {
		t.Fatalf("GetJSON returned error: %v", err)
	}
	if !found {
		t.Fatal("expected item to be found")
	}
	if out.Kind != "report.ready" {
		t.Fatalf("unexpected out item: %#v", out)
	}
}

func TestPutItemRejectsMissingTable(t *testing.T) {
	t.Parallel()

	client := NewWithAPI(&fakeItemAPI{}, "")
	err := client.PutItem(context.Background(), map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberS{Value: "x"},
	})
	if err == nil {
		t.Fatal("expected missing table error")
	}
}

type fakeItemAPI struct {
	putInput      *dynamodb.PutItemInput
	getItem       map[string]types.AttributeValue
	batchGetInput *dynamodb.BatchGetItemInput
	batchGetCalls int
	batchGetOut   *dynamodb.BatchGetItemOutput
	queryOut      []map[string]types.AttributeValue
	deleted       *dynamodb.DeleteItemInput
}

func (api *fakeItemAPI) PutItem(_ context.Context, input *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	api.putInput = input
	return &dynamodb.PutItemOutput{}, nil
}

func (api *fakeItemAPI) GetItem(_ context.Context, input *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	_ = input
	return &dynamodb.GetItemOutput{
		Item: api.getItem,
	}, nil
}

func (api *fakeItemAPI) BatchGetItem(_ context.Context, input *dynamodb.BatchGetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error) {
	api.batchGetInput = input
	api.batchGetCalls++
	if api.batchGetOut != nil {
		return api.batchGetOut, nil
	}
	return &dynamodb.BatchGetItemOutput{}, nil
}

func (api *fakeItemAPI) DeleteItem(_ context.Context, input *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	api.deleted = input
	return &dynamodb.DeleteItemOutput{}, nil
}

func (api *fakeItemAPI) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{Items: api.queryOut}, nil
}

func (api *fakeItemAPI) ListTables(_ context.Context, _ *dynamodb.ListTablesInput, _ ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return &dynamodb.ListTablesOutput{TableNames: []string{"audit-events"}}, nil
}

func TestMarshalShape(t *testing.T) {
	t.Parallel()

	item, err := attributevalue.MarshalMap(struct {
		PK string `dynamodbav:"pk"`
	}{PK: "tenant#acme"})
	if err != nil {
		t.Fatalf("MarshalMap returned error: %v", err)
	}
	if got := item["pk"].(*types.AttributeValueMemberS).Value; got != "tenant#acme" {
		t.Fatalf("unexpected pk: %q", got)
	}
}

type fakePagingAPI struct {
	fakeItemAPI
	pages []dynamodb.QueryOutput
	idx   int
}

func (f *fakePagingAPI) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if f.idx >= len(f.pages) {
		return &dynamodb.QueryOutput{}, nil
	}
	out := f.pages[f.idx]
	f.idx++
	return &out, nil
}

func TestQueryJSONAll_Paginates(t *testing.T) {
	t.Parallel()

	pk := "asset-1"
	item1, err := attributevalue.MarshalMap(struct {
		AssetID string `dynamodbav:"assetId"`
		SK      string `dynamodbav:"sk"`
	}{AssetID: pk, SK: "ART#001#a"})
	if err != nil {
		t.Fatal(err)
	}
	item2, err := attributevalue.MarshalMap(struct {
		AssetID string `dynamodbav:"assetId"`
		SK      string `dynamodbav:"sk"`
	}{AssetID: pk, SK: "ART#002#b"})
	if err != nil {
		t.Fatal(err)
	}
	lek := map[string]types.AttributeValue{
		"assetId": &types.AttributeValueMemberS{Value: pk},
		"sk":      &types.AttributeValueMemberS{Value: "ART#001#a"},
	}
	api := &fakePagingAPI{
		pages: []dynamodb.QueryOutput{
			{Items: []map[string]types.AttributeValue{item1}, LastEvaluatedKey: lek},
			{Items: []map[string]types.AttributeValue{item2}},
		},
	}
	client := NewWithAPI(api, "artifacts")

	var out []struct {
		AssetID string `dynamodbav:"assetId"`
		SK      string `dynamodbav:"sk"`
	}
	if err := client.QueryJSONAll(context.Background(), &dynamodb.QueryInput{}, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].SK != "ART#001#a" || out[1].SK != "ART#002#b" {
		t.Fatalf("%+v", out)
	}
}

func TestQueryJSONPage_ReturnsOpaqueCursor(t *testing.T) {
	t.Parallel()

	item, err := attributevalue.MarshalMap(struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}{UserID: "user-1", SK: "ASSET#001"})
	if err != nil {
		t.Fatal(err)
	}
	lek := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: "user-1"},
		"sk":     &types.AttributeValueMemberS{Value: "ASSET#001"},
	}
	api := &fakePagingAPI{
		pages: []dynamodb.QueryOutput{
			{Items: []map[string]types.AttributeValue{item}, LastEvaluatedKey: lek},
		},
	}
	client := NewWithAPI(api, "assets")

	var out []struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}
	page, err := client.QueryJSONPage(context.Background(), &dynamodb.QueryInput{}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore || page.NextCursor == "" {
		t.Fatalf("expected next cursor, got %+v", page)
	}
	decoded, err := DecodeCursor(page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if got := decoded["sk"].(*types.AttributeValueMemberS).Value; got != "ASSET#001" {
		t.Fatalf("unexpected decoded sk: %q", got)
	}
}

func TestCursorRoundTripSupportsScalarKeys(t *testing.T) {
	t.Parallel()

	key := map[string]types.AttributeValue{
		"s": &types.AttributeValueMemberS{Value: "value"},
		"n": &types.AttributeValueMemberN{Value: "42"},
		"b": &types.AttributeValueMemberB{Value: []byte{1, 2, 3}},
		"x": &types.AttributeValueMemberBOOL{Value: true},
	}
	cursor, err := EncodeCursor(key)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCursor(cursor)
	if err != nil {
		t.Fatal(err)
	}
	if decoded["s"].(*types.AttributeValueMemberS).Value != "value" {
		t.Fatalf("unexpected string key: %#v", decoded["s"])
	}
	if decoded["n"].(*types.AttributeValueMemberN).Value != "42" {
		t.Fatalf("unexpected number key: %#v", decoded["n"])
	}
	if string(decoded["b"].(*types.AttributeValueMemberB).Value) != string([]byte{1, 2, 3}) {
		t.Fatalf("unexpected binary key: %#v", decoded["b"])
	}
	if !decoded["x"].(*types.AttributeValueMemberBOOL).Value {
		t.Fatalf("unexpected bool key: %#v", decoded["x"])
	}
}

func TestDecodeCursorRejectsInvalidCursor(t *testing.T) {
	t.Parallel()

	_, err := DecodeCursor("not a cursor")
	if !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestBatchGetJSON(t *testing.T) {
	t.Parallel()

	item1, err := attributevalue.MarshalMap(struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}{UserID: "user-1", SK: "ASSET#001"})
	if err != nil {
		t.Fatal(err)
	}
	item2, err := attributevalue.MarshalMap(struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}{UserID: "user-1", SK: "ASSET#002"})
	if err != nil {
		t.Fatal(err)
	}
	api := &fakeItemAPI{
		batchGetOut: &dynamodb.BatchGetItemOutput{
			Responses: map[string][]map[string]types.AttributeValue{
				"assets": {item1, item2},
			},
		},
	}
	client := NewWithAPI(api, "assets")

	keys := []any{
		struct {
			UserID string `dynamodbav:"userId"`
			SK     string `dynamodbav:"sk"`
		}{UserID: "user-1", SK: "ASSET#001"},
		struct {
			UserID string `dynamodbav:"userId"`
			SK     string `dynamodbav:"sk"`
		}{UserID: "user-1", SK: "ASSET#002"},
	}
	var out []struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}
	if err := client.BatchGetJSON(context.Background(), keys, &out); err != nil {
		t.Fatal(err)
	}
	if api.batchGetInput == nil || len(api.batchGetInput.RequestItems["assets"].Keys) != 2 {
		t.Fatalf("unexpected batch input: %#v", api.batchGetInput)
	}
	if len(out) != 2 || out[0].SK != "ASSET#001" || out[1].SK != "ASSET#002" {
		t.Fatalf("unexpected batch output: %+v", out)
	}
}

func TestBatchGetJSONSplitsLargeKeySets(t *testing.T) {
	t.Parallel()

	api := &fakeItemAPI{
		batchGetOut: &dynamodb.BatchGetItemOutput{},
	}
	client := NewWithAPI(api, "assets")
	keys := make([]any, 205)
	for i := range keys {
		keys[i] = struct {
			UserID string `dynamodbav:"userId"`
			SK     string `dynamodbav:"sk"`
		}{UserID: "user-1", SK: fmt.Sprintf("ASSET#%03d", i)}
	}
	var out []struct {
		UserID string `dynamodbav:"userId"`
		SK     string `dynamodbav:"sk"`
	}
	if err := client.BatchGetJSON(context.Background(), keys, &out); err != nil {
		t.Fatal(err)
	}
	if api.batchGetCalls != 3 {
		t.Fatalf("expected 3 batch get calls, got %d", api.batchGetCalls)
	}
	if got := len(api.batchGetInput.RequestItems["assets"].Keys); got != 5 {
		t.Fatalf("expected final chunk with 5 keys, got %d", got)
	}
}

func TestDeleteQueryAndHealth(t *testing.T) {
	t.Parallel()

	api := &fakeItemAPI{}
	client := NewWithAPI(api, "audit-events")

	if err := client.DeleteJSON(context.Background(), struct {
		PK string `dynamodbav:"pk"`
	}{PK: "tenant#acme"}); err != nil {
		t.Fatalf("DeleteJSON returned error: %v", err)
	}
	if api.deleted == nil {
		t.Fatal("expected delete input")
	}

	queryItems, err := attributevalue.MarshalList([]struct {
		PK string `dynamodbav:"pk"`
	}{
		{PK: "tenant#acme"},
	})
	if err != nil {
		t.Fatalf("MarshalList returned error: %v", err)
	}
	api.queryOut = []map[string]types.AttributeValue{
		queryItems[0].(*types.AttributeValueMemberM).Value,
	}
	var out []struct {
		PK string `dynamodbav:"pk"`
	}
	if err := client.QueryJSON(context.Background(), &dynamodb.QueryInput{}, &out); err != nil {
		t.Fatalf("QueryJSON returned error: %v", err)
	}
	if len(out) != 1 || out[0].PK != "tenant#acme" {
		t.Fatalf("unexpected query output: %#v", out)
	}

	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
}
