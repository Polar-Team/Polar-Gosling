package spinner

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// ANSI color codes
const (
	blue  = "\033[34m"
	cyan  = "\033[36m"
	reset = "\033[0m"
	clear = "\r\033[K"
)

// Wings-style animation frames — spreads outward like wings flapping
var frames = []string{
	"  ·  ",
	" ‹·› ",
	"‹‹·››",
	"«‹·›»",
	"‹‹·››",
	" ‹·› ",
}

// Spinner displays a wings-style animated spinner with a message.
type Spinner struct {
	mu      sync.Mutex
	writer  io.Writer
	message string
	active  bool
	done    chan struct{}
}

// New creates a new Spinner that writes to stderr.
func New(message string) *Spinner {
	return &Spinner{
		writer:  os.Stderr,
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins the spinner animation in a background goroutine.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go s.animate()
}

// Stop halts the spinner and prints a final completion message.
func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	<-s.done

	icon := cyan + "✓" + reset
	if !success {
		icon = "\033[31m✗" + reset
	}
	fmt.Fprintf(s.writer, "%s %s %s\n", clear, icon, s.message)
}

// UpdateMessage changes the spinner text while it's running.
func (s *Spinner) UpdateMessage(msg string) {
	s.mu.Lock()
	s.message = msg
	s.mu.Unlock()
}

func (s *Spinner) animate() {
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.done)

	i := 0
	for {
		s.mu.Lock()
		active := s.active
		msg := s.message
		s.mu.Unlock()

		if !active {
			return
		}

		frame := frames[i%len(frames)]
		fmt.Fprintf(s.writer, "%s%s%s%s %s", clear, blue, frame, reset, msg)

		i++
		<-ticker.C
	}
}

// Run is a convenience helper: shows the spinner while fn executes,
// then stops with success/failure based on the returned error.
func Run(message string, fn func() error) error {
	s := New(message)
	s.Start()
	err := fn()
	s.Stop(err == nil)
	return err
}
