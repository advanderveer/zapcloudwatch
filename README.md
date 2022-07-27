# zapcloudwatch

Provides an AWS CloudWatch logging core for Uber's zap

## Backlog

- [ ] SHOULD implement graceful shutdown through Stop implementation
- [ ] SHOULD implement auto-create of stream if describe doesn't return the stream
- [ ] COULD make sure that when a put fails, we don't discard the log lines
- [ ] COULD fix the stream's next sequence token when the specific exception is thrown
- [ ] COULD continue writing to buffer while sync is running
- [ ] COULD create a huge integration test that writes with high concurrency and checks if all the logs actually
      end up in CloudWatch

## V2

- Sync cannot be async, when sync is done the code needs to be sure that the sync has completed, or maybe
  we can make the sync that is part of the write async, manual sync must be all the way?
  - But is there then no race between who reads the token first, no that is the case anyway
- Need to do it without context being passed into the sync/write, maybe allow configuration of context builders
