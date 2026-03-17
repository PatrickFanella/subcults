// Package audit provides audit logging functionality for tracking access to
// sensitive endpoints and operations for compliance and incident response.
package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"
)

// ExportFormat defines supported export formats.
type ExportFormat string

const (
	// ExportFormatCSV exports logs as comma-separated values.
	ExportFormatCSV ExportFormat = "csv"
	// ExportFormatJSON exports logs as JSON array.
	ExportFormatJSON ExportFormat = "json"
)

// ExportOptions configures audit log export parameters.
type ExportOptions struct {
	Format    ExportFormat // Export format (csv or json)
	From      time.Time    // Start of time range (inclusive)
	To        time.Time    // End of time range (inclusive)
	UserDID   string       // Filter by user DID (optional)
	Limit     int          // Maximum number of entries to export (0 = no limit)
}

// ExportLogs exports audit logs matching the given options.
// Returns the exported data as bytes in the specified format.
func ExportLogs(repo Repository, opts ExportOptions) ([]byte, error) {
	// Validate format
	if opts.Format != ExportFormatCSV && opts.Format != ExportFormatJSON {
		return nil, fmt.Errorf("unsupported export format: %s", opts.Format)
	}

	// Query logs based on filters
	// Note: Query without limit first, then filter by time range, then apply limit
	// This ensures we get the correct number of results after time filtering
	var logs []*AuditLog
	var err error

	if opts.UserDID != "" {
		// Export for specific user - query without limit first
		logs, err = repo.QueryByUser(opts.UserDID, 0)
	} else {
		// Export all logs (would need a new repository method)
		// For now, we'll use a high limit as a workaround
		// In production, this should be a proper QueryAll with pagination
		return nil, fmt.Errorf("export all logs not yet implemented - use UserDID filter")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	// Filter by time range if specified
	if !opts.From.IsZero() || !opts.To.IsZero() {
		logs = filterByTimeRange(logs, opts.From, opts.To)
	}

	// Apply limit after time filtering to get correct number of results
	if opts.Limit > 0 && len(logs) > opts.Limit {
		logs = logs[:opts.Limit]
	}

	// Export in requested format
	switch opts.Format {
	case ExportFormatCSV:
		return exportToCSV(logs)
	case ExportFormatJSON:
		return exportToJSON(logs)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", opts.Format)
	}
}

// filterByTimeRange filters logs to only include entries within the time range.
func filterByTimeRange(logs []*AuditLog, from, to time.Time) []*AuditLog {
	var filtered []*AuditLog
	for _, log := range logs {
		// Check if log is within range
		if !from.IsZero() && log.CreatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && log.CreatedAt.After(to) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

// exportToCSV exports audit logs to CSV format.
func exportToCSV(logs []*AuditLog) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write header
	header := []string{
		"ID",
		"Timestamp (UTC)",
		"User DID",
		"Entity Type",
		"Entity ID",
		"Action",
		"Outcome",
		"Request ID",
		"IP Address",
		"User Agent",
		"Previous Hash",
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, log := range logs {
		row := []string{
			log.ID,
			log.CreatedAt.Format(time.RFC3339),
			log.UserDID,
			log.EntityType,
			log.EntityID,
			log.Action,
			log.Outcome,
			log.RequestID,
			log.IPAddress,
			log.UserAgent,
			log.PreviousHash,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// exportToJSON exports audit logs to JSON format.
func exportToJSON(logs []*AuditLog) ([]byte, error) {
	// Create a structure suitable for JSON export
	type exportLog struct {
		ID           string    `json:"id"`
		Timestamp    string    `json:"timestamp"` // ISO 8601 format
		UserDID      string    `json:"user_did"`
		EntityType   string    `json:"entity_type"`
		EntityID     string    `json:"entity_id"`
		Action       string    `json:"action"`
		Outcome      string    `json:"outcome"`
		RequestID    string    `json:"request_id,omitempty"`
		IPAddress    string    `json:"ip_address,omitempty"`
		UserAgent    string    `json:"user_agent,omitempty"`
		PreviousHash string    `json:"previous_hash,omitempty"`
	}

	exportLogs := make([]exportLog, len(logs))
	for i, log := range logs {
		exportLogs[i] = exportLog{
			ID:           log.ID,
			Timestamp:    log.CreatedAt.Format(time.RFC3339),
			UserDID:      log.UserDID,
			EntityType:   log.EntityType,
			EntityID:     log.EntityID,
			Action:       log.Action,
			Outcome:      log.Outcome,
			RequestID:    log.RequestID,
			IPAddress:    log.IPAddress,
			UserAgent:    log.UserAgent,
			PreviousHash: log.PreviousHash,
		}
	}

	data, err := json.MarshalIndent(exportLogs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return data, nil
}
