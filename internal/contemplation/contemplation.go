// Package contemplation implements the mandatory 15-minute i386 Neural Root
// synchronisation that must complete before the control-plane (AI Teacher) may
// issue tasks to the runner agent (IA Student) over the Micro-Bus.
//
// During the Contemplation Period the system verifies its understanding of core
// i386 architectural concepts — GDT, IDT, Memory Segments, Paging, and the
// 32-bit instruction-set limits — and re-reads the Open386 toolchain
// specifications to ensure its curriculum is technically sound for a 386/486
// target.  Progress is emitted to an io.Writer so that a WinStratch progress
// dialog (or a plain terminal) can render the "Synchronizing Neural Root..."
// bar with per-second concept descriptions.
//
// Kernel-lock contract:
//
//	A Lock is created at startup and Run is called in a background goroutine.
//	Any code path that would access the Micro-Bus or dispatch tasks must call
//	Ready() / <-Done() first.  While the lock is held, SubmitRun returns
//	codes.Unavailable with a descriptive message.
package contemplation

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// DefaultDuration is the mandatory contemplation period length (15 minutes).
const DefaultDuration = 15 * time.Minute

// concept describes a single i386 architectural topic re-read and verified
// during the contemplation pass.
type concept struct {
	startSec int
	endSec   int
	subject  string
	detail   string
}

// schedule is the ordered sequence of i386 topics covered during the 900-second
// contemplation window.  The ranges must be contiguous and cover [0, 900).
var schedule = []concept{
	{
		startSec: 0,
		endSec:   180,
		subject:  "GDT (Global Descriptor Table)",
		detail: "Re-reading segment descriptor layout: base[31:0], limit[19:0], " +
			"DPL[1:0], P, S, type[3:0], G, D/B, L flags for 32-bit protected mode.",
	},
	{
		startSec: 180,
		endSec:   360,
		subject:  "IDT (Interrupt Descriptor Table)",
		detail: "Verifying gate descriptors: Interrupt/Trap/Task gates, " +
			"ISR entry points, privilege levels, and 256-vector layout on i386.",
	},
	{
		startSec: 360,
		endSec:   540,
		subject:  "Memory Segments and Paging (CR3/CR0)",
		detail: "Confirming segmented flat model, CR0.PE=1, CR0.PG=1, " +
			"CR3 page-directory base, 4 KB and 4 MB page frames on i386.",
	},
	{
		startSec: 540,
		endSec:   720,
		subject:  "32-bit Instruction Set Limits",
		detail: "Auditing instruction width (32-bit operand/address prefixes), " +
			"absence of 64-bit REX prefix, LOCK/REP prefixes, FPU x87 ops.",
	},
	{
		startSec: 720,
		endSec:   900,
		subject:  "Open386 Toolchain Curriculum Review",
		detail: "Re-reading Open386 compiler/linker flags: -m32, -march=i386, " +
			"ld script load address 0x100000, flat binary vs ELF32 targets.",
	},
}

// ConceptForSecond returns the contemplation subject and detail description
// that apply at the given elapsed second (0-based).  It can be used by UI
// code (e.g. a WinStratch progress dialog) to render per-second descriptions.
func ConceptForSecond(sec int, totalSec int) (subject, detail string) {
	for _, c := range schedule {
		// Scale fixed schedule to the configured total duration.
		scaledStart := c.startSec * totalSec / 900
		scaledEnd := c.endSec * totalSec / 900
		if sec >= scaledStart && sec < scaledEnd {
			return c.subject, c.detail
		}
	}
	return "Finalising Neural Root synchronisation", "Releasing Neural Registry lock."
}

// Lock is a one-shot startup gate.  Create it with New, call Run in a
// goroutine, and use Ready or <-Done() to wait for the contemplation period to
// end before accessing the Micro-Bus.
type Lock struct {
	out      io.Writer
	duration time.Duration

	once      sync.Once
	done      chan struct{} // closed when contemplation completes
	startedAt time.Time
	mu        sync.Mutex // guards startedAt
}

// New returns a Lock configured to run for d.
// Progress lines are written to out (os.Stdout if nil).
func New(d time.Duration, out io.Writer) *Lock {
	if out == nil {
		out = os.Stdout
	}
	return &Lock{
		out:      out,
		duration: d,
		done:     make(chan struct{}),
	}
}

// Run executes the full contemplation pass, printing one progress line per
// second to l.out.  It blocks until the period expires and then closes the
// Done channel so that waiters are unblocked.
//
// Run must be called exactly once per Lock; subsequent calls are no-ops.
func (l *Lock) Run() {
	l.once.Do(func() {
		l.mu.Lock()
		l.startedAt = time.Now()
		l.mu.Unlock()

		totalSec := int(l.duration.Seconds())
		if totalSec <= 0 {
			close(l.done)
			return
		}

		fmt.Fprintf(l.out,
			"[Neural Root]   0%% (  0/%d) Synchronizing Neural Root — Contemplation Period begins.\n",
			totalSec)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for elapsed := 1; elapsed <= totalSec; elapsed++ {
			<-ticker.C

			pct := (elapsed * 100) / totalSec
			subject, _ := ConceptForSecond(elapsed-1, totalSec)
			fmt.Fprintf(l.out,
				"[Neural Root] %3d%% (%3d/%d) Contemplating i386: %s\n",
				pct, elapsed, totalSec, subject)
		}

		fmt.Fprintf(l.out,
			"[Neural Root] 100%% (%d/%d) Neural Registry lock released. AI Teacher may now access the Micro-Bus.\n",
			totalSec, totalSec)
		close(l.done)
	})
}

// Done returns a channel that is closed once the contemplation period ends.
// Callers can select on it or block with <-l.Done().
func (l *Lock) Done() <-chan struct{} {
	return l.done
}

// Ready reports whether the contemplation period has already ended.
func (l *Lock) Ready() bool {
	select {
	case <-l.done:
		return true
	default:
		return false
	}
}

// RemainingSeconds returns the approximate number of whole seconds remaining
// in the contemplation period, or 0 if the period has ended.
func (l *Lock) RemainingSeconds() int {
	if l.Ready() {
		return 0
	}
	l.mu.Lock()
	started := l.startedAt
	l.mu.Unlock()
	if started.IsZero() {
		return int(l.duration.Seconds())
	}
	remaining := l.duration - time.Since(started)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds())
}
