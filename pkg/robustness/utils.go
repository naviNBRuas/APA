package robustness

import (
	"fmt"
	"sync/atomic"
	"time"
)

var idCounter atomic.Uint64

func generateID() string {
	n := idCounter.Add(1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), n)
}
