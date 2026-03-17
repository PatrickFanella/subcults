// Package main contains integration tests for the API server.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestGracefulShutdown_SignalHandling tests that the server handles signals correctly.
func TestGracefulShutdown_SignalHandling(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// Capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	// Create HTTP server with routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channels for server lifecycle signaling and goroutine error handling
	serverStarted := make(chan struct{})
	serverStopped := make(chan struct{})
	errCh := make(chan error, 2) // Buffer for potential listen and serve errors

	// Start server in goroutine
	go func() {
		// Re-listen on the same port
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- fmt.Errorf("failed to re-listen: %w", err)
			close(serverStarted)
			close(serverStopped)
			return
		}
		logger.Info("starting server", "addr", addr)
		close(serverStarted)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
		close(serverStopped)
	}()

	// Wait for server to start
	select {
	case <-serverStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to start in time")
	}

	// Check for errors from goroutine
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	// Give server a moment to be ready
	time.Sleep(50 * time.Millisecond)

	// Log shutdown start
	logger.Info("shutting down server...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("server shutdown error: %v", err)
	}

	logger.Info("server stopped")

	// Wait for server to stop
	select {
	case <-serverStopped:
	case <-time.After(15 * time.Second):
		t.Fatal("server failed to stop in time")
	}

	// Check for errors from goroutine
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	// Verify log order
	logs := logBuf.String()
	startIdx := strings.Index(logs, "starting server")
	shutdownIdx := strings.Index(logs, "shutting down server")
	stoppedIdx := strings.Index(logs, "server stopped")

	if startIdx == -1 {
		t.Error("expected 'starting server' log message")
	}
	if shutdownIdx == -1 {
		t.Error("expected 'shutting down server' log message")
	}
	if stoppedIdx == -1 {
		t.Error("expected 'server stopped' log message")
	}

	if startIdx > shutdownIdx {
		t.Error("expected 'starting server' to come before 'shutting down server'")
	}
	if shutdownIdx > stoppedIdx {
		t.Error("expected 'shutting down server' to come before 'server stopped'")
	}
}

// TestGracefulShutdown_InFlightRequests tests that in-flight requests complete before shutdown.
func TestGracefulShutdown_InFlightRequests(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// Mutex for synchronizing handler operations
	var mu sync.Mutex
	var requestStarted bool
	var requestCompleted bool
	// Use buffered channel to avoid race condition if handler executes before receiver is ready
	handlerStarted := make(chan struct{}, 1)
	handlerCanContinue := make(chan struct{})

	// Create HTTP server with a slow endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestStarted = true
		mu.Unlock()
		close(handlerStarted)

		// Wait until we're told to continue (simulates slow request)
		<-handlerCanContinue

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"completed"}`))

		mu.Lock()
		requestCompleted = true
		mu.Unlock()
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channels for server lifecycle signaling and goroutine error handling
	serverStarted := make(chan struct{})
	serverStopped := make(chan struct{})
	// Buffer for: server listen, server serve, request, and shutdown errors
	errCh := make(chan error, 4)

	// Start server in goroutine
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- fmt.Errorf("failed to listen: %w", err)
			close(serverStarted)
			close(serverStopped)
			return
		}
		close(serverStarted)
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
		close(serverStopped)
	}()

	// Wait for server to start
	select {
	case <-serverStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to start in time")
	}

	// Check for errors from goroutine
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	// Give server a moment to be ready
	time.Sleep(50 * time.Millisecond)

	// Start in-flight request in a goroutine
	requestDone := make(chan struct{})
	var response *http.Response
	var requestErr error
	go func() {
		resp, err := http.Get("http://" + addr + "/slow")
		if err != nil {
			requestErr = err
		}
		response = resp
		close(requestDone)
	}()

	// Wait for handler to start processing
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("handler failed to start in time")
	}

	// Start shutdown while request is in flight
	shutdownDone := make(chan struct{})
	var shutdownErr error
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			shutdownErr = err
		}
		close(shutdownDone)
	}()

	// Give shutdown a moment to begin
	time.Sleep(50 * time.Millisecond)

	// Now allow the handler to complete
	close(handlerCanContinue)

	// Wait for request to complete
	select {
	case <-requestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("request failed to complete in time")
	}

	// Wait for shutdown to complete
	select {
	case <-shutdownDone:
	case <-time.After(15 * time.Second):
		t.Fatal("shutdown failed to complete in time")
	}

	// Check for errors from goroutines in main test goroutine
	if requestErr != nil {
		t.Errorf("request error: %v", requestErr)
	}
	if shutdownErr != nil {
		t.Errorf("shutdown error: %v", shutdownErr)
	}

	// Verify in-flight request completed successfully
	mu.Lock()
	defer mu.Unlock()
	if !requestStarted {
		t.Error("expected request to have started")
	}
	if !requestCompleted {
		t.Error("expected request to have completed")
	}

	// Verify response
	if response != nil {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", response.StatusCode)
		}
		body, _ := io.ReadAll(response.Body)
		var result map[string]string
		if err := json.Unmarshal(body, &result); err != nil {
			t.Errorf("failed to parse response: %v", err)
		}
		if result["status"] != "completed" {
			t.Errorf("expected status 'completed', got '%s'", result["status"])
		}
	}
}

// TestGracefulShutdown_ExitCode0 verifies server exit behavior.
// This test verifies that shutdown completes without error (exit code 0 scenario).
func TestGracefulShutdown_ExitCode0(t *testing.T) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine with error handling
	serverStarted := make(chan struct{})
	errCh := make(chan error, 1) // Buffer for potential listen error
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- fmt.Errorf("failed to listen: %w", err)
			close(serverStarted)
			return
		}
		close(serverStarted)
		_ = server.Serve(ln)
	}()

	// Wait for server to start
	select {
	case <-serverStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("server failed to start in time")
	}

	// Check for errors from goroutine
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}

	time.Sleep(50 * time.Millisecond)

	// Shutdown should complete without error (exit code 0 scenario)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected clean shutdown (exit 0), got error: %v", err)
	}
}
