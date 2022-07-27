package zapcloudwatch

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var (
	ErrBufferStillFull = errors.New("buffer still full after repeated sync-on-write")
)

type BufferedWriter struct {
	buf    *Buffer
	st     chan *string
	cl     CloudWatchClient
	group  string
	stream string
}

func NewBufferedWriter(
	cl CloudWatchClient,
	group, stream string,
	buf *Buffer,
	startSeqTok *string,
) (w *BufferedWriter) {
	w = &BufferedWriter{
		buf:    buf,
		st:     make(chan *string, 1),
		cl:     cl,
		group:  group,
		stream: stream}

	w.st <- startSeqTok
	return
}

func (w *BufferedWriter) Write(ts int64, msg string) error {
	fmt.Println(msg)
	for i := 0; i < 2; i++ {
		full, disc := w.buf.Push(types.InputLogEvent{Timestamp: &ts, Message: &msg})
		if disc || !full {
			return nil
		}

		fmt.Println("FULL!")

		// buffer is full, sync in place and retry push on next iteration
		if err := w.Sync(); err != nil {
			return fmt.Errorf("failed to sync after write: %w", err)
		}
	}

	return ErrBufferStillFull
}

func (w *BufferedWriter) Sync() error {
	token, ctx := <-w.st, context.Background()
	nextToken, err := w.putLogs(ctx, w.buf.Batch(), token)
	if err != nil {
		w.st <- token
		return fmt.Errorf("failed to put logs: %w", err)
	}

	w.st <- nextToken
	return nil
}

func (w *BufferedWriter) putLogs(ctx context.Context, batch []types.InputLogEvent, tok *string) (next *string, err error) {
	fmt.Println("LEN", len(batch))
	in := &cloudwatchlogs.PutLogEventsInput{
		LogEvents:     batch,
		LogGroupName:  &w.group,
		LogStreamName: &w.stream,
		SequenceToken: tok,
	}

	out, err := w.cl.PutLogEvents(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("failed to call PutLogEvents: %w", err)
	}

	return out.NextSequenceToken, nil
}
