package s3eventutils

import (
	"io/ioutil"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func createSNSEvent(records ...events.SNSEventRecord) events.SNSEvent {
	return events.SNSEvent{Records: records}
}

func createSNSRecord(message string) events.SNSEventRecord {
	snsEntity := events.SNSEntity{
		Type:     "Notification",
		TopicArn: "arn:aws:sns:us-east-1:xxxx:MilkyWay",
		Message:  string(message),
	}

	snsEventRecord := events.SNSEventRecord{
		EventSubscriptionArn: "arn:aws:sns:us-east-1:xxxx:TOPIC:fad1bad1-feed-dead-face-bb111222333",
		SNS:                  snsEntity,
	}

	return snsEventRecord
}

func Test_s3EventRecordFromSNSWrapper(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/valid_message_s3.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)))

	r, err := s3EventRecordFromSNSWrapper(snsEvent)

	assert.NoError(t, err)
	assert.Equal(t, "bktname", r.S3.Bucket.Name)
	assert.Equal(t, "some/file/in/s3.txt", r.S3.Object.Key)
}

func Test_s3EventRecordFromSNSWrapper_error_sns_record_count(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/valid_message_s3.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)), createSNSRecord(string(b)))

	_, err = s3EventRecordFromSNSWrapper(snsEvent)
	assert.Error(t, err)
}

func Test_s3EventRecordFromSNSWrapper_error_invalid_message(t *testing.T) {
	snsEvent := createSNSEvent(createSNSRecord("not json"))

	_, err := s3EventRecordFromSNSWrapper(snsEvent)
	assert.Error(t, err)
}

func Test_s3EventRecordFromSNSWrapper_error_s3_Record_count(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/invalid_message_s3_count.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)))

	_, err = s3EventRecordFromSNSWrapper(snsEvent)
	assert.Error(t, err)
}

func TestUriFromSNSS3EventMessage(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/valid_message_s3.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)))

	uri, err := UriFromSNSS3EventMessage(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, "s3://bktname/some/file/in/s3.txt", uri)
}

func TestUriFromSNSS3EventMessage_folder(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/valid_message_s3_folder.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)))

	uri, err := UriFromSNSS3EventMessage(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, "s3://bktname/some/file/in/folder/", uri)
}

func TestUriFromSNSS3EventMessage_error(t *testing.T) {
	snsEvent := createSNSEvent(createSNSRecord("not json"))

	_, err := UriFromSNSS3EventMessage(snsEvent)
	assert.Error(t, err)
}

func TestS3ObjectFromSNSS3EventMessage(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/valid_message_s3.json")
	assert.NoError(t, err)

	snsEvent := createSNSEvent(createSNSRecord(string(b)))

	bucket, key, err := S3ObjectFromSNSS3EventMessage(snsEvent)
	assert.NoError(t, err)
	assert.Equal(t, "bktname", bucket)
	assert.Equal(t, "some/file/in/s3.txt", key)
}

func TestS3ObjectFromSNSS3EventMessage_error(t *testing.T) {
	snsEvent := createSNSEvent(createSNSRecord("not json"))

	_, _, err := S3ObjectFromSNSS3EventMessage(snsEvent)
	assert.Error(t, err)
}
