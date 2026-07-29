// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package limit

// DefaultMaxConcurrency is the limit applied until Configure says otherwise.
const DefaultMaxConcurrency = 8

// The collectors fan out per namespace, per pod/node and finally per kubectl/curl invocation. Without an upper
// bound a large cluster leads to thousands of concurrently running child processes, each with its own resident
// memory - which is what makes steadybit-debug exhaust the memory of the machine it runs on. The semaphores
// below bound each of those levels so that the footprint stays independent of the cluster size.
//
// Acquisition order is Namespaces -> Items/Nodes -> Commands. Never acquire a semaphore of an earlier level
// while holding a later one, otherwise the levels can deadlock each other.
//
// Nodes is separate from Items on purpose: a cluster has far more nodes than steadybit pods, and goroutines
// queue on a semaphore in FIFO order. Sharing one semaphore would put the thousands of node goroutines ahead of
// the platform, agent and extension collectors and delay the interesting data until node collection is done.
var (
	Namespaces *Semaphore
	Items      *Semaphore
	Nodes      *Semaphore
	Commands   *Semaphore
)

func init() {
	Configure(DefaultMaxConcurrency)
}

// Configure applies the concurrency limits, a value below one disables them. It must be called before the
// collectors are started.
func Configure(maxConcurrency int) {
	Namespaces = New(maxConcurrency)
	Items = New(maxConcurrency)
	Nodes = New(maxConcurrency)
	Commands = New(2 * maxConcurrency)
}

// Semaphore bounds the number of operations running at the same time. A limit below one means unbounded.
type Semaphore struct {
	slots chan struct{}
}

func New(limit int) *Semaphore {
	if limit < 1 {
		return &Semaphore{}
	}
	return &Semaphore{slots: make(chan struct{}, limit)}
}

// Acquire blocks until a slot is free and returns the function that releases it again.
func (s *Semaphore) Acquire() func() {
	if s.slots == nil {
		return func() {}
	}
	s.slots <- struct{}{}
	return func() {
		<-s.slots
	}
}
