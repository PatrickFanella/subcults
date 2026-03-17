// Package main is the entry point for the backfill command.
// It provides historical AT Protocol data ingestion from Jetstream replay
// and CAR file imports with checkpoint-based resume capability.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/onnwee/subcults/internal/backfill"
	"github.com/onnwee/subcults/internal/indexer"
	"github.com/onnwee/subcults/internal/middleware"
)

func main() {
	// CLI flags
	source := flag.String("source", "jetstream", "backfill source: 'jetstream' or 'car'")
	startTS := flag.Int64("start-ts", 0, "start timestamp in microseconds (Jetstream mode)")
	endTS := flag.Int64("end-ts", 0, "end timestamp in microseconds (Jetstream mode, 0 = now)")
	carPath := flag.String("car-file", "", "path to CAR file (CAR mode)")
	batchSize := flag.Int("batch", 1000, "records per checkpoint batch")
	dryRun := flag.Bool("dry-run", false, "validate without writing to database")
	resume := flag.Bool("resume", true, "resume from last checkpoint if available")
	help := flag.Bool("help", false, "display help message")
	flag.Parse()

	if *help {
		fmt.Println("Subcults Backfill Tool")
		fmt.Println()
		fmt.Println("Ingests historical AT Protocol data from Jetstream replay or CAR files.")
		fmt.Println("Supports checkpoint-based resume and dry-run validation.")
		fmt.Println()
		fmt.Println("Usage: backfill [options]")
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # Backfill from Jetstream with time range")
		fmt.Println("  backfill --source=jetstream --start-ts=1700000000000000")
		fmt.Println()
		fmt.Println("  # Import from CAR file")
		fmt.Println("  backfill --source=car --car-file=export.car")
		fmt.Println()
		fmt.Println("  # Dry run to validate records")
		fmt.Println("  backfill --source=jetstream --start-ts=1700000000000000 --dry-run")
		os.Exit(0)
	}

	// Initialize logger
	env := os.Getenv("SUBCULT_ENV")
	if env == "" {
		env = "development"
	}
	logger := middleware.NewLogger(env)
	slog.SetDefault(logger)

	// Validate flags
	if *source != "jetstream" && *source != "car" {
		logger.Error("invalid source, must be 'jetstream' or 'car'", "source", *source)
		os.Exit(1)
	}
	if *source == "car" && *carPath == "" {
		logger.Error("--car-file is required when source is 'car'")
		os.Exit(1)
	}
	if *source == "jetstream" && *startTS == 0 {
		logger.Error("--start-ts is required when source is 'jetstream'")
		os.Exit(1)
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	cancel()

	// Initialize indexer components
	repo := indexer.NewPostgresRecordRepository(db, logger)
	filterMetrics := indexer.NewFilterMetrics()
	filter := indexer.NewRecordFilter(filterMetrics)

	// Initialize checkpoint store
	checkpointStore := backfill.NewPostgresCheckpointStore(db, logger)

	// Build config
	cfg := backfill.Config{
		Source:    *source,
		StartTS:   *startTS,
		EndTS:     *endTS,
		CARPath:   *carPath,
		BatchSize: *batchSize,
		DryRun:    *dryRun,
		Resume:    *resume,
		Logger:    logger,
	}

	// Create runner
	runner := backfill.NewRunner(cfg, repo, filter, checkpointStore)

	// Setup graceful shutdown
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("received signal, stopping backfill...", "signal", sig)
		cancel()
	}()

	logger.Info("starting backfill",
		"source", *source,
		"start_ts", *startTS,
		"end_ts", *endTS,
		"batch_size", *batchSize,
		"dry_run", *dryRun,
		"resume", *resume,
	)

	// Run backfill
	result, err := runner.Run(ctx)
	if err != nil {
		logger.Error("backfill failed", "error", err)
		os.Exit(1)
	}

	logger.Info("backfill completed",
		"records_processed", result.RecordsProcessed,
		"records_skipped", result.RecordsSkipped,
		"errors", result.Errors,
		"duration", result.Duration,
	)
}
