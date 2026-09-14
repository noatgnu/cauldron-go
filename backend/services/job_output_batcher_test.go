package services

import (
	"sync"
	"testing"
	"time"
)

type flushCall struct {
	jobID string
	lines []string
}

func newRecordingBatcher(delay time.Duration) (*JobOutputBatcher, func() []flushCall, func(n int, timeout time.Duration) bool) {
	var mu sync.Mutex
	var calls []flushCall
	notify := make(chan struct{}, 1024)

	b := NewJobOutputBatcher(delay, func(jobID string, lines []string) {
		mu.Lock()
		calls = append(calls, flushCall{jobID: jobID, lines: append([]string(nil), lines...)})
		mu.Unlock()
		notify <- struct{}{}
	})

	getCalls := func() []flushCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]flushCall(nil), calls...)
	}

	waitForCalls := func(n int, timeout time.Duration) bool {
		deadline := time.After(timeout)
		for {
			mu.Lock()
			count := len(calls)
			mu.Unlock()
			if count >= n {
				return true
			}
			select {
			case <-notify:
			case <-deadline:
				return false
			}
		}
	}

	return b, getCalls, waitForCalls
}

func TestJobOutputBatcher_DoesNotFlushSynchronously(t *testing.T) {
	b, getCalls, _ := newRecordingBatcher(50 * time.Millisecond)

	b.Add("job-1", "line 1")

	if len(getCalls()) != 0 {
		t.Errorf("expected no synchronous flush, got %d calls", len(getCalls()))
	}
}

func TestJobOutputBatcher_FlushesAfterDelay(t *testing.T) {
	b, getCalls, waitForCalls := newRecordingBatcher(20 * time.Millisecond)

	b.Add("job-1", "line 1")

	if !waitForCalls(1, time.Second) {
		t.Fatal("expected a flush after the delay, got none")
	}
	calls := getCalls()
	if len(calls) != 1 || calls[0].jobID != "job-1" || len(calls[0].lines) != 1 || calls[0].lines[0] != "line 1" {
		t.Errorf("unexpected flush calls: %+v", calls)
	}
}

func TestJobOutputBatcher_CoalescesRapidLinesIntoOneFlush(t *testing.T) {
	b, getCalls, waitForCalls := newRecordingBatcher(50 * time.Millisecond)

	for i := 0; i < 20; i++ {
		b.Add("job-1", "line")
	}

	if !waitForCalls(1, time.Second) {
		t.Fatal("expected a flush after the delay, got none")
	}
	time.Sleep(60 * time.Millisecond) // give any (unwanted) extra flush a chance to land
	calls := getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 flush call for 20 rapid lines, got %d", len(calls))
	}
	if len(calls[0].lines) != 20 {
		t.Errorf("expected all 20 lines in the single flush, got %d", len(calls[0].lines))
	}
}

func TestJobOutputBatcher_SubsequentLinesGetANewFlush(t *testing.T) {
	b, _, waitForCalls := newRecordingBatcher(20 * time.Millisecond)

	b.Add("job-1", "first")
	if !waitForCalls(1, time.Second) {
		t.Fatal("expected first flush")
	}

	b.Add("job-1", "second")
	if !waitForCalls(2, time.Second) {
		t.Fatal("expected second flush after new lines were added")
	}
}

func TestJobOutputBatcher_FlushJobIsImmediateAndSkipsIfEmpty(t *testing.T) {
	b, getCalls, _ := newRecordingBatcher(time.Hour) // long delay: only FlushJob should trigger it

	b.FlushJob("job-1")
	if len(getCalls()) != 0 {
		t.Error("expected FlushJob on an empty job to be a no-op")
	}

	b.Add("job-1", "line 1")
	b.Add("job-1", "line 2")
	b.FlushJob("job-1")

	calls := getCalls()
	if len(calls) != 1 || len(calls[0].lines) != 2 {
		t.Fatalf("expected FlushJob to immediately flush both buffered lines, got %+v", calls)
	}
}

func TestJobOutputBatcher_IndependentJobsBatchSeparately(t *testing.T) {
	b, getCalls, waitForCalls := newRecordingBatcher(20 * time.Millisecond)

	b.Add("job-1", "a")
	b.Add("job-2", "b")

	if !waitForCalls(2, time.Second) {
		t.Fatal("expected a flush for each job")
	}

	calls := getCalls()
	byJob := map[string][]string{}
	for _, c := range calls {
		byJob[c.jobID] = c.lines
	}
	if len(byJob["job-1"]) != 1 || byJob["job-1"][0] != "a" {
		t.Errorf("job-1 flush = %v", byJob["job-1"])
	}
	if len(byJob["job-2"]) != 1 || byJob["job-2"][0] != "b" {
		t.Errorf("job-2 flush = %v", byJob["job-2"])
	}
}

func TestJobOutputBatcher_ConcurrentAddIsRaceSafe(t *testing.T) {
	b, _, waitForCalls := newRecordingBatcher(20 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.Add("job-1", "line")
		}(i)
	}
	wg.Wait()

	if !waitForCalls(1, time.Second) {
		t.Fatal("expected at least one flush after concurrent adds")
	}
}
