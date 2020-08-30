package lambdautils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/pkg/errors"
)

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

	nowFunc func() time.Time
	svcFunc func(client.ConfigProvider) dynamodbiface.DynamoDBAPI
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

	err := json.Unmarshal([]byte(s), lock)
	if err != nil {
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

// now is used internally to assist stubs on time.Now() for testing
func (lock *SNSLock) now() time.Time {
	if lock.nowFunc != nil {
		return lock.nowFunc()
	}

	return time.Now()
}

// svc is used internally to assist stubs on dynamodb for testing
func (lock *SNSLock) svc(p client.ConfigProvider) dynamodbiface.DynamoDBAPI {
	if lock.svcFunc != nil {
		return lock.svcFunc(p)
	}

	return dynamodb.New(p)
}

// messageHash returns the sha256 of the message embedded in the sns event
func (lock *SNSLock) messageHash(snsEvent events.SNSEvent) string {
	message := snsEvent.Records[0].SNS.Message
	sum := sha256.Sum256([]byte(message))
	return fmt.Sprintf("%x", sum)
}

// expires returns the current time + ttl in Epoch format as a string
func (lock *SNSLock) expires() string {
	d := time.Duration(lock.TTL) * time.Second
	t := lock.now().Add(d).Unix()
	return strconv.FormatInt(t, 10)
}

// current returns the current time in Epoch format as a string
func (lock *SNSLock) current() string {
	return strconv.FormatInt(lock.now().Unix(), 10)
}

// putItemInput constructs the input for the given id insertion into dynamodb.
// It applies a conditional expression that causes failures when the id has
// already been added but not yet expired.
func (lock *SNSLock) putItemInput(id string) *dynamodb.PutItemInput {
	condition := "attribute_not_exists(id) OR :cur > expire"

	return &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
			"expire": {
				N: aws.String(lock.expires()),
			},
		},
		TableName:           aws.String(lock.Table),
		ConditionExpression: aws.String(condition),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cur": {
				N: aws.String(lock.current()),
			},
		},
	}
}

// AvailableById returns true if the given id is available for use (not locked)
// and it returns false if it is locked.
//
// Locked is defined as the record being in the configured dynamodb table and
// not expires.
func (lock *SNSLock) AvailableById(id string) (bool, error) {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(lock.Region),
	})

	if err != nil {
		return false, errors.Wrap(err, "failed getting session")
	}

	svc := lock.svc(s)
	input := lock.putItemInput(id)

	for attempts := 1; attempts <= 12; attempts++ {
		_, err = svc.PutItem(input)
		if err == nil {
			break
		}
		errString := err.Error()
		if strings.Contains(errString, "connection reset by peer") {
			time.Sleep(time.Duration(lock.TTL) * time.Millisecond)
			continue // retry
		}
		break
	}

	if err == nil {
		return true, nil
	}

	aerr, ok := err.(awserr.Error)
	if ok && aerr.Code() == dynamodb.ErrCodeConditionalCheckFailedException {
		return false, nil
	}

	return false, errors.Wrapf(err, "failed put %v to %v", id, lock.Table)
}

// Available returns true if the snsEvent is available for use (not locked) and
// it returns false if it is locked.
//
// Locked is defined as the record being in the configured dynamodb table and
// not expires.
func (lock *SNSLock) Available(snsEvent events.SNSEvent) (bool, error) {
	if len(snsEvent.Records) != 1 {
		return false, fmt.Errorf("expected only 1 SNS event, received: %v", len(snsEvent.Records))
	}

	id := lock.messageHash(snsEvent)
	return lock.AvailableById(id)
}
