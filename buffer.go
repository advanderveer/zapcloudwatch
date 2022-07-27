package zapcloudwatch

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type Buffer struct {
	mu    sync.Mutex
	evs   []types.InputLogEvent
	early int64
	late  int64
	bsize int

	rules struct {
		maxNumEvents int
		maxByteSize  int
		maxSpanTime  int64
		maxFuture    int64
		maxPast      int64
	}
}

func NewBuffer() (b *Buffer) {
	return NewCustomBuffer(
		10000,
		1048576,    // 1 MiB
		86400000,   // 24 hours
		7199000,    // 2 hours (-1sec)
		1209999000, // 14 days (+1sec)
	)
}

func NewCustomBuffer(maxNumEvents, maxByteSize int, maxSpanTime, maxFuture, maxPast int64) (b *Buffer) {
	b = &Buffer{}
	b.reset()
	b.rules.maxNumEvents = maxNumEvents
	b.rules.maxByteSize = maxByteSize
	b.rules.maxSpanTime = maxSpanTime
	b.rules.maxFuture = maxFuture
	b.rules.maxPast = maxPast
	return
}

func (b *Buffer) reset() {
	b.early, b.late = math.MaxInt, 0
	b.bsize = 0
	b.evs = nil // don't keep the underlying array
}

func (b *Buffer) all() (all []types.InputLogEvent) {
	for _, ev := range b.evs {
		all = append(all, types.InputLogEvent{
			Message:   ev.Message,
			Timestamp: ev.Timestamp,
		})
	}
	return
}

func (b *Buffer) All() (all []types.InputLogEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.all()
}

func (b *Buffer) Batch() (batch []types.InputLogEvent) {
	b.mu.Lock()
	batch = b.all()
	b.reset()
	b.mu.Unlock()

	sort.SliceStable(batch, func(i, j int) bool {
		return aws.ToInt64(batch[i].Timestamp) < aws.ToInt64(batch[j].Timestamp)
	})

	return
}

func (b *Buffer) Push(ev types.InputLogEvent) (full, discard bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// the maximum number of log events in a batch is 10,000
	if len(b.evs) >= b.rules.maxNumEvents {
		return true, false
	}

	// The maximum batch size is 1,048,576 bytes
	bsize := b.bsize + len(*ev.Message) + 26
	if bsize > b.rules.maxByteSize {
		return true, false
	}

	earliest, latest := b.early, b.late
	if *ev.Timestamp < earliest {
		earliest = *ev.Timestamp
	}
	if *ev.Timestamp > latest {
		latest = *ev.Timestamp
	}

	// a batch of log events in a single request cannot span more than 24 hours
	if latest-earliest > b.rules.maxSpanTime {
		return true, false
	}

	// None of the log events in the batch can be more than 2(-1sec) hours in the future.
	now := time.Now()
	if (*ev.Timestamp - now.UnixMilli()) > b.rules.maxFuture {
		return false, true
	}

	// None of the log events in the batch can be older than 14 days (+1sec)
	if (now.UnixMilli() - *ev.Timestamp) > b.rules.maxPast {
		return false, true
	}

	// accept the event into the buffer, update counts
	b.evs = append(b.evs, ev)
	b.early, b.late = earliest, latest
	b.bsize = bsize
	return
}
