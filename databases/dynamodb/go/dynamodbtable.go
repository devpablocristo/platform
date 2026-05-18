package dynamodbtable

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Config representa la configuración mínima para una tabla DynamoDB.
type Config struct {
	Region   string
	Endpoint string
	Table    string
}

type itemAPI interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	BatchGetItem(context.Context, *dynamodb.BatchGetItemInput, ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	ListTables(context.Context, *dynamodb.ListTablesInput, ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
}

// ErrInvalidCursor indica que un cursor recibido del cliente no puede decodificarse.
var ErrInvalidCursor = errors.New("invalid dynamodb cursor")

// Page describe una página DynamoDB reusable con cursor opaco.
type Page struct {
	HasMore    bool
	NextCursor string
}

// Client envuelve acceso reusable a una tabla fija.
type Client struct {
	api   itemAPI
	table string
}

// ConfigFromEnv carga configuración desde env.
func ConfigFromEnv(prefix string) Config {
	prefix = normalizeEnvPrefix(prefix)
	if prefix == "" {
		return Config{
			Region:   firstNonEmpty(os.Getenv("AWS_REGION"), "us-east-1"),
			Endpoint: firstNonEmpty(os.Getenv("DYNAMODB_ENDPOINT")),
			Table:    firstNonEmpty(os.Getenv("DYNAMODB_TABLE")),
		}
	}
	return Config{
		Region:   firstNonEmpty(os.Getenv(prefix+"AWS_REGION"), os.Getenv("AWS_REGION"), "us-east-1"),
		Endpoint: firstNonEmpty(os.Getenv(prefix+"ENDPOINT"), os.Getenv("DYNAMODB_ENDPOINT")),
		Table:    firstNonEmpty(os.Getenv(prefix+"TABLE"), os.Getenv("DYNAMODB_TABLE")),
	}
}

// New crea un cliente DynamoDB desde configuración local.
func New(ctx context.Context, config Config) (*Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(config.Region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return NewFromAWSConfig(awsCfg, config), nil
}

// NewFromAWSConfig crea un cliente reutilizando `aws.Config`.
func NewFromAWSConfig(awsCfg awssdk.Config, config Config) *Client {
	api := dynamodb.NewFromConfig(awsCfg, func(options *dynamodb.Options) {
		if strings.TrimSpace(config.Endpoint) != "" {
			options.BaseEndpoint = awssdk.String(strings.TrimSpace(config.Endpoint))
		}
	})
	return NewWithAPI(api, config.Table)
}

// NewWithAPI crea un cliente desde un adapter inyectado.
func NewWithAPI(api itemAPI, table string) *Client {
	return &Client{
		api:   api,
		table: strings.TrimSpace(table),
	}
}

// Table devuelve el nombre fijo de la tabla.
func (c *Client) Table() string {
	if c == nil {
		return ""
	}
	return c.table
}

// PutItem guarda un item raw.
func (c *Client) PutItem(ctx context.Context, item map[string]types.AttributeValue) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if strings.TrimSpace(c.table) == "" {
		return fmt.Errorf("dynamodb table is required")
	}
	if len(item) == 0 {
		return fmt.Errorf("dynamodb item is required")
	}
	_, err := c.api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: awssdk.String(c.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put dynamodb item: %w", err)
	}
	return nil
}

// PutJSON serializa una estructura a AttributeValue map y la persiste.
func (c *Client) PutJSON(ctx context.Context, value any) error {
	item, err := attributevalue.MarshalMap(value)
	if err != nil {
		return fmt.Errorf("marshal dynamodb item: %w", err)
	}
	return c.PutItem(ctx, item)
}

// GetJSON resuelve una key tipada y deserializa el item si existe.
func (c *Client) GetJSON(ctx context.Context, key any, out any) (bool, error) {
	if c == nil || c.api == nil {
		return false, fmt.Errorf("dynamodb client is nil")
	}
	if strings.TrimSpace(c.table) == "" {
		return false, fmt.Errorf("dynamodb table is required")
	}

	marshaledKey, err := attributevalue.MarshalMap(key)
	if err != nil {
		return false, fmt.Errorf("marshal dynamodb key: %w", err)
	}
	output, err := c.api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: awssdk.String(c.table),
		Key:       marshaledKey,
	})
	if err != nil {
		return false, fmt.Errorf("get dynamodb item: %w", err)
	}
	if len(output.Item) == 0 {
		return false, nil
	}
	if err := attributevalue.UnmarshalMap(output.Item, out); err != nil {
		return false, fmt.Errorf("unmarshal dynamodb item: %w", err)
	}
	return true, nil
}

