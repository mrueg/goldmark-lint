package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const watchInterval = 500 * time.Millisecond

// runWatch polls allFiles for modification-time changes every watchInterval and
// calls relintFn with each batch of changed files. It runs until the process
// receives an interrupt or SIGTERM signal.
func runWatch(allFiles []string, relintFn func(files []string)) {
	// Seed initial modification times.
	mtimes := make(map[string]time.Time, len(allFiles))
	for _, f := range allFiles {
		if fi, err := os.Stat(f); err == nil {
			mtimes[f] = fi.ModTime()
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	fmt.Fprintf(os.Stderr, "Watching %d file(s) for changes... (press Ctrl+C to stop)\n", len(allFiles))

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			return
		case <-ticker.C:
			var changed []string
			for _, f := range allFiles {
				fi, err := os.Stat(f)
				if err != nil {
					continue
				}
				if fi.ModTime() != mtimes[f] {
					mtimes[f] = fi.ModTime()
					changed = append(changed, f)
				}
			}
			if len(changed) > 0 {
				relintFn(changed)
			}
		}
	}
}
