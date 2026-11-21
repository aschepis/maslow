package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aschepis/maslow/internal/runner"
)

// exitOnInterrupt kills running steps and exits on SIGINT or SIGTERM. Steps run in their own
// process groups (ADR 0044), so without this Ctrl-C would leave the current step running.
// No evidence is written: a partial report must not read as a verdict.
func exitOnInterrupt() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		fmt.Fprintln(os.Stderr, "maslow: interrupted")
		runner.KillRunning()
		os.Exit(exitCodeFor(sig))
	}()
}

// exitCodeFor follows the shell convention of 128 plus the signal number.
func exitCodeFor(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 130
}
