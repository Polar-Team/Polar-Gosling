package rift

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// LifecycleHooks are callbacks invoked by the Orchestrator at each state transition.
// Implementations provision/deprovision the actual Rift VM via OpenTofu or cloud SDK.
type LifecycleHooks interface {
	// OnStart is called when the orchestrator wants to bring the Rift VM up.
	OnStart(ctx context.Context) error
	// OnStop is called when the orchestrator wants to gracefully stop the Rift VM.
	// The implementation should flush caches before returning.
	OnStop(ctx context.Context) error
	// OnDestroy is called when the orchestrator wants to permanently destroy the VM.
	OnDestroy(ctx context.Context) error
}

// Orchestrator manages the Rift server lifecycle: scheduled windows, on-demand
// wake, idle shutdown, cache sync, and anti-flap protection.
type Orchestrator struct {
	cfg          *Config
	state        *StateManager
	cache        *ImageCache
	hooks        LifecycleHooks
	activeConns  atomic.Int64
	mu           sync.Mutex
	stopCh       chan struct{}
	wakeRequests chan struct{}
}

// NewOrchestrator creates an Orchestrator. hooks may be nil in tests.
func NewOrchestrator(cfg *Config, cache *ImageCache, hooks LifecycleHooks) *Orchestrator {
	return &Orchestrator{
		cfg:          cfg,
		state:        NewStateManager(cfg.AntiFlap),
		cache:        cache,
		hooks:        hooks,
		stopCh:       make(chan struct{}),
		wakeRequests: make(chan struct{}, 8),
	}
}

// State returns the current lifecycle state.
func (o *Orchestrator) State() State {
	return o.state.Current()
}

// ActiveConnections returns the number of currently connected runners.
func (o *Orchestrator) ActiveConnections() int64 {
	return o.activeConns.Load()
}

// WakeUp signals that a runner needs Rift. If Rift is off, it will be started.
// If a shutdown is pending (anti-flap), the pending shutdown is cancelled.
func (o *Orchestrator) WakeUp(ctx context.Context) error {
	// Cancel any pending shutdown — a new request arrived.
	if o.state.PendingShutdown() {
		o.state.CancelPendingShutdown()
		log.Printf("rift orchestrator: pending shutdown cancelled (new request)")
		return nil
	}

	current := o.state.Current()
	if current == StateRunning || current == StateStarting {
		return nil
	}

	// Signal the run loop to start.
	select {
	case o.wakeRequests <- struct{}{}:
	default:
	}
	return nil
}

// TrackConnection increments the active connection counter.
// Call TrackDisconnect when the runner disconnects.
func (o *Orchestrator) TrackConnection() {
	o.activeConns.Add(1)
}

// TrackDisconnect decrements the active connection counter.
func (o *Orchestrator) TrackDisconnect() {
	o.activeConns.Add(-1)
}

// Run starts the orchestration loop. It blocks until ctx is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
	cacheTicker := time.NewTicker(o.cfg.CacheSyncInterval)
	idleTicker := time.NewTicker(10 * time.Second) // check idle every 10s
	defer cacheTicker.Stop()
	defer idleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return o.shutdown(context.Background())

		case <-o.wakeRequests:
			if err := o.ensureRunning(ctx); err != nil {
				log.Printf("rift orchestrator: start failed: %v", err)
			}

		case <-cacheTicker.C:
			if o.state.Current() == StateRunning {
				if err := o.cache.Sync(ctx); err != nil {
					log.Printf("rift orchestrator: cache sync error: %v", err)
				}
			}

		case <-idleTicker.C:
			o.checkIdle(ctx)
		}
	}
}

// ensureRunning transitions to StateRunning if not already there.
func (o *Orchestrator) ensureRunning(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	current := o.state.Current()
	if current == StateRunning || current == StateStarting {
		return nil
	}

	o.state.Transition(StateStarting)
	log.Printf("rift orchestrator: starting Rift VM")

	if o.hooks != nil {
		if err := o.hooks.OnStart(ctx); err != nil {
			o.state.Transition(StateOff)
			return fmt.Errorf("rift orchestrator: OnStart: %w", err)
		}
	}

	o.state.Transition(StateRunning)
	log.Printf("rift orchestrator: Rift VM is running")
	return nil
}

// checkIdle shuts down Rift if it has been idle longer than IdleTimeout.
func (o *Orchestrator) checkIdle(ctx context.Context) {
	if o.state.Current() != StateRunning {
		return
	}
	if o.activeConns.Load() > 0 {
		return
	}
	if time.Since(o.state.EnteredAt()) < o.cfg.IdleTimeout {
		return
	}

	log.Printf("rift orchestrator: idle timeout reached, initiating shutdown")
	if err := o.shutdown(ctx); err != nil {
		log.Printf("rift orchestrator: idle shutdown error: %v", err)
	}
}

// shutdown syncs the cache and stops the Rift VM.
func (o *Orchestrator) shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.state.Current() == StateOff {
		return nil
	}

	// Attempt anti-flap transition; if blocked, mark pending and return.
	if !o.state.Transition(StateStopping) {
		log.Printf("rift orchestrator: shutdown blocked by anti-flap delay")
		return nil
	}

	log.Printf("rift orchestrator: syncing cache before shutdown")
	if err := o.cache.Sync(ctx); err != nil {
		log.Printf("rift orchestrator: cache sync on shutdown: %v", err)
	}

	if o.hooks != nil {
		if err := o.hooks.OnStop(ctx); err != nil {
			log.Printf("rift orchestrator: OnStop error: %v", err)
		}
	}

	o.state.Transition(StateOff)
	log.Printf("rift orchestrator: Rift VM stopped")
	return nil
}
