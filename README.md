# bbq - a basic batch queue for Go

[![GoDoc](https://godoc.org/github.com/notduncansmith/bbq?status.svg)](https://godoc.org/github.com/notduncansmith/bbq) [![Build Status](https://travis-ci.com/notduncansmith/bbq.svg?branch=master)](https://travis-ci.com/notduncansmith/bbq) [![codecov](https://codecov.io/gh/notduncansmith/bbq/branch/master/graph/badge.svg)](https://codecov.io/gh/notduncansmith/bbq)

bbq allows you to batch messages by time or count, then flush them to a function of your choice. bbq is thread-safe, utilizing Go's native `sync.RWMutex`. Flushes are synchronous. If the queue is empty, the flush function will not be called.

## Usage

```go
package main

import (
    "fmt"
    "github.com/notduncansmith/bbq"
)

func main() {
    flush := func(ms []interface{}) error {
		for _, m := range ms {
			fmt.Println(m.(string))
		}
		return nil
	}
	q := bbq.NewBatchQueue(flush, BatchQueueOptions{time.Millisecond, 2})
	q.Enqueue("🍖")
	time.Sleep(time.Millisecond)
	// Output:
	// 🍖
}
```

## License

Released under [The MIT License](https://opensource.org/licenses/MIT) (see `LICENSE.txt`).

Copyright 2019 Duncan Smith
