//go:build !windows

package main

import (
	"errors"
	"syscall"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

func TestBuildStopKillsTheRunningCommand(t *testing.T) {
	done := make(chan error, 1)
	go func() { done <- run(nil, "test", t.TempDir(), "sleep 30 & wait") }()
	var pid int
	wait.Until(t, "the command is registered", func() bool {
		activeRun.Lock()
		defer activeRun.Unlock()
		if activeRun.cmd == nil || activeRun.cmd.Process == nil {
			return false
		}
		pid = activeRun.cmd.Process.Pid
		return true
	})
	stopRunning() // what the signal handler calls
	wait.For(t, "run to return", done)
	wait.Until(t, "the process group is gone", func() bool {
		return errors.Is(syscall.Kill(-pid, 0), syscall.ESRCH)
	})
}
