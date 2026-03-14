package rift

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// Server is the Rift HTTP server that proxies Docker API requests to the local
// Docker daemon and enforces authentication on every request.
type Server struct {
	cfg          *Config
	auth         *Authenticator
	orchestrator *Orchestrator
	httpServer   *http.Server
}

// NewServer creates a Server. The auth token must already be resolved (not a URI).
func NewServer(cfg *Config, resolvedToken string, orchestrator *Orchestrator) *Server {
	s := &Server{
		cfg:          cfg,
		auth:         NewAuthenticator(resolvedToken),
		orchestrator: orchestrator,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.Handle("/", s.auth.Middleware(http.HandlerFunc(s.handleDockerProxy)))

	s.httpServer = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 5 * time.Minute, // long for image pulls
		IdleTimeout:  120 * time.Second,
	}
	return s
}

// ListenAndServe starts the HTTP server. It blocks until the server stops.
func (s *Server) ListenAndServe() error {
	log.Printf("rift server: listening on %s", s.cfg.ListenAddr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the resolved listen address (useful when port 0 is used in tests).
func (s *Server) Addr() string {
	return s.cfg.ListenAddr
}

// handleHealth is an unauthenticated liveness probe.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	state := s.orchestrator.State()
	if state == StateRunning {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	fmt.Fprintf(w, "rift state: %s\n", state)
}

// handleDockerProxy reverse-proxies authenticated requests to the Docker daemon.
func (s *Server) handleDockerProxy(w http.ResponseWriter, r *http.Request) {
	// Wake Rift if it is off (on-demand start).
	if err := s.orchestrator.WakeUp(r.Context()); err != nil {
		http.Error(w, "rift: failed to wake server", http.StatusServiceUnavailable)
		return
	}

	if s.orchestrator.State() != StateRunning {
		http.Error(w, "rift: server not yet running", http.StatusServiceUnavailable)
		return
	}

	s.orchestrator.TrackConnection()
	defer s.orchestrator.TrackDisconnect()

	proxy := s.newDockerProxy()
	proxy.ServeHTTP(w, r)
}

// newDockerProxy builds a reverse proxy to the Docker daemon socket.
func (s *Server) newDockerProxy() *httputil.ReverseProxy {
	target := &url.URL{
		Scheme: "http",
		Host:   "docker",
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", s.cfg.DockerSocketPath)
		},
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	return proxy
}
