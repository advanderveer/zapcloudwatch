package zapcloudwatch

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// LogEvent creates a standard CloudWatch log event for testing
func LogEvent(offset int64, msg ...string) types.InputLogEvent {
	if len(msg) < 1 {
		msg = append(msg, "foo")
	}

	return types.InputLogEvent{
		Timestamp: aws.Int64(time.Now().UnixMilli() + offset),
		Message:   aws.String(msg[0]),
	}
}
