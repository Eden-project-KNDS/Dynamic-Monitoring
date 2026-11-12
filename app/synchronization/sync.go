package synchronization

import (
	"context"
	"os/signal"
	"sync"
	"syscall"
)

// NUMBER_OF_GOROUTINES is how many long-running goroutines the program starts (now only the http shutdown one).
const NUMBER_OF_GOROUTINES = 1

// SyncStruct is shared by all long-running goroutines, so they can stop together.
type SyncStruct struct {
	Ctx context.Context // cancelled on SIGINT or SIGTERM
	Wg  *sync.WaitGroup // main waits on it until all goroutines stop
}

// CreateSignalHandling listens for SIGINT and SIGTERM. Calling stop cancels Ctx by hand.
// SIGKILL can't be caught, so it is not handled.
func CreateSignalHandling() (*SyncStruct, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(NUMBER_OF_GOROUTINES)

	return &SyncStruct{Ctx: ctx, Wg: &wg}, stop
}
