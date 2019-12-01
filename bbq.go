package bbq

import (
	"sync"
	"time"
)

// Callback is a channel returned whenever an item is enqueued, and closed (possibly after sending an error) when an item is processed
type Callback = chan error

// Flush handles the contents of the batch queue and optionally returns an error
type Flush = func([]interface{}) error

// BatchQueue is a thread-safe buffer of items that calls a given `flush` function with its contents when reaching a predefined count or time interval, and then empties itself
type BatchQueue struct {
	mut           *sync.RWMutex
	items         []interface{}
	cbs           []Callback
	flushTime     time.Duration
	flushCount    int
	lastFlushTime time.Time
	flush         Flush
	waiting       bool
}

// BatchQueueOptions define the behavior of the batch queue
type BatchQueueOptions struct {
	// FlushTime controls the minimum time spent between flushes that do not exeed FlushSize. Setting this to 0 will flush on every Enqueue().
	FlushTime time.Duration

	// FlushCount is the number of items that can accumulate within FlushTime before being flushed immediately. Set this to 0 will flush on every Enqueue().
	FlushCount int
}

// DefaultOptions will flush at least once per second, including whenever the queue reaches 1024 items
var DefaultOptions = BatchQueueOptions{time.Second, 1024}

// NewBatchQueue returns a queue
func NewBatchQueue(flush Flush, opts BatchQueueOptions) *BatchQueue {
	if opts.FlushTime == 0 {
		opts.FlushTime = DefaultOptions.FlushTime
	}
	if opts.FlushCount == 0 {
		opts.FlushCount = DefaultOptions.FlushCount
	}
	mut := &sync.RWMutex{}
	items := []interface{}{}
	cbs := []Callback{}
	return &BatchQueue{mut, items, cbs, opts.FlushTime, opts.FlushCount, time.Now(), flush, false}
}

// Enqueue puts an item on the batch queue
func (q *BatchQueue) Enqueue(item interface{}) Callback {
	q.mut.Lock()
	cb := make(chan error, 1)
	q.items = append(q.items, item)
	q.cbs = append(q.cbs, cb)

	if len(q.items) >= q.flushCount {
		q.mut.Unlock()
		q.FlushNow()
		return cb
	}

	if time.Now().After(q.lastFlushTime.Add(q.flushTime)) {
		q.mut.Unlock()
		q.FlushNow()
		return cb
	}

	if !q.waiting {
		q.waiting = true
		q.setFlushTimeout()
	}

	q.mut.Unlock() // not using defer because we need to unlock before flushing

	return cb
}

// FlushNow will immediately flush the batch queue
func (q *BatchQueue) FlushNow() {
	q.mut.Lock()
	defer q.mut.Unlock()
	tmp := make([]interface{}, len(q.items))
	copy(tmp, q.items)
	q.items = []interface{}{}

	if err := q.flush(tmp); err != nil {
		for _, cb := range q.cbs {
			cb <- err
			close(cb)
		}
	} else {
		for _, cb := range q.cbs {
			close(cb)
		}
	}

	q.cbs = []Callback{}

	q.lastFlushTime = time.Now()
	q.waiting = false
}

func (q *BatchQueue) setFlushTimeout() {
	go func() {
		time.Sleep(q.flushTime)
		q.mut.RLock()
		if q.waiting {
			q.mut.RUnlock()
			q.FlushNow()
		} else {
			q.mut.RUnlock()
		}
	}()
}
