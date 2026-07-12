// Package idempotency provee un store de idempotencia con lease sobre DynamoDB,
// pensado para consumidores de colas (SQS/otros) que deben procesar cada mensaje
// exactamente una vez pese a reentregas.
//
// Modelo: cada intento se identifica por una Key (p.ej. el jobId del mensaje).
// Claim reclama la key con un lease; devuelve false SOLO si ya se completó. Si
// otro worker la tiene in-progress con lease vigente, devuelve ErrInProgress
// para que la cola reintente más tarde. Un intento previo `failed` o con lease
// vencido se re-reclama (reclaim), permitiendo reintentar errores transitorios.
//
// La seguridad de "exactamente una vez" para EFECTOS (escrituras) NO debe
// depender solo de este store: un fallo entre la escritura y Complete puede
// reprocesar. Combinar con escrituras idempotentes por Key (condition-put) del
// lado del consumidor.
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	// DefaultLease debe ser mayor que el timeout del consumidor y menor que el
	// TTL, para poder reintentar jobs colgados tras el claim.
	DefaultLease = 10 * time.Minute
	// DefaultRetain debe superar la retención máxima de la cola (SQS: 14 días)
	// para que un retry tardío no reprocese.
	DefaultRetain = 15 * 24 * time.Hour

	statusInProgress = "in_progress"
	statusCompleted  = "completed"
	statusFailed     = "failed"
)

// ErrInProgress indica que la key la tiene otro worker con lease vigente.
var ErrInProgress = errors.New("idempotency: key already in progress")

// Claim describe un intento de reclamar una key.
type Claim struct {
	Key       string
	Now       time.Time     // zero → time.Now().UTC()
	LeaseFor  time.Duration // <=0 → DefaultLease
	RetainFor time.Duration // <=0 → DefaultRetain
}

// Store es la superficie durable de idempotencia con lease.
type Store interface {
	// Claim reclama la key. true = se puede procesar; false = ya completada.
	// ErrInProgress = otro worker la tiene con lease vigente.
	Claim(ctx context.Context, claim Claim) (bool, error)
	Complete(ctx context.Context, key string, now time.Time) error
	Fail(ctx context.Context, key, reason string, now time.Time) error
}

// dynamoAPI es la superficie mínima de DynamoDB que usa el store (la satisface
// *dynamodb.Client; permite fakes en test).
type dynamoAPI interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

// DynamoStore implementa Store sobre DynamoDB. `keyAttr` es el nombre del
// atributo partition-key de la tabla (p.ej. "jobId").
type DynamoStore struct {
	client  dynamoAPI
	table   string
	keyAttr string
}

// NewDynamoStore adapta un *dynamodb.Client + tabla a Store. Si keyAttr es "",
// usa "key".
func NewDynamoStore(client *dynamodb.Client, table, keyAttr string) *DynamoStore {
	return NewWithAPI(client, table, keyAttr)
}

// NewWithAPI construye el store desde una implementación inyectada (para test).
func NewWithAPI(api dynamoAPI, table, keyAttr string) *DynamoStore {
	if strings.TrimSpace(keyAttr) == "" {
		keyAttr = "key"
	}
	return &DynamoStore{client: api, table: strings.TrimSpace(table), keyAttr: keyAttr}
}

// row son los atributos internos (sin la partition key, que es dinámica).
type row struct {
	Status     string `dynamodbav:"status"`
	ClaimedAt  string `dynamodbav:"claimedAt,omitempty"`
	UpdatedAt  string `dynamodbav:"updatedAt,omitempty"`
	FailReason string `dynamodbav:"failReason,omitempty"`
	LeaseUntil int64  `dynamodbav:"leaseUntil,omitempty"`
	ExpiresAt  int64  `dynamodbav:"expiresAt"`
	Attempts   int64  `dynamodbav:"attempts,omitempty"`
}

func (s *DynamoStore) keyItem(key string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		s.keyAttr: &types.AttributeValueMemberS{Value: key},
	}
}

