package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// killPortHolder finds and kills any process holding the given TCP port on localhost.
// Used so that repeat `auth login` attempts don't fail when a prior callback server
// was interrupted and left the socket in TIME_WAIT or actively bound.
func killPortHolder(port int) {
	// Quick check: is the port free already?
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		_ = ln.Close()
		return
	}

	// Find PIDs holding the port and send SIGKILL.
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port)).Output()
	if err != nil {
		return
	}
	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pid := range pids {
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		_ = exec.Command("kill", "-9", pid).Run()
	}
	// Give the OS a moment to release the port.
	time.Sleep(200 * time.Millisecond)
}

// CallbackServer handles OAuth callback
type CallbackServer struct {
	port   int
	server *http.Server
	code   chan string
	err    chan error
}

// NewCallbackServer creates a new callback server
func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{
		port: port,
		code: make(chan string, 1),
		err:  make(chan error, 1),
	}
}

// Start begins listening for the OAuth callback
func (s *CallbackServer) Start(expectedState string) error {
	// Auto-kill any process holding our port from a previous failed auth attempt
	killPortHolder(s.port)

	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check for error
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			s.err <- fmt.Errorf("authorization denied: %s", errParam)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Failed</title></head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
<h1>Authorization Failed</h1>
<p>You denied access to the application.</p>
<p>You can close this window.</p>
</body>
</html>`)
			return
		}

		// Verify state
		state := r.URL.Query().Get("state")
		if state != expectedState {
			s.err <- fmt.Errorf("state mismatch: expected %s, got %s", expectedState, state)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Failed</title></head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
<h1>Authorization Failed</h1>
<p>Security validation failed. Please try again.</p>
<p>You can close this window.</p>
</body>
</html>`)
			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			s.err <- fmt.Errorf("no authorization code received")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Failed</title></head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
<h1>Authorization Failed</h1>
<p>No authorization code received.</p>
<p>You can close this window.</p>
</body>
</html>`)
			return
		}

		// Success!
		s.code <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Authorization Successful</title></head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
<h1>Authorization Successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
	})

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.err <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// WaitForCode blocks until an authorization code is received or timeout
func (s *CallbackServer) WaitForCode(timeout time.Duration) (string, error) {
	select {
	case code := <-s.code:
		return code, nil
	case err := <-s.err:
		return "", err
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

// Stop shuts down the callback server
func (s *CallbackServer) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetRedirectURI returns the redirect URI for OAuth
func (s *CallbackServer) GetRedirectURI() string {
	return fmt.Sprintf("http://localhost:%d/callback", s.port)
}
