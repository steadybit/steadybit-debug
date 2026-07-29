// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package limit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSemaphoreBoundsConcurrency(t *testing.T) {
	const limit = 3
	semaphore := New(limit)

	var running, peak atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release := semaphore.Acquire()
			defer release()

			current := running.Add(1)
			for {
				observed := peak.Load()
				if current <= observed || peak.CompareAndSwap(observed, current) {
					break
				}
			}
			// hold the slot long enough for every other goroutine to reach its own Acquire, so that an Acquire
			// that does not block shows up as a peak far above the limit
			time.Sleep(20 * time.Millisecond)
			running.Add(-1)
		}()
	}
	wg.Wait()

	if peak.Load() > limit {
		t.Errorf("expected at most %d concurrent operations, got %d", limit, peak.Load())
	}
	if peak.Load() < 2 {
		t.Errorf("expected the operations to overlap, got a peak of %d - the test proves nothing", peak.Load())
	}
}

func TestSemaphoreWithoutLimitDoesNotBlock(t *testing.T) {
	for _, semaphore := range []*Semaphore{New(0), New(-1)} {
		release := semaphore.Acquire()
		release()
	}
}