func (s *DynamoStore) Claim(ctx context.Context, claim Claim) (bool, error) {
	if s == nil || s.client == nil {
		return false, fmt.Errorf("idempotency client is nil")
	}
	if strings.TrimSpace(s.table) == "" {
		return false, fmt.Errorf("idempotency table is required")
	}
	if strings.TrimSpace(claim.Key) == "" {
		return false, fmt.Errorf("idempotency key is required")
	}
	now := normalizeTime(claim.Now)
	lease := claim.LeaseFor
	if lease <= 0 {
		lease = DefaultLease
	}
	retain := claim.RetainFor
	if retain <= 0 {
		retain = DefaultRetain
	}

	item, err := attributevalue.MarshalMap(row{
		Status:     statusInProgress,
		ClaimedAt:  now.Format(time.RFC3339),
		UpdatedAt:  now.Format(time.RFC3339),
		LeaseUntil: now.Add(lease).Unix(),
		ExpiresAt:  now.Add(retain).Unix(),
		Attempts:   1,
	})
	if err != nil {
		return false, fmt.Errorf("marshal idempotency row: %w", err)
	}
	item[s.keyAttr] = &types.AttributeValueMemberS{Value: claim.Key}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                awssdk.String(s.table),
		Item:                     item,
		ConditionExpression:      awssdk.String("attribute_not_exists(#k)"),
		ExpressionAttributeNames: map[string]string{"#k": s.keyAttr},
	})
	if err == nil {
		return true, nil
	}
	var ccfe *types.ConditionalCheckFailedException
	if !errors.As(err, &ccfe) {
		return false, fmt.Errorf("put idempotency claim: %w", err)
	}

	existing, err := s.getRow(ctx, claim.Key)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, fmt.Errorf("idempotency conflict without row for key %q", claim.Key)
	}
	if existing.Status == "" || existing.Status == statusCompleted {
		return false, nil
	}
	stale := existing.LeaseUntil > 0 && existing.LeaseUntil < now.Unix()
	if existing.Status != statusFailed && !stale {
		return false, ErrInProgress
	}
	if err := s.reclaim(ctx, claim.Key, now, lease, retain); err != nil {
		return false, err
	}
	return true, nil
}

func (s *DynamoStore) Complete(ctx context.Context, key string, now time.Time) error {
	now = normalizeTime(now)
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        awssdk.String(s.table),
		Key:              s.keyItem(key),
		UpdateExpression: awssdk.String("SET #s = :completed, #completedAt = :now, #updatedAt = :now REMOVE #lease"),
		ExpressionAttributeNames: map[string]string{
			"#s":           "status",
			"#completedAt": "completedAt",
			"#updatedAt":   "updatedAt",
			"#lease":       "leaseUntil",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":completed": &types.AttributeValueMemberS{Value: statusCompleted},
			":now":       &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}
	return nil
}

func (s *DynamoStore) Fail(ctx context.Context, key, reason string, now time.Time) error {
	now = normalizeTime(now)
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        awssdk.String(s.table),
		Key:              s.keyItem(key),
		UpdateExpression: awssdk.String("SET #s = :failed, #failedAt = :now, #updatedAt = :now, #reason = :reason REMOVE #lease"),
		ExpressionAttributeNames: map[string]string{
			"#s":         "status",
			"#failedAt":  "failedAt",
			"#updatedAt": "updatedAt",
			"#reason":    "failReason",
			"#lease":     "leaseUntil",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":failed": &types.AttributeValueMemberS{Value: statusFailed},
			":now":    &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":reason": &types.AttributeValueMemberS{Value: truncate(reason, 500)},
		},
	})
	if err != nil {
		return fmt.Errorf("fail idempotency key: %w", err)
	}
	return nil
}

func (s *DynamoStore) getRow(ctx context.Context, key string) (*row, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: awssdk.String(s.table),
		Key:       s.keyItem(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get idempotency row: %w", err)
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	var r row
	if err := attributevalue.UnmarshalMap(out.Item, &r); err != nil {
		return nil, fmt.Errorf("unmarshal idempotency row: %w", err)
	}
	return &r, nil
}

func (s *DynamoStore) reclaim(ctx context.Context, key string, now time.Time, lease, retain time.Duration) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:           awssdk.String(s.table),
		Key:                 s.keyItem(key),
		UpdateExpression:    awssdk.String("SET #s = :inProgress, #claimedAt = :now, #updatedAt = :now, #lease = :lease, #expires = :expires REMOVE #failedAt, #reason ADD #attempts :one"),
		ConditionExpression: awssdk.String("#s = :failed OR attribute_not_exists(#lease) OR #lease < :nowUnix"),
		ExpressionAttributeNames: map[string]string{
			"#s":         "status",
			"#claimedAt": "claimedAt",
			"#updatedAt": "updatedAt",
			"#lease":     "leaseUntil",
			"#expires":   "expiresAt",
			"#failedAt":  "failedAt",
			"#reason":    "failReason",
			"#attempts":  "attempts",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inProgress": &types.AttributeValueMemberS{Value: statusInProgress},
			":failed":     &types.AttributeValueMemberS{Value: statusFailed},
			":now":        &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":lease":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Add(lease).Unix())},
			":expires":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Add(retain).Unix())},
			":nowUnix":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
			":one":        &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			return ErrInProgress
		}
		return fmt.Errorf("reclaim idempotency key: %w", err)
	}
	return nil
}

func normalizeTime(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
