package transfer

import "fmt"

// State represents a transfer's lifecycle state.
type State string

const (
	StateRequested  State = "Requested"
	StateValidating State = "Validating"
	StateAnalyzing  State = "Analyzing"
	StateApproved   State = "Approved"
	StateFundsPosted State = "FundsPosted"
	StateCompleted  State = "Completed"
	StateRejected   State = "Rejected"
	StateReturned   State = "Returned"
)

// validTransitions defines the allowed state transitions.
var validTransitions = map[State][]State{
	StateRequested:   {StateValidating},
	StateValidating:  {StateRejected, StateAnalyzing},
	StateAnalyzing:   {StateRejected, StateApproved},
	StateApproved:    {StateFundsPosted},
	StateFundsPosted: {StateCompleted, StateReturned},
	StateCompleted:   {StateReturned}, // return after settlement
}

// CanTransition returns true if transitioning from src to dst is valid.
func CanTransition(src, dst State) bool {
	allowed, ok := validTransitions[src]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == dst {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is invalid.
func ValidateTransition(src, dst State) error {
	if !CanTransition(src, dst) {
		return fmt.Errorf("invalid transition from %s to %s", src, dst)
	}
	return nil
}
