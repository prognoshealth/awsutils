package s3eventutils

import "encoding/json"

// S3TestEvent ...
type S3TestEvent struct {
	Service   string
	Event     string
	Time      string
	Bucket    string
	RequestID string
	HostID    string
}

// CheckIfS3TestEvent ...
func CheckIfS3TestEvent(message string) bool {
	event := new(S3TestEvent)
	if err := json.Unmarshal([]byte(message), event); err != nil {
		return false
	}

	return event.Event == "s3:TestEvent"
}
