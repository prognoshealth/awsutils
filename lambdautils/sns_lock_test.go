package lambdautils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

func TestNewSNSLock(t *testing.T) {
	cases := []struct {
		ttl               int64
		retry             int64
		expectedTTL       int64
		expectedRetryWait int64
	}{
		{15, 30, 15, 30},
		{15, 0, 15, 500},
		{0, 30, 300, 30},
	}

	for _, c := range cases {
		l := NewSNSLock("r", "t", c.ttl, c.retry)

		assert.Equal(t, "r", l.Region)
		assert.Equal(t, "t", l.Table)
		assert.Equal(t, c.expectedTTL, l.TTL)
		assert.Equal(t, c.expectedRetryWait, l.RetryWait)
	}
}

func TestNewSNSLockFromJson(t *testing.T) {
	cases := []struct {
		json              string
		expectedRegion    string
		expectedTable     string
		expectedTTL       int64
		expectedRetryWait int64
	}{
		{`{"region": "r1", "table": "t1", "ttl": 15}`, "r1", "t1", 15, 500},
		{`{"region": "r2", "table": "t2", "ttl": 30}`, "r2", "t2", 30, 500},
		{`{"region": "r3", "table": "t3"}`, "r3", "t3", 300, 500},
		{`{"region": "r3", "table": "t3", "retry-wait": 250}`, "r3", "t3", 300, 250},
	}

	for _, c := range cases {
		l, err := NewSNSLockFromJson(c.json)
		assert.NoError(t, err)

		assert.Equal(t, c.expectedRegion, l.Region)
		assert.Equal(t, c.expectedTable, l.Table)
		assert.Equal(t, c.expectedTTL, l.TTL)
		assert.Equal(t, c.expectedRetryWait, l.RetryWait)
	}
}

func TestNewSNSLockFromJson_errorUnmarshal(t *testing.T) {
	json := `{...`
	_, err := NewSNSLockFromJson(json)
	assert.Error(t, err)
}

func TestNewSNSLockFromJson_errorRegion(t *testing.T) {
	json := `{"table": "t1", "ttl": 15}`
	_, err := NewSNSLockFromJson(json)
	assert.Error(t, err)
}

func TestNewSNSLockFromJson_errorTable(t *testing.T) {
	json := `{"region": "r2", "ttl": 30}`
	_, err := NewSNSLockFromJson(json)
	assert.Error(t, err)
}

func TestSNSLock_messageHash(t *testing.T) {
	b, err := os.ReadFile("testdata/valid_sns_string_event.json")
	assert.NoError(t, err)

	snsEventRecord := &events.SNSEventRecord{}
	assert.NoError(t, json.Unmarshal(b, snsEventRecord))

	snsEvent := events.SNSEvent{
		Records: []events.SNSEventRecord{
			*snsEventRecord,
		},
	}

	l := &SNSLock{}

	expected := "d2837a5c7d52bf9f472b16bd851d6c09579a80fe5e4fbf293a988c117ee90bb0"
	actual, err := l.messageHash(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSNSLock_messageHash_json(t *testing.T) {
	b, err := os.ReadFile("testdata/valid_sns_json_event.json")
	assert.NoError(t, err)

	snsEventRecord := &events.SNSEventRecord{}
	assert.NoError(t, json.Unmarshal(b, snsEventRecord))

	snsEvent := events.SNSEvent{
		Records: []events.SNSEventRecord{
			*snsEventRecord,
		},
	}

	l := &SNSLock{}

	expected := "7dfaa4af204fccecf31a47d8d10d60194776670866fe83145cc75a0395f6da75"
	actual, err := l.messageHash(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSNSLock_messageHash_s3(t *testing.T) {
	b, err := os.ReadFile("testdata/valid_sns_s3_event.json")
	assert.NoError(t, err)

	snsEventRecord := &events.SNSEventRecord{}
	assert.NoError(t, json.Unmarshal(b, snsEventRecord))

	snsEvent := events.SNSEvent{
		Records: []events.SNSEventRecord{
			*snsEventRecord,
		},
	}

	l := &SNSLock{}

	// Set custom hash function for s3Event
	l.SetHashFunc(func(message string) (string, error) {
		var s3Event events.S3Event
		err := json.Unmarshal([]byte(message), &s3Event)
		if err != nil {
			return "", errors.Wrap(err, "failed to unmarshal S3 event")
		}

		if len(s3Event.Records) != 1 {
			return "", fmt.Errorf("expected only 1 S3 event record, received: %d", len(s3Event.Records))
		}

		s3Record := s3Event.Records[0]
		data := s3Record.S3.Bucket.Arn + s3Record.S3.Object.Key + fmt.Sprint(s3Record.S3.Object.Size) + s3Record.S3.Object.ETag
		sum := sha256.Sum256([]byte(data))
		return fmt.Sprintf("%x", sum), nil
	})

	expected := "88c06f58a0517bb46f1e4ce51257d2f320f0f54d4daf69ceabf017ac119e924b"
	actual, err := l.messageHash(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSNSLock_expires(t *testing.T) {
	l := &SNSLock{TTL: 15}
	l.nowFunc = func() time.Time { return time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC) }

	expected := "1257894015"
	actual := l.expires()
	assert.Equal(t, expected, actual)
}

func TestSNSLock_current(t *testing.T) {
	l := &SNSLock{TTL: 15}
	l.nowFunc = func() time.Time { return time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC) }

	expected := "1257894000"
	actual := l.current()
	assert.Equal(t, expected, actual)
}

func TestSNSLock_putItemInput(t *testing.T) {
	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.nowFunc = func() time.Time { return time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC) }

	input := l.putItemInput("1234")

	assert.Equal(t, "t1", *input.TableName)
	assert.Equal(t, "attribute_not_exists(id) OR :cur > expire", *input.ConditionExpression)
	assert.Equal(t, "1257894000", *input.ExpressionAttributeValues[":cur"].N)
	assert.Equal(t, "1234", *input.Item["id"].S)
	assert.Equal(t, "1257894900", *input.Item["expire"].N)
}

type successMockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *successMockDynamoDBClient) PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, nil
}

type failedMockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *failedMockDynamoDBClient) PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, awserr.New(dynamodb.ErrCodeConditionalCheckFailedException, "condition fail", errors.New("test fail"))
}

type errorMockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
}

func (m *errorMockDynamoDBClient) PutItem(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	return nil, errors.New("test fail")
}

func TestSNSLock_AvailableById(t *testing.T) {
	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.svcFunc = func(client.ConfigProvider) dynamodbiface.DynamoDBAPI { return &successMockDynamoDBClient{} }

	available, err := l.AvailableById("1234")
	assert.NoError(t, err)
	assert.True(t, available)
}

func TestSNSLock_AvailableById_nope(t *testing.T) {
	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.svcFunc = func(client.ConfigProvider) dynamodbiface.DynamoDBAPI { return &failedMockDynamoDBClient{} }

	available, err := l.AvailableById("1234")
	assert.NoError(t, err)
	assert.False(t, available)
}

func TestSNSLock_AvailableById_error(t *testing.T) {
	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.svcFunc = func(client.ConfigProvider) dynamodbiface.DynamoDBAPI { return &errorMockDynamoDBClient{} }

	_, err := l.AvailableById("1234")
	assert.Error(t, err)
}

func TestSNSLock_Available(t *testing.T) {
	b, err := os.ReadFile("testdata/valid_sns_string_event.json")
	assert.NoError(t, err)

	snsEventRecord := &events.SNSEventRecord{}
	assert.NoError(t, json.Unmarshal(b, snsEventRecord))

	snsEvent := events.SNSEvent{
		Records: []events.SNSEventRecord{
			*snsEventRecord,
		},
	}

	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.svcFunc = func(client.ConfigProvider) dynamodbiface.DynamoDBAPI { return &successMockDynamoDBClient{} }

	available, err := l.Available(snsEvent)
	assert.NoError(t, err)
	assert.True(t, available)
}

func TestSNSLock_Available_errorRecords(t *testing.T) {
	b, err := os.ReadFile("testdata/valid_sns_string_event.json")
	assert.NoError(t, err)

	snsEventRecord := &events.SNSEventRecord{}
	assert.NoError(t, json.Unmarshal(b, snsEventRecord))

	snsEvent := events.SNSEvent{
		Records: []events.SNSEventRecord{
			*snsEventRecord,
			*snsEventRecord,
		},
	}

	l := &SNSLock{Region: "r1", Table: "t1", TTL: 900}
	l.svcFunc = func(client.ConfigProvider) dynamodbiface.DynamoDBAPI { return &successMockDynamoDBClient{} }

	_, err = l.Available(snsEvent)
	assert.Error(t, err)
}
