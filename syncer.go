package zapcloudwatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// CloudWatchClient describes the part of the cloudwatch client that we require
type CloudWatchClient interface {
	DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
	PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error)
	CreateLogStream(ctx context.Context, params *cloudwatchlogs.CreateLogStreamInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogStreamOutput, error)
}

type Syncer struct {
	client   CloudWatchClient
	config   Config
	sequence chan *string
	buffer   *Buffer
}

func NewSyncer(cfg Config, cl CloudWatchClient, buf *Buffer) (s *Syncer) {
	s = &Syncer{
		config:   cfg,
		client:   cl,
		sequence: make(chan *string, 1),
		buffer:   buf,
	}

	return
}

// Write the log event to the buffer. If the buffer is full this method will automatically call
// Sync to persist the logs to the CloudWatch API.
func (s *Syncer) Write(ctx context.Context, le types.InputLogEvent) (err error) {
	for i := 0; i < 4; i++ {
		full, disc := s.buffer.Push(le)
		fmt.Println(*le.Message, full)
		if disc || !full {
			return nil
		}

		if err = s.Sync(ctx); err != nil {
			return fmt.Errorf("failed to sync during write to full buffer: %w", err)
		}
	}

	return fmt.Errorf("failed to write to full buffer after consecutive syncs")
}

func (s *Syncer) putLogs(ctx context.Context, token *string, batch []types.InputLogEvent) {
	in := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     batch,
		LogGroupName:  aws.String(s.config.LogGroupName),
		LogStreamName: aws.String(s.config.LogStreamName),
		SequenceToken: token,
	}

	out, err := s.client.PutLogEvents(ctx, in)
	if err != nil {
		// @TODO We need to let the user know that we've lost logs at this point.
		fmt.Println("Error", err)
		// return fmt.Errorf("failed to put log events: %w", err)
	}
	s.sequence <- out.NextSequenceToken // make a new sequence token available
}

// Sync will flush the contents of the buffer and write to the CloudWatch API.
func (s *Syncer) Sync(ctx context.Context) (err error) {
	batch := s.buffer.Batch()
	if len(batch) < 1 {
		return
	}

	token := <-s.sequence        // block until there is a token available
	s.putLogs(ctx, token, batch) // send logs async so it can continue
	return
}

// Start the syncer by describing the logstream for the starting sequence.
func (s *Syncer) Start(ctx context.Context) error {
	group, stream := s.config.LogGroupName, s.config.LogStreamName

	in := &cloudwatchlogs.DescribeLogStreamsInput{
		LogStreamNamePrefix: aws.String(stream),
		Limit:               aws.Int32(2),
		LogGroupName:        aws.String(group)}

	out, err := s.client.DescribeLogStreams(ctx, in)
	switch {
	case err != nil:
		return fmt.Errorf("failed to describe log stream: %w", err)
	case len(out.LogStreams) == 1:
		s.sequence <- out.LogStreams[0].UploadSequenceToken
		return nil
	case len(out.LogStreams) > 2:
		return fmt.Errorf("more than one logstream that start with '%s'", stream)
	case len(out.LogStreams) < 1 && s.config.AutoCreateStream:

		// we auto-create a new logstream
		if _, err := s.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  aws.String(group),
			LogStreamName: aws.String(stream),
		}); err != nil {
			return fmt.Errorf("failed to auto-create stream '%s' for group '%s': %w", stream, group, err)
		}

		// with a new logstream the sequence can be empty
		s.sequence <- nil
		return nil
	default:
		return fmt.Errorf("no log stream called '%s' in group '%s'", stream, group)
	}
}

func (s *Syncer) Stop(ctx context.Context) (err error) {
	token := <-s.sequence // wait until any running syncs are done
	if batch := s.buffer.Batch(); len(batch) > 0 {
		s.putLogs(ctx, token, batch) // put any final logs in the buffer
	}

	close(s.sequence)
	return nil
}