// BatchGetJSON resuelve varias keys primarias y deserializa los items encontrados.
func (c *Client) BatchGetJSON(ctx context.Context, keys []any, out any) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if strings.TrimSpace(c.table) == "" {
		return fmt.Errorf("dynamodb table is required")
	}
	if len(keys) == 0 {
		if err := attributevalue.UnmarshalListOfMaps(nil, out); err != nil {
			return fmt.Errorf("unmarshal dynamodb batch items: %w", err)
		}
		return nil
	}

	marshaledKeys := make([]map[string]types.AttributeValue, 0, len(keys))
	for _, key := range keys {
		marshaledKey, err := attributevalue.MarshalMap(key)
		if err != nil {
			return fmt.Errorf("marshal dynamodb batch key: %w", err)
		}
		marshaledKeys = append(marshaledKeys, marshaledKey)
	}

	var allItems []map[string]types.AttributeValue
	for start := 0; start < len(marshaledKeys); start += 100 {
		end := start + 100
		if end > len(marshaledKeys) {
			end = len(marshaledKeys)
		}
		requestItems := map[string]types.KeysAndAttributes{
			c.table: {Keys: marshaledKeys[start:end]},
		}
		for len(requestItems) > 0 {
			output, err := c.api.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
				RequestItems: requestItems,
			})
			if err != nil {
				return fmt.Errorf("batch get dynamodb items: %w", err)
			}
			allItems = append(allItems, output.Responses[c.table]...)
			requestItems = output.UnprocessedKeys
		}
	}
	if err := attributevalue.UnmarshalListOfMaps(allItems, out); err != nil {
		return fmt.Errorf("unmarshal dynamodb batch items: %w", err)
	}
	return nil
}

// DeleteJSON borra un item por key tipada.
func (c *Client) DeleteJSON(ctx context.Context, key any) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if strings.TrimSpace(c.table) == "" {
		return fmt.Errorf("dynamodb table is required")
	}

	marshaledKey, err := attributevalue.MarshalMap(key)
	if err != nil {
		return fmt.Errorf("marshal dynamodb key: %w", err)
	}
	if _, err := c.api.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: awssdk.String(c.table),
		Key:       marshaledKey,
	}); err != nil {
		return fmt.Errorf("delete dynamodb item: %w", err)
	}
	return nil
}

// QueryJSON ejecuta una query y deserializa todos los items.
func (c *Client) QueryJSON(ctx context.Context, input *dynamodb.QueryInput, out any) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if input == nil {
		return fmt.Errorf("dynamodb query input is required")
	}
	query := *input
	if query.TableName == nil || strings.TrimSpace(awssdk.ToString(query.TableName)) == "" {
		query.TableName = awssdk.String(c.table)
	}
	output, err := c.api.Query(ctx, &query)
	if err != nil {
		return fmt.Errorf("query dynamodb items: %w", err)
	}
	if err := attributevalue.UnmarshalListOfMaps(output.Items, out); err != nil {
		return fmt.Errorf("unmarshal dynamodb items: %w", err)
	}
	return nil
}

// QueryJSONPage ejecuta una query y devuelve el cursor para la siguiente página si existe.
func (c *Client) QueryJSONPage(ctx context.Context, input *dynamodb.QueryInput, out any) (Page, error) {
	if c == nil || c.api == nil {
		return Page{}, fmt.Errorf("dynamodb client is nil")
	}
	if input == nil {
		return Page{}, fmt.Errorf("dynamodb query input is required")
	}
	query := *input
	if query.TableName == nil || strings.TrimSpace(awssdk.ToString(query.TableName)) == "" {
		query.TableName = awssdk.String(c.table)
	}
	output, err := c.api.Query(ctx, &query)
	if err != nil {
		return Page{}, fmt.Errorf("query dynamodb items: %w", err)
	}
	if err := attributevalue.UnmarshalListOfMaps(output.Items, out); err != nil {
		return Page{}, fmt.Errorf("unmarshal dynamodb items: %w", err)
	}
	nextCursor, err := EncodeCursor(output.LastEvaluatedKey)
	if err != nil {
		return Page{}, err
	}
	return Page{
		HasMore:    nextCursor != "",
		NextCursor: nextCursor,
	}, nil
}

