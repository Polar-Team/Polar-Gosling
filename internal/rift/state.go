package rift

import (
	"sync"
	"time"
)

// StateManager tracks the current lifecycle state of the Rift server and
// enforces the anti-flap delay before allowing transitions to off/destroy.
type StateManager struct {
	mu              sync.RWMutex
	current         State
	enteredAt       time.Time
	antiFlap        time.Duration
	pendingShutdown bool
}

// NewStateManager creates a StateManager starting in StateOff.
func NewStateManager(antiFlap time.Duration) *StateManager {
	return &StateManager{
		current:   StateOff,
		enteredAt: time.Now(),
		antiFlap:  antiFlap,
	}
}

// Current returns the current state.
func (s *StateManager) Current() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// EnteredAt returns when the current state was entered.
func (s *StateManager) EnteredAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enteredAt
}

// Transition attempts to move to the target state.
// Transitions to StateOff or StateDestroying are blocked until the anti-flap
// delay has elapsed since the server entered StateRunning.
// Returns true if the transition was applied, false if blocked by anti-flap.
func (s *StateManager) Transition(target State) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == target {
		return true
	}

	// Anti-flap: block off/destroy transitions if we just became running.
	if (target == StateOff || target == StateDestroying) && s.current == StateRunning {
		elapsed := time.Since(s.enteredAt)
		if elapsed < s.antiFlap {
			s.pendingShutdown = true
			return false
		}
	}

	s.current = target
	s.enteredAt = time.Now()
	s.pendingShutdown = false
	return true
}

// PendingShutdown reports whether a shutdown was requested but blocked by anti-flap.
func (s *StateManager) PendingShutdown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pendingShutdown
}

// CancelPendingShutdown clears a pending shutdown (e.g. a new request arrived).
func (s *StateManager) CancelPendingShutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingShutdown = false
}

// AntiFlap returns the configured anti-flap duration.
func (s *StateManager) AntiFlap() time.Duration {
	return s.antiFlap
}
