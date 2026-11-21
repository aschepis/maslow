// Package progress describes the events a run reports while it works, so the
// CLI can show them without the verify and audit layers writing to a terminal.
package progress

import "time"

// Phase says what an Event marks.
type Phase int

const (
	// Plan comes first and carries Total, the number of items the run will report.
	Plan Phase = iota
	// Start marks an item beginning.
	Start
	// Finish marks an item ending, with its outcome.
	Finish
)

// Event is one step of a run. Kind and Name match the item's evidence; an
// item such as a contract or the policy rules may finish with several statuses.
type Event struct {
	Phase    Phase
	Total    int
	Kind     string
	Name     string
	Statuses []string
	Attempts int
	Duration time.Duration
}

// Func receives events. A nil Func ignores them.
type Func func(Event)

// Emit sends e to f unless f is nil.
func (f Func) Emit(e Event) {
	if f != nil {
		f(e)
	}
}

// Step reports run as one item: a Start, then a Finish with the statuses and
// attempt count run returns and the time it took.
func (f Func) Step(kind, name string, run func() (statuses []string, attempts int)) {
	f.Emit(Event{Phase: Start, Kind: kind, Name: name})
	began := time.Now()
	statuses, attempts := run()
	f.Emit(Event{Phase: Finish, Kind: kind, Name: name, Statuses: statuses, Attempts: attempts, Duration: time.Since(began)})
}
