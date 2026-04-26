package commands

import (
	"os"
	"sync/atomic"
)

// restartChan is used to signal a restart request
var restartChan = make(chan struct{})

// restartRequested flag
var restartRequested atomic.Bool

// RequestRestart signals that the server should restart
func RequestRestart() {
	restartRequested.Store(true)
	// Send on channel to wake up the server
	restartChan <- struct{}{}
}

// WaitForRestart blocks until a restart is requested
func WaitForRestart() <-chan struct{} {
	return restartChan
}

// IsRestartRequested returns true if a restart was requested
func IsRestartRequested() bool {
	return restartRequested.Load()
}

// GetPID returns the current process ID
func GetPID() int {
	return os.Getpid()
}
