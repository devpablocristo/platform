package idempotency

import (
	"context"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// fakeDDB simula la superficie mínima de DynamoDB para el state machine.
type fakeDDB struct {
	putErr     error // devuelto por PutItem (p.ej. CCFE para simular conflicto)
	existing   *row  // fila devuelta por GetItem (nil = no existe)
	updateErr  error
	puts       int
	updates    int
	lastUpdate *dynamodb.UpdateItemInput
}

func (f *fakeDDB) PutItem(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.puts++
	return &dynamodb.PutItemOutput{}, f.putErr
}

func (f *fakeDDB) GetItem(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if f.existing == nil {
		return &dynamodb.GetItemOutput{}, nil
	}
	item, _ := attributevalue.MarshalMap(*f.existing)
	item["jobId"] = &types.AttributeValueMemberS{Value: "j1"}
	return &dynamodb.GetItemOutput{Item: item}, nil
}

func (f *fakeDDB) UpdateItem(_ context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updates++
	f.lastUpdate = in
	return &dynamodb.UpdateItemOutput{}, f.updateErr
}

var ccfe = &types.ConditionalCheckFailedException{Message: awssdk.String("conditional check failed")}

func store(f *fakeDDB) *DynamoStore { return NewWithAPI(f, "medmory-v2-processed-jobs", "jobId") }

func TestClaimFirstTimeSucceeds(t *testing.T) {
	f := &fakeDDB{}
	ok, err := store(f).Claim(context.Background(), Claim{Key: "j1"})
	if err != nil || !ok {
		t.Fatalf("primer claim debería ser true; ok=%v err=%v", ok, err)
	}
	if f.puts != 1 {
		t.Fatalf("esperaba 1 PutItem, got %d", f.puts)
	}
}

func TestClaimAlreadyCompletedReturnsFalse(t *testing.T) {
	f := &fakeDDB{putErr: ccfe, existing: &row{Status: statusCompleted}}
	ok, err := store(f).Claim(context.Background(), Claim{Key: "j1"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("job completado no debería re-reclamarse")
	}
}

func TestClaimInProgressWithLiveLease(t *testing.T) {
	future := time.Now().UTC().Add(5 * time.Minute).Unix()
	f := &fakeDDB{putErr: ccfe, existing: &row{Status: statusInProgress, LeaseUntil: future}}
	_, err := store(f).Claim(context.Background(), Claim{Key: "j1"})
	if err != ErrInProgress {
		t.Fatalf("esperaba ErrInProgress, got %v", err)
	}
}

func TestClaimReclaimsFailed(t *testing.T) {
	f := &fakeDDB{putErr: ccfe, existing: &row{Status: statusFailed}}
	ok, err := store(f).Claim(context.Background(), Claim{Key: "j1"})
	if err != nil || !ok {
		t.Fatalf("job failed debería re-reclamarse; ok=%v err=%v", ok, err)
	}
	if f.updates != 1 {
		t.Fatalf("esperaba 1 UpdateItem (reclaim), got %d", f.updates)
	}
}

func TestClaimReclaimsStaleLease(t *testing.T) {
	past := time.Now().UTC().Add(-5 * time.Minute).Unix()
	f := &fakeDDB{putErr: ccfe, existing: &row{Status: statusInProgress, LeaseUntil: past}}
	ok, err := store(f).Claim(context.Background(), Claim{Key: "j1"})
	if err != nil || !ok {
		t.Fatalf("lease vencido debería re-reclamarse; ok=%v err=%v", ok, err)
	}
}

func TestCompleteAndFail(t *testing.T) {
	f := &fakeDDB{}
	s := store(f)
	if err := s.Complete(context.Background(), "j1", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Fail(context.Background(), "j1", "boom", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if f.updates != 2 {
		t.Fatalf("esperaba 2 updates, got %d", f.updates)
	}
}
