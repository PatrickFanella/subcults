// Package main is the entry point for the Jetstream indexer.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq" // Postgres driver
	"github.com/onnwee/subcults/internal/db"
	"github.com/onnwee/subcults/internal/indexer"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	help := flag.Bool("help", false, "display help message")
	flag.Parse()

	if *help {
		fmt.Println("Subcults Jetstream Indexer")
		fmt.Println()
		fmt.Println("Usage: indexer [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Initialize logger
	// Defaults to production mode (JSON format) if SUBCULT_ENV not set
	env := os.Getenv("SUBCULT_ENV")
	if env == "" {
		env = "production"
	}
	logger := middleware.NewLogger(env)
	slog.SetDefault(logger)

	// Get metrics port from environment or default to 9090
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}

	// Get internal auth token from environment (optional)
	internalToken := os.Getenv("INTERNAL_AUTH_TOKEN")

	// Initialize Prometheus registry and metrics
	reg := prometheus.NewRegistry()
	metrics := indexer.NewMetrics()
	if err := metrics.Register(reg); err != nil {
		logger.Error("failed to register metrics", "error", err)
		os.Exit(1)
	}

	// Initialize database slow query metrics
	dbMetrics := db.NewSlowQueryMetrics()
	if err := dbMetrics.Register(reg); err != nil {
		logger.Error("failed to register db metrics", "error", err)
		os.Exit(1)
	}
	logger.Info("database slow query metrics registered")

	// Create HTTP server for metrics
	mux := http.NewServeMux()
	metricsHandler := indexer.MetricsHandler(reg)
	if internalToken != "" {
		metricsHandler = indexer.InternalAuthMiddleware(internalToken)(metricsHandler)
	}
	mux.Handle("/internal/indexer/metrics", metricsHandler)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
			logger.Error("failed to write health response", "error", err)
		}
	})

	metricsServer := &http.Server{
		Addr:         ":" + metricsPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start metrics server in goroutine
	go func() {
		logger.Info("starting metrics server", "port", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server error", "error", err)
			os.Exit(1)
		}
	}()

	// Initialize Jetstream client with backpressure handling
	jetstreamURL := os.Getenv("JETSTREAM_URL")
	if jetstreamURL == "" {
		jetstreamURL = "wss://jetstream1.us-east.bsky.network/subscribe"
		logger.Warn("JETSTREAM_URL not set, using default", "url", jetstreamURL)
	}

	config := indexer.DefaultConfig(jetstreamURL)

	// Initialize repository based on DATABASE_URL environment variable
	var repo indexer.RecordRepository
	var sequenceTracker indexer.SequenceTracker
	var cleanupService interface {
		Start(context.Context)
		Stop()
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		// Use Postgres repository when DATABASE_URL is provided
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			logger.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		// Test connection
		if err := db.Ping(); err != nil {
			logger.Error("failed to ping database", "error", err)
			os.Exit(1)
		}

		logger.Info("using Postgres repository", "database_url", databaseURL)
		repo = indexer.NewPostgresRecordRepository(db, logger)
		sequenceTracker = indexer.NewPostgresSequenceTracker(db, logger)

		// Use Postgres cleanup service
		cleanupConfig := indexer.DefaultCleanupConfig()
		cleanupService = indexer.NewCleanupService(db, logger, cleanupConfig)
	} else {
		// Fall back to in-memory repository for testing
		logger.Warn("DATABASE_URL not set, using in-memory repository (data will not persist)")
		memRepo := indexer.NewInMemoryRecordRepository(logger)
		repo = memRepo
		sequenceTracker = indexer.NewInMemorySequenceTracker(logger)
		cleanupConfig := indexer.DefaultCleanupConfig()
		cleanupService = indexer.NewInMemoryCleanupService(memRepo, logger, cleanupConfig)
	}

	filter := indexer.NewRecordFilter(indexer.NewFilterMetrics())

	// Create main context for the application
	// All child contexts will be derived from this for proper shutdown coordination
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Start cleanup service with app context
	cleanupService.Start(appCtx)

	// Message handler - now with transactional database persistence and sequence tracking
	handler := func(messageType int, payload []byte) error {
		start := time.Now()

		// Decode CBOR message to extract timestamp for lag calculation
		msg, err := indexer.DecodeCBORMessage(payload)
		if err != nil {
			// Log error with context but don't fail - continue processing
			logger.Debug("failed to decode message for lag calculation",
				slog.String("error", err.Error()))
		} else if msg != nil && msg.TimeUS > 0 {
			// Calculate processing lag: current time - message timestamp
			messageTime := time.Unix(0, msg.TimeUS*1000) // Convert microseconds to nanoseconds
			lag := time.Since(messageTime)
			metrics.SetProcessingLag(lag.Seconds())

			logger.Debug("processing message",
				slog.String("kind", msg.Kind),
				slog.Duration("lag", lag))
		}

		// Filter and validate the record
		result := filter.FilterCBOR(payload)
		metrics.IncMessagesProcessed()

		// If record doesn't match our lexicon, skip (but still update sequence)
		if !result.Matched {
			// Update sequence even for non-matched records to avoid re-processing
			if msg != nil && msg.TimeUS > 0 {
				if err := sequenceTracker.UpdateSequence(appCtx, msg.TimeUS); err != nil {
					logger.Warn("failed to update sequence for non-matched record",
						slog.String("error", err.Error()))
				}
			}
			return nil
		}

		// If record failed validation, log and increment error counter
		if !result.Valid {
			metrics.IncMessagesError()
			logger.Warn("record validation failed",
				slog.String("collection", result.Collection),
				slog.String("did", result.DID),
				slog.String("rkey", result.RKey),
				slog.String("error", result.Error.Error()))
			// Update sequence even for invalid records to avoid re-processing
			if msg != nil && msg.TimeUS > 0 {
				if err := sequenceTracker.UpdateSequence(appCtx, msg.TimeUS); err != nil {
					logger.Warn("failed to update sequence for invalid record",
						slog.String("error", err.Error()))
				}
			}
			return nil // Don't fail the entire stream for validation errors
		}

		// Handle delete operations
		if result.Operation == "delete" {
			if err := repo.DeleteRecord(appCtx, result.DID, result.Collection, result.RKey); err != nil {
				metrics.IncDatabaseWritesFailed()
				logger.Error("failed to delete record",
					slog.String("collection", result.Collection),
					slog.String("did", result.DID),
					slog.String("rkey", result.RKey),
					slog.String("error", err.Error()))
				return nil // Don't fail stream on delete errors
			}
			logger.Info("record deleted",
				slog.String("collection", result.Collection),
				slog.String("did", result.DID),
				slog.String("rkey", result.RKey))

			// Update sequence after successful delete
			if msg != nil && msg.TimeUS > 0 {
				if err := sequenceTracker.UpdateSequence(appCtx, msg.TimeUS); err != nil {
					logger.Error("failed to update sequence after delete",
						slog.String("error", err.Error()))
				}
			}
			return nil
		}

		// Upsert record with transaction support
		recordID, isNew, err := repo.UpsertRecord(appCtx, &result)
		if err != nil {
			metrics.IncDatabaseWritesFailed()
			logger.Error("failed to upsert record",
				slog.String("collection", result.Collection),
				slog.String("did", result.DID),
				slog.String("rkey", result.RKey),
				slog.String("error", err.Error()))
			return nil // Don't fail stream on upsert errors
		}

		// If record was skipped due to idempotency, don't count as upsert
		if recordID == "" {
			logger.Debug("record skipped (idempotent)",
				slog.String("collection", result.Collection),
				slog.String("did", result.DID),
				slog.String("rkey", result.RKey))
			// Update sequence even for idempotent records to avoid re-processing
			if msg != nil && msg.TimeUS > 0 {
				if err := sequenceTracker.UpdateSequence(appCtx, msg.TimeUS); err != nil {
					logger.Warn("failed to update sequence for idempotent record",
						slog.String("error", err.Error()))
				}
			}
			return nil
		}

		// Record successful upsert
		metrics.IncUpserts()
		logger.Info("record upserted",
			slog.String("record_id", recordID),
			slog.String("collection", result.Collection),
			slog.String("did", result.DID),
			slog.String("rkey", result.RKey),
			slog.Bool("is_new", isNew))

		// Record ingestion latency
		metrics.ObserveIngestLatency(time.Since(start).Seconds())

		// Update sequence after successful processing
		if msg != nil && msg.TimeUS > 0 {
			if err := sequenceTracker.UpdateSequence(appCtx, msg.TimeUS); err != nil {
				logger.Error("failed to update sequence after upsert",
					slog.String("error", err.Error()))
			}
		}

		return nil
	}

	client, err := indexer.NewClientWithSequenceTracker(config, handler, logger, metrics, sequenceTracker)
	if err != nil {
		logger.Error("failed to create jetstream client", "error", err)
		os.Exit(1)
	}

	// Log resume status on startup
	lastSeq, err := sequenceTracker.GetLastSequence(appCtx)
	if err != nil {
		logger.Warn("failed to get last sequence on startup", "error", err)
	} else if lastSeq > 0 {
		logger.Info("will resume from last processed sequence",
			slog.Int64("cursor", lastSeq),
			slog.Time("last_message_time", time.Unix(0, lastSeq*1000)))
	} else {
		logger.Info("starting from beginning (no previous sequence found)")
	}

	// Start Jetstream client in background
	clientDone := make(chan error, 1)
	go func() {
		logger.Info("starting jetstream client", "url", jetstreamURL)
		// Client uses app context for proper shutdown coordination
		clientDone <- client.Run(appCtx)
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("received shutdown signal")
	case err := <-clientDone:
		logger.Error("jetstream client exited unexpectedly", "error", err)
	}

	logger.Info("shutting down indexer...")

	// Cancel app context to trigger graceful shutdown of all components
	// This will stop: cleanup service, client, and any in-flight DB operations
	appCancel()

	// Stop cleanup service
	logger.Info("stopping cleanup service")
	cleanupService.Stop()

	// Wait for client to finish with longer timeout to account for drain
	select {
	case <-clientDone:
		logger.Info("jetstream client stopped")
	case <-time.After(15 * time.Second):
		logger.Warn("jetstream client shutdown timeout exceeded")
	}

	// Create context with timeout for metrics server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := metricsServer.Shutdown(ctx); err != nil {
		logger.Error("metrics server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("indexer stopped")
}
