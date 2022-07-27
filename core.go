package zapcloudwatch

import (
	"go.uber.org/zap/zapcore"
)

type Writer interface {
	Write(tx int64, msg string) error
	Sync() error
}

// Core implements a zap core that persists to cloudwatch
type Core struct {
	zapcore.LevelEnabler
	enc zapcore.Encoder
	out Writer
}

// NewCore inits a new CloudWatch core
func NewCore(enc zapcore.Encoder, w Writer, enab zapcore.LevelEnabler) *Core {
	return &Core{LevelEnabler: enab, enc: enc, out: w}
}

// Write the log entry
func (c *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	ts, msg :=
		ent.Time.UnixMilli(), // CloudWatch expects timestamps in millisecond since epoch
		buf.String()

	err = c.out.Write(ts, msg)
	buf.Free()
	if err != nil {
		return err
	}

	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		c.Sync()
	}

	return nil
}

// With adds structured context to the Core.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	for i := range fields {
		fields[i].AddTo(c.enc)
	}
	return clone
}

// Check determines whether the supplied Entry should be logged.
func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Sync flushes buffered logs (if any).
func (c *Core) Sync() error {
	return c.out.Sync()
}

// clone the core
func (c *Core) clone() *Core {
	return &Core{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
	}
}
