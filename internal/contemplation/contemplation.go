// Package contemplation implements the mandatory startup pause used to
// re-synchronize the system with 32-bit constraints before work begins.
package contemplation

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultDuration is the enforced contemplation period length.
const DefaultDuration = 15 * time.Minute

const defaultTick = time.Second

// Options configures the contemplation routine.
type Options struct {
	Duration time.Duration
	Tick     time.Duration
	Writer   io.Writer
	Label    string
	Concepts []string
}

// Run blocks for the configured duration while emitting a progress bar and
// rotating concept descriptions. The context may be used to cancel early in
// tests; production callers should pass context.Background().
func Run(ctx context.Context, opts Options) error {
	duration := opts.Duration
	if duration <= 0 {
		duration = DefaultDuration
	}
	tick := opts.Tick
	if tick <= 0 {
		tick = defaultTick
	}
	w := opts.Writer
	if w == nil {
		w = os.Stdout
	}
	label := opts.Label
	if label == "" {
		label = "Synchronizing Neural Root..."
	}
	concepts := opts.Concepts
	if len(concepts) == 0 {
		concepts = defaultConcepts()
	}

	start := time.Now()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		elapsed := time.Since(start)
		if elapsed >= duration {
			renderProgress(w, label, concepts, duration, duration)
			fmt.Fprint(w, "\n")
			return nil
		}

		renderProgress(w, label, concepts, elapsed, duration)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// DurationFromEnv parses an override duration from the provided environment
// variable. Values can be integer seconds or Go duration strings.
func DurationFromEnv(envVar string, fallback time.Duration) time.Duration {
	if fallback <= 0 {
		fallback = DefaultDuration
	}
	raw := strings.TrimSpace(os.Getenv(envVar))
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return fallback
}

func renderProgress(w io.Writer, label string, concepts []string, elapsed, total time.Duration) {
	percent := int(float64(elapsed) / float64(total) * 100)
	if percent > 100 {
		percent = 100
	}
	barWidth := 32
	filled := percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
	concept := concepts[conceptIndex(elapsed, len(concepts))]

	fmt.Fprintf(w, "\r[%s] %3d%% %s %s", bar, percent, label, concept)
}

func conceptIndex(elapsed time.Duration, total int) int {
	if total == 0 {
		return 0
	}
	sec := int(elapsed.Seconds())
	if sec < 0 {
		sec = 0
	}
	return sec % total
}

func defaultConcepts() []string {
	return []string{
		"Revalidating GDT base/limit alignment",
		"Rehearsing IDT vectors and gates",
		"Verifying CR0/CR3 paging discipline",
		"Refreshing 32-bit segment selector hygiene",
		"Rejecting 64-bit opcode drift (386/486 focus)",
		"Re-reading Open386 toolchain notes for class",
		"Holding micro-bus lock; WAIT state enforced",
	}
}
