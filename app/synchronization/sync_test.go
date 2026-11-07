package synchronization

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// waitForCancel fails the test if Ctx is not cancelled within a second.
func waitForCancel(t *testing.T, s *SyncStruct) {
	t.Helper()
	select {
	case <-s.Ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled")
	}
}

func TestSignalCancelsContext(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		s, stop := CreateSignalHandling()

		// The signal is caught by CreateSignalHandling, so it does not kill the test.
		if err := syscall.Kill(os.Getpid(), sig); err != nil {
			t.Fatal(err)
		}
		waitForCancel(t, s)
		stop()
	}
}

func TestStopCancelsContext(t *testing.T) {
	s, stop := CreateSignalHandling()
	stop()
	waitForCancel(t, s)
}
