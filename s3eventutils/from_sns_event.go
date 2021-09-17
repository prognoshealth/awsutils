package s3eventutils

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/events"
)

// S3EventRecordFromSNSWrapper extracts the underlying s3 event record wrapped
// within the sns event.
func S3EventRecordFromSNSWrapper(snsEvent events.SNSEvent) (*events.S3EventRecord, error) {
	if len(snsEvent.Records) != 1 {
		return nil, errors.New(fmt.Sprintf("expected only 1 SNS event, received: %v", len(snsEvent.Records)))
	}

	message := snsEvent.Records[0].SNS.Message

	s3Event := new(events.S3Event)
	if err := json.Unmarshal([]byte(message), s3Event); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal %+v", s3Event)
	}

	if len(s3Event.Records) != 1 {
		return nil, fmt.Errorf("expect only 1 S3 event, received: %v", len(s3Event.Records))
	}

	return &s3Event.Records[0], nil
}

// UriFromSNSS3EventMessage extracts the s3 uri from an s3 event wrapped
// sns event.
func UriFromSNSS3EventMessage(snsEvent events.SNSEvent) (string, error) {
	b, k, err := S3ObjectFromSNSS3EventMessage(snsEvent)
	if err != nil {
		return "", errors.Wrap(err, "failed getting s3 bucket and key")
	}

	uri := fmt.Sprintf("s3://%s", path.Join(b, k))

	if strings.HasSuffix(k, "/") {
		uri = uri + "/"
	}

	return uri, nil
}

// S3ObjectFromSNSS3EventMessage extracts the bucket and key from an s3 event wrapped
// sns event.
func S3ObjectFromSNSS3EventMessage(snsEvent events.SNSEvent) (string, string, error) {
	record, err := S3EventRecordFromSNSWrapper(snsEvent)
	if err != nil {
		return "", "", errors.Wrap(err, "failed unwrapping s3 event record from sns")
	}

	return record.S3.Bucket.Name, record.S3.Object.Key, nil
}
