package transfer

import (
	"testing"
)

func TestValidTransitions(t *testing.T) {
	validCases := [][2]State{
		{StateRequested, StateValidating},
		{StateValidating, StateRejected},
		{StateValidating, StateAnalyzing},
		{StateAnalyzing, StateRejected},
		{StateAnalyzing, StateApproved},
		{StateApproved, StateFundsPosted},
		{StateFundsPosted, StateCompleted},
		{StateFundsPosted, StateReturned},
		{StateCompleted, StateReturned},
	}

	for _, tc := range validCases {
		if !CanTransition(tc[0], tc[1]) {
			t.Errorf("expected valid transition %s -> %s", tc[0], tc[1])
		}
	}
}

func TestInvalidTransitions(t *testing.T) {
	invalidCases := [][2]State{
		{StateRequested, StateApproved},
		{StateRequested, StateCompleted},
		{StateValidating, StateCompleted},
		{StateAnalyzing, StateFundsPosted},
		{StateApproved, StateCompleted},
		{StateCompleted, StateRequested},
		{StateRejected, StateApproved},
		{StateReturned, StateCompleted},
	}

	for _, tc := range invalidCases {
		if CanTransition(tc[0], tc[1]) {
			t.Errorf("expected invalid transition %s -> %s", tc[0], tc[1])
		}
	}
}

func TestTransferTransition(t *testing.T) {
	tr := &Transfer{
		ID:    "test-1",
		State: StateRequested,
	}

	if err := tr.Transition(StateValidating); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
	if tr.State != StateValidating {
		t.Errorf("expected Validating, got %s", tr.State)
	}

	if err := tr.Transition(StateRequested); err == nil {
		t.Error("expected error for invalid transition Validating -> Requested")
	}
}
