package services

import (
	"sync"
	"time"
)

// JobOutputBatcher coalesces rapid per-line job output into periodic flushes so a chatty
// script's output doesn't serialize every other caller behind a single shared DB connection.
type JobOutputBatcher struct {
	mu       sync.Mutex
	pending  map[string][]string
	flushing map[string]bool
	flush    func(jobID string, lines []string)
	delay    time.Duration
}

// NewJobOutputBatcher creates a batcher that calls flush at most once per delay per job,
// with all lines added since the last flush.
func NewJobOutputBatcher(delay time.Duration, flush func(jobID string, lines []string)) *JobOutputBatcher {
	return &JobOutputBatcher{
		pending:  make(map[string][]string),
		flushing: make(map[string]bool),
		flush:    flush,
		delay:    delay,
	}
}

// Add buffers a line for jobID, scheduling a flush after delay if one isn't already pending.
func (b *JobOutputBatcher) Add(jobID string, line string) {
	b.mu.Lock()
	b.pending[jobID] = append(b.pending[jobID], line)
	alreadyScheduled := b.flushing[jobID]
	b.flushing[jobID] = true
	b.mu.Unlock()

	if alreadyScheduled {
		return
	}
	time.AfterFunc(b.delay, func() { b.FlushJob(jobID) })
}

// FlushJob immediately flushes any buffered lines for jobID. Safe to call even if nothing
// is buffered, and safe to call concurrently with a scheduled flush (only one wins).
func (b *JobOutputBatcher) FlushJob(jobID string) {
	b.mu.Lock()
	lines := b.pending[jobID]
	delete(b.pending, jobID)
	delete(b.flushing, jobID)
	b.mu.Unlock()

	if len(lines) == 0 {
		return
	}
	b.flush(jobID, lines)
}
