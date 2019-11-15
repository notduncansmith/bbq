package bbq

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func Example() {
	flush := func(ms []interface{}) error {
		str := ""
		for _, m := range ms {
			str += m.(string) + " "
		}
		str += fmt.Sprintf("(%d)", len(str))
		fmt.Println(str)
		return nil
	}
	q := NewBatchQueue(flush, BatchQueueOptions{time.Second, 2})
	q.Enqueue("hello")
	q.Enqueue("world")
	// Output:
	// hello world (12)
}

func Example_time() {
	flush := func(ms []interface{}) error {
		for _, m := range ms {
			fmt.Println(m.(string))
		}
		return nil
	}
	q := NewBatchQueue(flush, BatchQueueOptions{1 * time.Millisecond, 2})
	q.Enqueue("🍪")
	time.Sleep(2 * time.Millisecond)
	// Output:
	// 🍪
}

func Example_now() {
	flush := func(ms []interface{}) error {
		for _, m := range ms {
			fmt.Println(m.(string))
		}
		return nil
	}
	q := NewBatchQueue(flush, BatchQueueOptions{time.Second, 2})
	q.Enqueue("🥑")
	q.FlushNow()
	// Output:
	// 🥑
}

type TestItem struct {
	key string
}

func TestRoundtrip(t *testing.T) {
	var out TestItem
	flush := func(ms []interface{}) error {
		for _, m := range ms {
			out = m.(TestItem)
		}
		return nil
	}

	q := NewBatchQueue(flush, BatchQueueOptions{3 * time.Second, 3})
	k1 := "k1"
	k2 := "k2"
	q.Enqueue(TestItem{k1})
	q.Enqueue(TestItem{k2})
	q.FlushNow()
	actual := out.key
	if actual != k2 {
		t.Errorf("Should be able to roundtrip key, got %v", actual)
	}
}

func TestFlushOnTime(t *testing.T) {
	// Note that unlike the other tests, this one uses a mutex around `out`.
	// This protects the `out` variable from data races.
	// This test will fail the race detector without the mutex.
	mut := sync.RWMutex{}
	var out TestItem
	flush := func(ms []interface{}) error {
		for _, m := range ms {
			mut.Lock()
			out = m.(TestItem)
			mut.Unlock()
		}
		return nil
	}
	q := NewBatchQueue(flush, BatchQueueOptions{1 * time.Millisecond, 3})
	k1 := "k1"
	k2 := "k2"
	q.Enqueue(TestItem{k1})
	q.FlushNow()
	time.Sleep(1 * time.Millisecond)
	cb := q.Enqueue(TestItem{k2})
	if err := <-cb; err != nil {
		t.Errorf("Should get callback with no error, got %v", err)
	}
	mut.RLock()
	actual := out.key
	mut.RUnlock()
	if actual != k2 {
		t.Errorf("Should be able to flush key, got %v", actual)
	}
}

func TestFlushOnCount(t *testing.T) {
	var out TestItem
	flush := func(ms []interface{}) error {
		for _, m := range ms {
			out = m.(TestItem)
		}
		return nil
	}

	q := NewBatchQueue(flush, BatchQueueOptions{time.Second, 1})
	k1 := "k1"

	q.Enqueue(TestItem{k1})
	actual := out.key
	if actual != k1 {
		t.Errorf("Should be able to flush key, got %v", actual)
	}
}

func TestErrCallback(t *testing.T) {
	flush := func(ms []interface{}) error {
		return errors.New("test")
	}
	q := NewBatchQueue(flush, BatchQueueOptions{1 * time.Millisecond, 3})
	k1 := "k1"

	cb := q.Enqueue(TestItem{k1})
	if err := <-cb; err == nil {
		t.Errorf("Should get callback with error, got nil")
	}
}

func TestFlushTimeout(t *testing.T) {
	flushes := 0
	flush := func(ms []interface{}) error {
		flushes += 1
		return nil
	}
	q := NewBatchQueue(flush, BatchQueueOptions{10 * time.Millisecond, 3})

	cb1 := q.Enqueue(TestItem{"k1"}) // sets flush timeout for 10ms
	q.FlushNow()                     // should clear timeout
	time.Sleep(10 * time.Millisecond)

	if err := <-cb1; err != nil {
		t.Errorf("Should get callback with no error, got %v", err)
	}

	if flushes != 1 {
		t.Errorf("Expected 1 flush, got %v", flushes)
	}
}