// QueryJSONAll ejecuta la misma query que QueryJSON pero sigue LastEvaluatedKey hasta agotar resultados.
func (c *Client) QueryJSONAll(ctx context.Context, input *dynamodb.QueryInput, out any) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if input == nil {
		return fmt.Errorf("dynamodb query input is required")
	}
	qi := *input
	if qi.TableName == nil || strings.TrimSpace(awssdk.ToString(qi.TableName)) == "" {
		qi.TableName = awssdk.String(c.table)
	}
	var allItems []map[string]types.AttributeValue
	for {
		outPage, err := c.api.Query(ctx, &qi)
		if err != nil {
			return fmt.Errorf("query dynamodb items: %w", err)
		}
		allItems = append(allItems, outPage.Items...)
		if len(outPage.LastEvaluatedKey) == 0 {
			break
		}
		qi.ExclusiveStartKey = outPage.LastEvaluatedKey
	}
	if err := attributevalue.UnmarshalListOfMaps(allItems, out); err != nil {
		return fmt.Errorf("unmarshal dynamodb items: %w", err)
	}
	return nil
}

// Health verifica acceso básico al servicio DynamoDB.
func (c *Client) Health(ctx context.Context) error {
	if c == nil || c.api == nil {
		return fmt.Errorf("dynamodb client is nil")
	}
	if _, err := c.api.ListTables(ctx, &dynamodb.ListTablesInput{Limit: awssdk.Int32(1)}); err != nil {
		return fmt.Errorf("list dynamodb tables: %w", err)
	}
	return nil
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

type cursorValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// EncodeCursor convierte una LastEvaluatedKey DynamoDB en un cursor opaco URL-safe.
func EncodeCursor(key map[string]types.AttributeValue) (string, error) {
	if len(key) == 0 {
		return "", nil
	}
	payload := make(map[string]cursorValue, len(key))
	for name, value := range key {
		switch v := value.(type) {
		case *types.AttributeValueMemberS:
			payload[name] = cursorValue{Type: "S", Value: v.Value}
		case *types.AttributeValueMemberN:
			payload[name] = cursorValue{Type: "N", Value: v.Value}
		case *types.AttributeValueMemberB:
			payload[name] = cursorValue{Type: "B", Value: base64.StdEncoding.EncodeToString(v.Value)}
		case *types.AttributeValueMemberBOOL:
			if v.Value {
				payload[name] = cursorValue{Type: "BOOL", Value: "true"}
			} else {
				payload[name] = cursorValue{Type: "BOOL", Value: "false"}
			}
		default:
			return "", fmt.Errorf("%w: unsupported attribute %q", ErrInvalidCursor, name)
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal dynamodb cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeCursor restaura un cursor opaco en una ExclusiveStartKey DynamoDB.
func DecodeCursor(cursor string) (map[string]types.AttributeValue, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var payload map[string]cursorValue
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, ErrInvalidCursor
	}
	if len(payload) == 0 {
		return nil, nil
	}
	key := make(map[string]types.AttributeValue, len(payload))
	for name, value := range payload {
		switch value.Type {
		case "S":
			key[name] = &types.AttributeValueMemberS{Value: value.Value}
		case "N":
			key[name] = &types.AttributeValueMemberN{Value: value.Value}
		case "B":
			decoded, err := base64.StdEncoding.DecodeString(value.Value)
			if err != nil {
				return nil, ErrInvalidCursor
			}
			key[name] = &types.AttributeValueMemberB{Value: decoded}
		case "BOOL":
			switch value.Value {
			case "true":
				key[name] = &types.AttributeValueMemberBOOL{Value: true}
			case "false":
				key[name] = &types.AttributeValueMemberBOOL{Value: false}
			default:
				return nil, ErrInvalidCursor
			}
		default:
			return nil, ErrInvalidCursor
		}
	}
	return key, nil
}
