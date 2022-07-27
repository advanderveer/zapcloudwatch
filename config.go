package zapcloudwatch

// Config configures how logs are written to CloudWatch
type Config struct {
	LogGroupName     string
	LogStreamName    string
	AutoCreateStream bool
}
