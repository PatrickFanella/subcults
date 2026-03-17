// Package indexer provides filtering and processing of AT Protocol records
// for the Subcults Jetstream indexer.
package indexer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/onnwee/subcults/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

var (
	// ErrRecordExists is returned when attempting to insert a duplicate record.
	ErrRecordExists = errors.New("record already exists")

	// ErrTransactionFailed is returned when a transaction cannot be completed.
	ErrTransactionFailed = errors.New("transaction failed")
)

// RecordRepository provides transactional database operations for AT Protocol records.
type RecordRepository interface {
	// UpsertRecord atomically inserts or updates a record with idempotency.
	// Returns the record ID and a boolean indicating if it was newly created (true) or updated (false).
	UpsertRecord(ctx context.Context, record *FilterResult) (string, bool, error)

	// DeleteRecord atomically removes a record.
	DeleteRecord(ctx context.Context, did, collection, rkey string) error

	// CheckIdempotencyKey verifies if an operation has already been processed.
	CheckIdempotencyKey(ctx context.Context, key string) (bool, error)
}

// PostgresRecordRepository implements RecordRepository using PostgreSQL with full transaction support.
type PostgresRecordRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresRecordRepository creates a new PostgresRecordRepository.
func NewPostgresRecordRepository(db *sql.DB, logger *slog.Logger) *PostgresRecordRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresRecordRepository{
		db:     db,
		logger: logger,
	}
}

