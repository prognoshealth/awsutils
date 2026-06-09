package lambdautils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// dynamoDBPutItemAPI is the minimal DynamoDB interface needed by SNSLock.
type dynamoDBPutItemAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

// SNSLock manages locking of sns messages using dynamodb. The SNS messages are
// locked using the hash of their message contents and the lock expires after
// the TTL (seconds) has expired.
//
// RetryWait (milliseconds) is used to manage retry backoff times.
type SNSLock struct {
	Region    string `json:"region"`
	Table     string `json:"table"`
	TTL       int64  `json:"ttl"`
	RetryWait int64  `json:"retry-wait"`

	nowFunc  func() time.Time
	svcFunc  func(aws.Config) dynamoDBPutItemAPI
	hashFunc func(string) (string, error)
}

// NewSNSLock returns a new sns lock instance to manage dynamodb locking
func NewSNSLock(region string, table string, ttl int64, retry int64) *SNSLock {
	lock := new(SNSLock)
	lock.Region = region
	lock.Table = table
	lock.TTL = ttl
	lock.RetryWait = retry

	if lock.TTL == 0 {
		lock.TTL = 300
	}

	if lock.RetryWait == 0 {
		lock.RetryWait = 500
	}

	return lock
}

// NewSNSLockFromJson returns a new sns lock instance to manage dynamodb locking
func NewSNSLockFromJson(s string) (*SNSLock, error) {
	lock := new(SNSLock)

	if err := json.Unmarshal([]byte(s), lock); err != nil {
		return nil, err
	}

	if lock.Region == "" {
		return nil, errors.New("region is required")
	}

	if lock.Table == "" {
		return nil, errors.New("table is required")
	}

	if lock.TTL == 0 {
		lock.TTL = 300
	}

	if lock.RetryWait == 0 {
		lock.RetryWait = 500
	}

	return lock, nil
}

func (lock *SNSLock) now() time.Time {
	if lock.nowFunc != nil {
		return lock.nowFunc()
	}
	return time.Now()
}

func (lock *SNSLock) svc(cfg aws.Config) dynamoDBPutItemAPI {
	if lock.svcFunc != nil {
		return lock.svcFunc(cfg)
	}
	return dynamodb.NewFromConfig(cfg)
}

func (lock *SNSLock) messageHash(snsEvent events.SNSEvent) (string, error) {
	message := snsEvent.Records[0].SNS.Message

	if lock.hashFunc != nil {
		return lock.hashFunc(message)
	}

	sum := sha256.Sum256([]byte(message))
	return fmt.Sprintf("%x", sum), nil
}

func (lock *SNSLock) expires() string {
	d := time.Duration(lock.TTL) * time.Second
	t := lock.now().Add(d).Unix()
	return strconv.FormatInt(t, 10)
}

func (lock *SNSLock) current() string {
	return strconv.FormatInt(lock.now().Unix(), 10)
}

func (lock *SNSLock) putItemInput(id string) *dynamodb.PutItemInput {
	condition := "attribute_not_exists(id) OR :cur > expire"

	return &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"id":     &types.AttributeValueMemberS{Value: id},
			"expire": &types.AttributeValueMemberN{Value: lock.expires()},
		},
		TableName:           aws.String(lock.Table),
		ConditionExpression: aws.String(condition),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cur": &types.AttributeValueMemberN{Value: lock.current()},
		},
	}
}

// AvailableById returns true if the given id is available for use (not locked)
// and it returns false if it is locked.
//
// Locked is defined as the record being in the configured dynamodb table and
// not expired.
func (lock *SNSLock) AvailableById(id string) (bool, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(lock.Region),
	)
	if err != nil {
		return false, fmt.Errorf("failed loading config: %w", err)
	}

	svc := lock.svc(cfg)
	input := lock.putItemInput(id)

	for attempts := 1; attempts <= 12; attempts++ {
		_, err = svc.PutItem(context.Background(), input)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "connection reset by peer") {
			time.Sleep(time.Duration(lock.TTL) * time.Millisecond)
			continue
		}
		break
	}

	if err == nil {
		return true, nil
	}

	var condErr *types.ConditionalCheckFailedException
	if errors.As(err, &condErr) {
		return false, nil
	}

	return false, fmt.Errorf("failed put %v to %v: %w", id, lock.Table, err)
}

// Available returns true if the snsEvent is available for use (not locked) and
// it returns false if it is locked.
//
// Locked is defined as the record being in the configured dynamodb table and
// not expired.
func (lock *SNSLock) Available(snsEvent events.SNSEvent) (bool, error) {
	if len(snsEvent.Records) != 1 {
		return false, fmt.Errorf("expected only 1 SNS event, received: %v", len(snsEvent.Records))
	}

	id, err := lock.messageHash(snsEvent)
	if err != nil {
		return false, fmt.Errorf("failed to hash message: %w", err)
	}
	return lock.AvailableById(id)
}

// SetHashFunc sets the hash function to use for message hashing
func (lock *SNSLock) SetHashFunc(f func(string) (string, error)) {
	lock.hashFunc = f
}
