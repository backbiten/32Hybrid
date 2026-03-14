// Package winstratch provides Windows 2000-style UI components for the 32Hybrid system.
// This file implements the contemplation period progress dialog.
package winstratch

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// ProgressFile is the file path where contemplation progress is written
	ProgressFile = "/tmp/contemplation_progress"

	// UnlockSentinel is the file that indicates contemplation is complete
	UnlockSentinel = "/tmp/neural_registry_unlocked"

	// UpdateInterval is how often to check for progress updates
	UpdateInterval = 1 * time.Second
)

// ContemplationProgress represents the current state of the contemplation period
type ContemplationProgress struct {
	ProgressPercent float64
	Concept         string
	Description     string
	Timestamp       time.Time
}

// ReadContemplationProgress reads the current contemplation state from the progress file
func ReadContemplationProgress() (*ContemplationProgress, error) {
	file, err := os.Open(ProgressFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open progress file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, 3)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading progress file: %w", err)
	}

	if len(lines) < 3 {
		return nil, fmt.Errorf("progress file has insufficient data")
	}

	percent, err := strconv.ParseFloat(lines[0], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid progress percentage: %w", err)
	}

	return &ContemplationProgress{
		ProgressPercent: percent,
		Concept:         lines[1],
		Description:     lines[2],
		Timestamp:       time.Now(),
	}, nil
}

// IsContemplationComplete checks if the neural registry has been unlocked
func IsContemplationComplete() bool {
	_, err := os.Stat(UnlockSentinel)
	return err == nil
}

// ShowContemplationDialog displays the contemplation progress bar
// This function blocks until contemplation is complete
func ShowContemplationDialog() error {
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   32Hybrid Neural Synchronization                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	ticker := time.NewTicker(UpdateInterval)
	defer ticker.Stop()

	lastPercent := -1.0

	for {
		select {
		case <-ticker.C:
			// Check if contemplation is complete
			if IsContemplationComplete() {
				drawFinalProgress()
				return nil
			}

			// Read current progress
			progress, err := ReadContemplationProgress()
			if err != nil {
				// Progress file might not exist yet at the very start
				continue
			}

			// Only redraw if progress has changed
			if progress.ProgressPercent != lastPercent {
				drawProgress(progress)
				lastPercent = progress.ProgressPercent
			}

			// Check for completion
			if progress.ProgressPercent >= 100.0 {
				time.Sleep(500 * time.Millisecond) // Brief pause to show 100%
				return nil
			}
		}
	}
}

// drawProgress renders the progress bar and current concept
func drawProgress(progress *ContemplationProgress) {
	// Clear previous lines (simple approach - move cursor up and clear)
	fmt.Print("\r\033[K") // Clear current line
	fmt.Print("\033[2A")  // Move up 2 lines
	fmt.Print("\033[K")   // Clear line

	// Draw progress bar
	barWidth := 60
	filled := int(progress.ProgressPercent * float64(barWidth) / 100.0)
	empty := barWidth - filled

	fmt.Print("Synchronizing Neural Root... ")
	fmt.Printf("%.1f%%\n", progress.ProgressPercent)

	fmt.Print("[")
	fmt.Print(strings.Repeat("█", filled))
	fmt.Print(strings.Repeat("░", empty))
	fmt.Print("]\n")

	// Draw concept and description
	fmt.Printf("\nCurrent Phase: %s\n", progress.Concept)
	fmt.Printf("└─ %s\n", progress.Description)

	// Time estimate
	if progress.ProgressPercent > 0 {
		totalSeconds := 900.0 // 15 minutes
		elapsedSeconds := (progress.ProgressPercent / 100.0) * totalSeconds
		remainingSeconds := int(totalSeconds - elapsedSeconds)

		minutes := remainingSeconds / 60
		seconds := remainingSeconds % 60
		fmt.Printf("\nEstimated time remaining: %d:%02d\n", minutes, seconds)
	}
}

// drawFinalProgress shows the completion message
func drawFinalProgress() {
	fmt.Print("\r\033[K") // Clear current line
	fmt.Print("\033[2A")  // Move up 2 lines
	fmt.Print("\033[K")   // Clear line

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   Neural Root Synchronized ✓                         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("i386 Architecture Knowledge: Verified")
	fmt.Println("Neural Registry: Unlocked")
	fmt.Println("AI Teacher: Ready for Operation")
	fmt.Println()
}

// DrawSimpleProgressBar is a simpler version for environments without full terminal control
func DrawSimpleProgressBar(percent float64, message string) {
	barWidth := 40
	filled := int(percent * float64(barWidth) / 100.0)
	empty := barWidth - filled

	fmt.Printf("\r%s [%s%s] %.1f%%",
		message,
		strings.Repeat("=", filled),
		strings.Repeat(" ", empty),
		percent)

	if percent >= 100.0 {
		fmt.Println()
	}
}

// WaitForContemplationComplete blocks until the contemplation period is finished
func WaitForContemplationComplete(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if IsContemplationComplete() {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for contemplation to complete")
		}

		time.Sleep(1 * time.Second)
	}
}