// UpsertRecord atomically inserts or updates a record with full transaction support.
// This implements the all-or-nothing requirement with idempotency.
func (r *PostgresRecordRepository) UpsertRecord(ctx context.Context, record *FilterResult) (string, bool, error) {
	// Start tracing span for the entire upsert operation
	ctx, endSpan := tracing.StartDBSpan(ctx, "ingestion_idempotency", tracing.DBOperationExec)
	defer func() {
		if err := recover(); err != nil {
			endSpan(fmt.Errorf("panic during upsert: %v", err))
			panic(err)
		}
	}()

	// Add trace attributes
	tracing.SetAttributes(ctx,
		attribute.String("collection", record.Collection),
		attribute.String("did", record.DID),
		attribute.String("rkey", record.RKey),
	)

	if !record.Valid || !record.Matched {
		err := fmt.Errorf("invalid or unmatched record")
		endSpan(err)
		return "", false, err
	}

	// Generate idempotency key from DID + Collection + RKey + Rev
	idempotencyKey := generateIdempotencyKey(record.DID, record.Collection, record.RKey, record.Rev)

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		r.logger.Error("failed to begin transaction",
			slog.String("error", err.Error()),
			slog.String("did", record.DID),
			slog.String("collection", record.Collection),
			slog.String("rkey", record.RKey))
		endSpan(err)
		return "", false, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always attempt rollback on function exit (no-op after successful commit)
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Warn("failed to rollback transaction",
				slog.String("error", err.Error()))
		}
	}()

	// Check idempotency - if we've already processed this exact record revision, skip
	var existingKey string
	checkQuery := `
		SELECT idempotency_key FROM ingestion_idempotency 
		WHERE idempotency_key = $1
	`
	err = tx.QueryRowContext(ctx, checkQuery, idempotencyKey).Scan(&existingKey)
	if err == nil {
		// Record already processed, commit and return existing state
		if err := tx.Commit(); err != nil {
			r.logger.Error("failed to commit idempotency check",
				slog.String("error", err.Error()))
			endSpan(err)
			return "", false, fmt.Errorf("failed to commit: %w", err)
		}
		r.logger.Debug("skipping duplicate record (already processed)",
			slog.String("idempotency_key", idempotencyKey),
			slog.String("did", record.DID),
			slog.String("collection", record.Collection),
			slog.String("rkey", record.RKey))
		tracing.AddEvent(ctx, "duplicate_record_skipped")
		endSpan(nil)
		return "", false, nil // Not an error, just already processed
	} else if err != sql.ErrNoRows {
		// Unexpected error
		r.logger.Error("failed to check idempotency",
			slog.String("error", err.Error()))
		endSpan(err)
		return "", false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// Route to appropriate table based on collection
	var recordID string
	var isNew bool

	switch record.Collection {
	case CollectionScene:
		recordID, isNew, err = r.upsertScene(ctx, tx, record)
	case CollectionEvent:
		recordID, isNew, err = r.upsertEvent(ctx, tx, record)
	case CollectionPost:
		recordID, isNew, err = r.upsertPost(ctx, tx, record)
	case CollectionAlliance:
		recordID, isNew, err = r.upsertAlliance(ctx, tx, record)
	default:
		// For now, only support the three main collections
		// Additional collections (membership, alliance, stream) will be added later
		err = fmt.Errorf("unsupported collection: %s", record.Collection)
	}

	if err != nil {
		r.logger.Error("failed to upsert record",
			slog.String("error", err.Error()),
			slog.String("collection", record.Collection))
		endSpan(err)
		return "", false, fmt.Errorf("failed to upsert %s: %w", record.Collection, err)
	}

	// Store idempotency key to prevent reprocessing
	insertIdempotency := `
		INSERT INTO ingestion_idempotency (idempotency_key, did, collection, rkey, rev, record_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err = tx.ExecContext(ctx, insertIdempotency, idempotencyKey, record.DID, record.Collection, record.RKey, record.Rev, recordID)
	if err != nil {
		r.logger.Error("failed to store idempotency key",
			slog.String("error", err.Error()))
		endSpan(err)
		return "", false, fmt.Errorf("failed to store idempotency key: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		r.logger.Error("failed to commit transaction",
			slog.String("error", err.Error()))
		endSpan(err)
		return "", false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Log successful transaction
	r.logger.Info("record upserted successfully",
		slog.String("record_id", recordID),
		slog.String("collection", record.Collection),
		slog.String("did", record.DID),
		slog.String("rkey", record.RKey),
		slog.Bool("is_new", isNew),
		slog.String("idempotency_key", idempotencyKey))

	// Add trace event and attributes for successful upsert
	tracing.AddEvent(ctx, "record_upserted",
		attribute.String("record_id", recordID),
		attribute.Bool("is_new", isNew))
	endSpan(nil)

	return recordID, isNew, nil
}

// DeleteRecord atomically soft-deletes a record with transaction support.
// Note: Idempotency keys are NOT cleaned up on delete. This is intentional to prevent
// re-ingestion of deleted records. If a record is deleted and then the same revision
// is replayed from Jetstream, it will be correctly skipped due to the existing
// idempotency key. This protects against accidental re-ingestion of deleted content.
// Uses soft delete (UPDATE ... SET deleted_at = NOW()) to match repository patterns.
func (r *PostgresRecordRepository) DeleteRecord(ctx context.Context, did, collection, rkey string) error {
	// Start tracing span
	ctx, endSpan := tracing.StartDBSpan(ctx, collection, tracing.DBOperationDelete)
	defer func() {
		if err := recover(); err != nil {
			endSpan(fmt.Errorf("panic during delete: %v", err))
			panic(err)
		}
	}()

	// Add trace attributes
	tracing.SetAttributes(ctx,
		attribute.String("collection", collection),
		attribute.String("did", did),
		attribute.String("rkey", rkey),
	)

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		r.logger.Error("failed to begin transaction for delete",
			slog.String("error", err.Error()))
		endSpan(err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always attempt rollback on function exit (no-op after successful commit)
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Warn("failed to rollback delete transaction",
				slog.String("error", err.Error()))
		}
	}()

	// Soft delete from appropriate table
	var query string
	switch collection {
	case CollectionScene:
		query = `UPDATE scenes SET deleted_at = NOW(), updated_at = NOW() WHERE record_did = $1 AND record_rkey = $2 AND deleted_at IS NULL`
	case CollectionEvent:
		query = `UPDATE events SET deleted_at = NOW(), updated_at = NOW() WHERE record_did = $1 AND record_rkey = $2 AND deleted_at IS NULL`
	case CollectionPost:
		query = `UPDATE posts SET deleted_at = NOW(), updated_at = NOW() WHERE record_did = $1 AND record_rkey = $2 AND deleted_at IS NULL`
	case CollectionAlliance:
		query = `UPDATE alliances SET deleted_at = NOW(), updated_at = NOW() WHERE record_did = $1 AND record_rkey = $2 AND deleted_at IS NULL`
	default:
		err := fmt.Errorf("unsupported collection for delete: %s", collection)
		endSpan(err)
		return err
	}

	result, err := tx.ExecContext(ctx, query, did, rkey)
	if err != nil {
		r.logger.Error("failed to delete record",
			slog.String("error", err.Error()))
		endSpan(err)
		return fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	// Commit transaction
	if err = tx.Commit(); err != nil {
		r.logger.Error("failed to commit delete transaction",
			slog.String("error", err.Error()))
		endSpan(err)
		return fmt.Errorf("failed to commit: %w", err)
	}

	r.logger.Info("record soft-deleted successfully",
		slog.String("collection", collection),
		slog.String("did", did),
		slog.String("rkey", rkey),
		slog.Int64("rows_affected", rowsAffected))

	tracing.AddEvent(ctx, "record_deleted",
		attribute.Int64("rows_affected", rowsAffected))
	endSpan(nil)

	return nil
}

// CheckIdempotencyKey verifies if an operation has already been processed.
func (r *PostgresRecordRepository) CheckIdempotencyKey(ctx context.Context, key string) (bool, error) {
	ctx, endSpan := tracing.StartDBSpan(ctx, "ingestion_idempotency", tracing.DBOperationQuery)
	
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM ingestion_idempotency WHERE idempotency_key = $1)`
	err := r.db.QueryRowContext(ctx, query, key).Scan(&exists)
	if err != nil {
		endSpan(err)
		return false, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	tracing.SetAttributes(ctx,
		attribute.Bool("exists", exists))

	endSpan(nil)
	return exists, nil
}

// upsertScene handles scene-specific upsert logic.
// Maps AT Protocol record to domain Scene model and persists to database.
func (r *PostgresRecordRepository) upsertScene(ctx context.Context, tx *sql.Tx, record *FilterResult) (string, bool, error) {
	// Map AT Protocol record to domain model
	domainScene, err := MapSceneRecord(record)
	if err != nil {
		return "", false, fmt.Errorf("failed to map scene record: %w", err)
	}

	// Check if record exists (including soft-deleted)
	var existingID string
	var deletedAt sql.NullTime
	checkQuery := `SELECT id, deleted_at FROM scenes WHERE record_did = $1 AND record_rkey = $2`
	err = tx.QueryRowContext(ctx, checkQuery, record.DID, record.RKey).Scan(&existingID, &deletedAt)

	if err == sql.ErrNoRows {
		// Insert new scene
		newID := uuid.New().String()
		insertQuery := `
			INSERT INTO scenes (
				id, name, description, owner_did, allow_precise, precise_point, 
				coarse_geohash, tags, visibility, palette, record_did, record_rkey,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				CASE
					WHEN $6 IS NOT NULL AND $7 IS NOT NULL
						THEN ST_SetSRID(ST_MakePoint($6, $7), 4326)
					ELSE NULL
				END,
				$8, $9, $10, $11, $12, $13, NOW(), NOW()
			)
		`

		// Prepare point coordinates (nullable)
		// Note: ST_MakePoint expects (longitude, latitude) order
		var lng, lat *float64
		if domainScene.PrecisePoint != nil && domainScene.AllowPrecise {
			lng = &domainScene.PrecisePoint.Lng
			lat = &domainScene.PrecisePoint.Lat
		}

		// Prepare palette JSON (nullable)
		// Align with database default '{}'::jsonb when no palette is provided
		var paletteJSON []byte
		if domainScene.Palette != nil {
			paletteJSON, err = json.Marshal(domainScene.Palette)
			if err != nil {
				return "", false, fmt.Errorf("failed to marshal palette: %w", err)
			}
		} else {
			paletteJSON = []byte("{}")
		}

		_, err = tx.ExecContext(ctx, insertQuery,
			newID,
			domainScene.Name,
			domainScene.Description,
			domainScene.OwnerDID,
			domainScene.AllowPrecise,
			lng, lat, // ST_MakePoint(lng, lat) - longitude first, latitude second
			domainScene.CoarseGeohash,
			domainScene.Tags,
			domainScene.Visibility,
			paletteJSON,
			record.DID,
			record.RKey,
		)
		if err != nil {
			return "", false, fmt.Errorf("failed to insert scene: %w", err)
		}
		return newID, true, nil
	} else if err != nil {
		return "", false, fmt.Errorf("failed to check scene existence: %w", err)
	}

	// Update existing scene
	updateQuery := `
		UPDATE scenes SET
			name = $2,
			description = $3,
			allow_precise = $4,
			precise_point = CASE
				WHEN $5 IS NOT NULL AND $6 IS NOT NULL
					THEN ST_SetSRID(ST_MakePoint($5, $6), 4326)
				ELSE NULL
			END,
			coarse_geohash = $7,
			tags = $8,
			visibility = $9,
			palette = $10,
			updated_at = NOW(),
			deleted_at = NULL
		WHERE id = $1
	`

	// Prepare point coordinates (nullable)
	// Note: ST_MakePoint expects (longitude, latitude) order
	var lng, lat *float64
	if domainScene.PrecisePoint != nil && domainScene.AllowPrecise {
		lng = &domainScene.PrecisePoint.Lng
		lat = &domainScene.PrecisePoint.Lat
	}

	// Prepare palette JSON (nullable)
	// Align with database default '{}'::jsonb when no palette is provided
	var paletteJSON []byte
	if domainScene.Palette != nil {
		paletteJSON, err = json.Marshal(domainScene.Palette)
		if err != nil {
			return "", false, fmt.Errorf("failed to marshal palette: %w", err)
		}
	} else {
		paletteJSON = []byte("{}")
	}

	_, err = tx.ExecContext(ctx, updateQuery,
		existingID,
		domainScene.Name,
		domainScene.Description,
		domainScene.AllowPrecise,
		lng, lat, // ST_MakePoint(lng, lat) - longitude first, latitude second
		domainScene.CoarseGeohash,
		domainScene.Tags,
		domainScene.Visibility,
		paletteJSON,
	)
	if err != nil {
		return "", false, fmt.Errorf("failed to update scene: %w", err)
	}

	return existingID, false, nil
}

// upsertEvent handles event-specific upsert logic.
// Maps AT Protocol record to domain Event model and persists to database.
// Performs scene_id lookup from sceneId reference in the record.
func (r *PostgresRecordRepository) upsertEvent(ctx context.Context, tx *sql.Tx, record *FilterResult) (string, bool, error) {
	// Map AT Protocol record to domain model
	domainEvent, err := MapEventRecord(record)
	if err != nil {
		return "", false, fmt.Errorf("failed to map event record: %w", err)
	}

	// Parse the AT Protocol record to get sceneId for lookup
	var atProtoEvent struct {
		SceneID string `json:"sceneId"`
	}
	if err := json.Unmarshal(record.Record, &atProtoEvent); err != nil {
		return "", false, fmt.Errorf("failed to parse event record for sceneId: %w", err)
	}

	// Lookup scene UUID by sceneId (which is a record identifier, not a UUID)
	// The sceneId in AT Protocol records is typically the scene's record_rkey
	var sceneUUID string
	sceneQuery := `SELECT id FROM scenes WHERE record_rkey = $1 AND deleted_at IS NULL LIMIT 1`
	err = tx.QueryRowContext(ctx, sceneQuery, atProtoEvent.SceneID).Scan(&sceneUUID)
	if err == sql.ErrNoRows {
		return "", false, fmt.Errorf("scene not found: sceneId=%s", atProtoEvent.SceneID)
	} else if err != nil {
		return "", false, fmt.Errorf("failed to lookup scene: %w", err)
	}

	// Check if event exists (including soft-deleted)
	var existingID string
	var deletedAt sql.NullTime
	checkQuery := `SELECT id, deleted_at FROM events WHERE record_did = $1 AND record_rkey = $2`
	err = tx.QueryRowContext(ctx, checkQuery, record.DID, record.RKey).Scan(&existingID, &deletedAt)

	if err == sql.ErrNoRows {
		// Insert new event
		newID := uuid.New().String()
		insertQuery := `
			INSERT INTO events (
				id, scene_id, title, description, allow_precise, precise_point,
				coarse_geohash, tags, status, starts_at, ends_at,
				record_did, record_rkey, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				CASE
					WHEN $6 IS NOT NULL AND $7 IS NOT NULL
						THEN ST_SetSRID(ST_MakePoint($6, $7), 4326)
					ELSE NULL
				END,
				$8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
			)
		`

		// Prepare point coordinates (nullable)
		// Note: ST_MakePoint expects (longitude, latitude) order
		var lng, lat *float64
		if domainEvent.PrecisePoint != nil && domainEvent.AllowPrecise {
			lng = &domainEvent.PrecisePoint.Lng
			lat = &domainEvent.PrecisePoint.Lat
		}

		_, err = tx.ExecContext(ctx, insertQuery,
			newID,
			sceneUUID,
			domainEvent.Title,
			domainEvent.Description,
			domainEvent.AllowPrecise,
			lng, lat, // ST_MakePoint(lng, lat) - longitude first, latitude second
			domainEvent.CoarseGeohash,
			domainEvent.Tags,
			domainEvent.Status,
			domainEvent.StartsAt,
			domainEvent.EndsAt,
			record.DID,
			record.RKey,
		)
		if err != nil {
			return "", false, fmt.Errorf("failed to insert event: %w", err)
		}
		return newID, true, nil
	} else if err != nil {
		return "", false, fmt.Errorf("failed to check event existence: %w", err)
	}

	// Update existing event
	updateQuery := `
		UPDATE events SET
			scene_id = $2,
			title = $3,
			description = $4,
			allow_precise = $5,
			precise_point = CASE
				WHEN $6 IS NOT NULL AND $7 IS NOT NULL
					THEN ST_SetSRID(ST_MakePoint($6, $7), 4326)
				ELSE NULL
			END,
			coarse_geohash = $8,
			tags = $9,
			status = $10,
			starts_at = $11,
			ends_at = $12,
			updated_at = NOW(),
			deleted_at = NULL
		WHERE id = $1
	`

	// Prepare point coordinates (nullable)
	// Note: ST_MakePoint expects (longitude, latitude) order
	var lng, lat *float64
	if domainEvent.PrecisePoint != nil && domainEvent.AllowPrecise {
		lng = &domainEvent.PrecisePoint.Lng
		lat = &domainEvent.PrecisePoint.Lat
	}

	_, err = tx.ExecContext(ctx, updateQuery,
		existingID,
		sceneUUID,
		domainEvent.Title,
		domainEvent.Description,
		domainEvent.AllowPrecise,
		lng, lat, // ST_MakePoint(lng, lat) - longitude first, latitude second
		domainEvent.CoarseGeohash,
		domainEvent.Tags,
		domainEvent.Status,
		domainEvent.StartsAt,
		domainEvent.EndsAt,
	)
	if err != nil {
		return "", false, fmt.Errorf("failed to update event: %w", err)
	}

	return existingID, false, nil
}

// upsertPost handles post-specific upsert logic.
// Maps AT Protocol record to domain Post model and persists to database.
// Performs scene_id and/or event_id lookup from identifiers in the record.
func (r *PostgresRecordRepository) upsertPost(ctx context.Context, tx *sql.Tx, record *FilterResult) (string, bool, error) {
	// Map AT Protocol record to domain model
	domainPost, err := MapPostRecord(record)
	if err != nil {
		return "", false, fmt.Errorf("failed to map post record: %w", err)
	}

	// Parse the AT Protocol record to get sceneId and/or eventId for lookup
	var atProtoPost struct {
		SceneID *string `json:"sceneId,omitempty"`
		EventID *string `json:"eventId,omitempty"`
	}
	if err := json.Unmarshal(record.Record, &atProtoPost); err != nil {
		return "", false, fmt.Errorf("failed to parse post record for references: %w", err)
	}

	// Lookup scene UUID if provided
	var sceneUUID *string
	if atProtoPost.SceneID != nil {
		var sceneID string
		sceneQuery := `SELECT id FROM scenes WHERE record_rkey = $1 AND deleted_at IS NULL LIMIT 1`
		err = tx.QueryRowContext(ctx, sceneQuery, *atProtoPost.SceneID).Scan(&sceneID)
		if err == sql.ErrNoRows {
			return "", false, fmt.Errorf("scene not found: sceneId=%s", *atProtoPost.SceneID)
		} else if err != nil {
			return "", false, fmt.Errorf("failed to lookup scene: %w", err)
		}
		sceneUUID = &sceneID
	}

	// Lookup event UUID if provided
	var eventUUID *string
	if atProtoPost.EventID != nil {
		var eventID string
		eventQuery := `SELECT id FROM events WHERE record_rkey = $1 AND deleted_at IS NULL LIMIT 1`
		err = tx.QueryRowContext(ctx, eventQuery, *atProtoPost.EventID).Scan(&eventID)
		if err == sql.ErrNoRows {
			return "", false, fmt.Errorf("event not found: eventId=%s", *atProtoPost.EventID)
		} else if err != nil {
			return "", false, fmt.Errorf("failed to lookup event: %w", err)
		}
		eventUUID = &eventID
	}

	// Check if post exists (including soft-deleted)
	var existingID string
	var deletedAt sql.NullTime
	checkQuery := `SELECT id, deleted_at FROM posts WHERE record_did = $1 AND record_rkey = $2`
	err = tx.QueryRowContext(ctx, checkQuery, record.DID, record.RKey).Scan(&existingID, &deletedAt)

	if err == sql.ErrNoRows {
		// Insert new post
		newID := uuid.New().String()
		insertQuery := `
			INSERT INTO posts (
				id, scene_id, event_id, author_did, text, attachments, labels,
				record_did, record_rkey, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`

		// Prepare attachments JSON
		var attachmentsJSON []byte
		if len(domainPost.Attachments) > 0 {
			attachmentsJSON, err = json.Marshal(domainPost.Attachments)
			if err != nil {
				return "", false, fmt.Errorf("failed to marshal attachments: %w", err)
			}
		} else {
			attachmentsJSON = []byte("[]")
		}

		_, err = tx.ExecContext(ctx, insertQuery,
			newID,
			sceneUUID,
			eventUUID,
			domainPost.AuthorDID,
			domainPost.Text,
			attachmentsJSON,
			domainPost.Labels,
			record.DID,
			record.RKey,
		)
		if err != nil {
			return "", false, fmt.Errorf("failed to insert post: %w", err)
		}
		return newID, true, nil
	} else if err != nil {
		return "", false, fmt.Errorf("failed to check post existence: %w", err)
	}

	// Update existing post
	updateQuery := `
		UPDATE posts SET
			scene_id = $2,
			event_id = $3,
			text = $4,
			attachments = $5,
			labels = $6,
			updated_at = NOW(),
			deleted_at = NULL
		WHERE id = $1
	`

	// Prepare attachments JSON
	var attachmentsJSON []byte
	if len(domainPost.Attachments) > 0 {
		attachmentsJSON, err = json.Marshal(domainPost.Attachments)
		if err != nil {
			return "", false, fmt.Errorf("failed to marshal attachments: %w", err)
		}
	} else {
		attachmentsJSON = []byte("[]")
	}

	_, err = tx.ExecContext(ctx, updateQuery,
		existingID,
		sceneUUID,
		eventUUID,
		domainPost.Text,
		attachmentsJSON,
		domainPost.Labels,
	)
	if err != nil {
		return "", false, fmt.Errorf("failed to update post: %w", err)
	}

	return existingID, false, nil
}

// upsertAlliance handles alliance-specific upsert logic.
// Maps AT Protocol record to domain Alliance model and persists to database.
// Performs scene_id lookups from fromSceneId and toSceneId references.
func (r *PostgresRecordRepository) upsertAlliance(ctx context.Context, tx *sql.Tx, record *FilterResult) (string, bool, error) {
	// Map AT Protocol record to domain model
	domainAlliance, err := MapAllianceRecord(record)
	if err != nil {
		return "", false, fmt.Errorf("failed to map alliance record: %w", err)
	}

	// Check if alliance exists first (including soft-deleted)
	var existingID string
	var deletedAt sql.NullTime
	checkQuery := `SELECT id, deleted_at FROM alliances WHERE record_did = $1 AND record_rkey = $2`
	err = tx.QueryRowContext(ctx, checkQuery, record.DID, record.RKey).Scan(&existingID, &deletedAt)

	// Parse the AT Protocol record to get fromSceneId and toSceneId for lookup
	// Only perform lookups once, shared by both insert and update paths
	var atProtoAlliance struct {
		FromSceneID string `json:"fromSceneId"`
		ToSceneID   string `json:"toSceneId"`
	}
	if err2 := json.Unmarshal(record.Record, &atProtoAlliance); err2 != nil {
		return "", false, fmt.Errorf("failed to parse alliance record for references: %w", err2)
	}

	// Lookup from_scene_id UUID
	var fromSceneUUID string
	fromSceneQuery := `SELECT id FROM scenes WHERE record_rkey = $1 AND deleted_at IS NULL LIMIT 1`
	err2 := tx.QueryRowContext(ctx, fromSceneQuery, atProtoAlliance.FromSceneID).Scan(&fromSceneUUID)
	if err2 == sql.ErrNoRows {
		return "", false, fmt.Errorf("from_scene not found: fromSceneId=%s", atProtoAlliance.FromSceneID)
	} else if err2 != nil {
		return "", false, fmt.Errorf("failed to lookup from_scene: %w", err2)
	}

	// Lookup to_scene_id UUID
	var toSceneUUID string
	toSceneQuery := `SELECT id FROM scenes WHERE record_rkey = $1 AND deleted_at IS NULL LIMIT 1`
	err2 = tx.QueryRowContext(ctx, toSceneQuery, atProtoAlliance.ToSceneID).Scan(&toSceneUUID)
	if err2 == sql.ErrNoRows {
		return "", false, fmt.Errorf("to_scene not found: toSceneId=%s", atProtoAlliance.ToSceneID)
	} else if err2 != nil {
		return "", false, fmt.Errorf("failed to lookup to_scene: %w", err2)
	}

	if err == sql.ErrNoRows {
		// Insert new alliance
		newID := uuid.New().String()
		insertQuery := `
			INSERT INTO alliances (
				id, from_scene_id, to_scene_id, weight, status, reason, since,
				record_did, record_rkey, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`

		_, err = tx.ExecContext(ctx, insertQuery,
			newID,
			fromSceneUUID,
			toSceneUUID,
			domainAlliance.Weight,
			domainAlliance.Status,
			domainAlliance.Reason,
			domainAlliance.Since,
			record.DID,
			record.RKey,
		)
		if err != nil {
			return "", false, fmt.Errorf("failed to insert alliance: %w", err)
		}
		return newID, true, nil
	} else if err != nil {
		return "", false, fmt.Errorf("failed to check alliance existence: %w", err)
	}

	// Update existing alliance
	updateQuery := `
		UPDATE alliances SET
			from_scene_id = $2,
			to_scene_id = $3,
			weight = $4,
			status = $5,
			reason = $6,
			since = $7,
			updated_at = NOW(),
			deleted_at = NULL
		WHERE id = $1
	`

	_, err = tx.ExecContext(ctx, updateQuery,
		existingID,
		fromSceneUUID,
		toSceneUUID,
		domainAlliance.Weight,
		domainAlliance.Status,
		domainAlliance.Reason,
		domainAlliance.Since,
	)
	if err != nil {
		return "", false, fmt.Errorf("failed to update alliance: %w", err)
	}

	return existingID, false, nil
}

// generateIdempotencyKey creates a deterministic key from record metadata.
// Format: SHA256(did + "\x00" + collection + "\x00" + rkey + "\x00" + rev)
// Uses NUL (\x00) separators to ensure unambiguous parsing (components cannot contain NUL bytes).
func generateIdempotencyKey(did, collection, rkey, rev string) string {
	// Build preimage with NUL separators for unambiguous hashing
	data := make([]byte, 0, len(did)+len(collection)+len(rkey)+len(rev)+3) // 3 NUL separators
	data = append(data, did...)
	data = append(data, 0)
	data = append(data, collection...)
	data = append(data, 0)
	data = append(data, rkey...)
	data = append(data, 0)
	data = append(data, rev...)

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// InMemoryRecordRepository provides an in-memory implementation for testing.
type InMemoryRecordRepository struct {
	mu              sync.RWMutex
	records         map[string]*FilterResult
	recordIDs       map[string]string // Maps composite key to stable record ID
	idempotencyKeys map[string]bool
	logger          *slog.Logger
}

// NewInMemoryRecordRepository creates a new in-memory repository.
func NewInMemoryRecordRepository(logger *slog.Logger) *InMemoryRecordRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &InMemoryRecordRepository{
		records:         make(map[string]*FilterResult),
		recordIDs:       make(map[string]string),
		idempotencyKeys: make(map[string]bool),
		logger:          logger,
	}
}

// UpsertRecord implements the interface for in-memory storage.
func (r *InMemoryRecordRepository) UpsertRecord(ctx context.Context, record *FilterResult) (string, bool, error) {
	if !record.Valid || !record.Matched {
		return "", false, fmt.Errorf("invalid or unmatched record")
	}

	idempotencyKey := generateIdempotencyKey(record.DID, record.Collection, record.RKey, record.Rev)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check idempotency
	if r.idempotencyKeys[idempotencyKey] {
		r.logger.Debug("skipping duplicate record",
			slog.String("idempotency_key", idempotencyKey))
		return "", false, nil
	}

	// Generate composite key
	key := fmt.Sprintf("%s:%s:%s", record.DID, record.Collection, record.RKey)

	// Check if record exists and get/create stable ID
	recordID, exists := r.recordIDs[key]
	if !exists {
		recordID = uuid.New().String()
		r.recordIDs[key] = recordID
	}

	// Store a deep copy of the record to avoid external mutation and data races
	copyRecord := *record
	if record.Record != nil {
		copyRecord.Record = append([]byte(nil), record.Record...)
	}
	r.records[key] = &copyRecord
	r.idempotencyKeys[idempotencyKey] = true

	r.logger.Info("record upserted in memory",
		slog.String("record_id", recordID),
		slog.Bool("is_new", !exists))

	return recordID, !exists, nil
}

// DeleteRecord implements the interface for in-memory storage.
// Note: Idempotency keys are NOT cleaned up on delete, consistent with Postgres behavior.
// This prevents re-ingestion of deleted records if the same revision is replayed.
func (r *InMemoryRecordRepository) DeleteRecord(ctx context.Context, did, collection, rkey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", did, collection, rkey)
	delete(r.records, key)
	r.logger.Info("record deleted from memory",
		slog.String("did", did),
		slog.String("collection", collection),
		slog.String("rkey", rkey))
	return nil
}

// CheckIdempotencyKey implements the interface for in-memory storage.
func (r *InMemoryRecordRepository) CheckIdempotencyKey(ctx context.Context, key string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.idempotencyKeys[key], nil
}
