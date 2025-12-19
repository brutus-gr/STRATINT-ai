package api

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"log/slog"
)

// AdminHandler handles admin-only operations
type AdminHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *sql.DB, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		db:     db,
		logger: logger,
	}
}

// DeleteAllData permanently deletes all events and sources from the database
func (h *AdminHandler) DeleteAllData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Warn("Admin initiated DELETE ALL DATA operation")

	ctx := r.Context()

	// Get counts before deletion
	var eventsCount, sourcesCount int64

	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&eventsCount)
	if err != nil {
		h.logger.Error("Failed to count events", "error", err)
		http.Error(w, "Failed to count events", http.StatusInternalServerError)
		return
	}

	err = h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources").Scan(&sourcesCount)
	if err != nil {
		h.logger.Error("Failed to count sources", "error", err)
		http.Error(w, "Failed to count sources", http.StatusInternalServerError)
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", "error", err)
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete all event-related data
	if _, err := tx.ExecContext(ctx, "DELETE FROM event_entities"); err != nil {
		h.logger.Error("Failed to delete event_entities", "error", err)
		http.Error(w, "Failed to delete event entities", http.StatusInternalServerError)
		return
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM event_sources"); err != nil {
		h.logger.Error("Failed to delete event_sources", "error", err)
		http.Error(w, "Failed to delete event sources", http.StatusInternalServerError)
		return
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM events"); err != nil {
		h.logger.Error("Failed to delete events", "error", err)
		http.Error(w, "Failed to delete events", http.StatusInternalServerError)
		return
	}

	// Delete all sources
	if _, err := tx.ExecContext(ctx, "DELETE FROM sources"); err != nil {
		h.logger.Error("Failed to delete sources", "error", err)
		http.Error(w, "Failed to delete sources", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.logger.Error("Failed to commit transaction", "error", err)
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully deleted all data",
		"events_deleted", eventsCount,
		"sources_deleted", sourcesCount,
	)

	// Return success response
	response := map[string]interface{}{
		"message":         "All data deleted successfully",
		"events_deleted":  eventsCount,
		"sources_deleted": sourcesCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RequeueFailedEnrichments resets failed enrichments back to pending for retry
func (h *AdminHandler) RequeueFailedEnrichments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("Admin initiated requeue of failed enrichments")

	ctx := r.Context()

	// Get counts before update
	var failedCount, pendingCount int64

	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources WHERE enrichment_status = 'failed'").Scan(&failedCount)
	if err != nil {
		h.logger.Error("Failed to count failed enrichments", "error", err)
		http.Error(w, "Failed to count failed enrichments", http.StatusInternalServerError)
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", "error", err)
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Update failed enrichments to pending
	result, err := tx.ExecContext(ctx, `
		UPDATE sources
		SET
			enrichment_status = 'pending',
			enrichment_error = NULL,
			enrichment_claimed_at = NULL
		WHERE enrichment_status = 'failed'
	`)
	if err != nil {
		h.logger.Error("Failed to update enrichment status", "error", err)
		http.Error(w, "Failed to update enrichment status", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		h.logger.Error("Failed to get rows affected", "error", err)
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		return
	}

	// Get new pending count
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources WHERE enrichment_status = 'pending'").Scan(&pendingCount)
	if err != nil {
		h.logger.Error("Failed to count pending enrichments", "error", err)
		http.Error(w, "Failed to count pending enrichments", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.logger.Error("Failed to commit transaction", "error", err)
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully requeued failed enrichments",
		"requeued_count", rowsAffected,
		"total_pending", pendingCount,
	)

	// Return success response
	response := map[string]interface{}{
		"message":        "Failed enrichments requeued successfully",
		"requeued_count": rowsAffected,
		"total_pending":  pendingCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteFailedEnrichments permanently deletes sources with failed enrichment status
func (h *AdminHandler) DeleteFailedEnrichments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("Admin initiated delete of failed enrichments")

	ctx := r.Context()

	// Get count before deletion
	var failedCount int64

	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources WHERE enrichment_status = 'failed'").Scan(&failedCount)
	if err != nil {
		h.logger.Error("Failed to count failed enrichments", "error", err)
		http.Error(w, "Failed to count failed enrichments", http.StatusInternalServerError)
		return
	}

	if failedCount == 0 {
		response := map[string]interface{}{
			"message":       "No failed enrichments to delete",
			"deleted_count": 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", "error", err)
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete sources with failed enrichment status
	result, err := tx.ExecContext(ctx, `
		DELETE FROM sources
		WHERE enrichment_status = 'failed'
	`)
	if err != nil {
		h.logger.Error("Failed to delete failed enrichments", "error", err)
		http.Error(w, "Failed to delete failed enrichments", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		h.logger.Error("Failed to get rows affected", "error", err)
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.logger.Error("Failed to commit transaction", "error", err)
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully deleted failed enrichments",
		"deleted_count", rowsAffected,
	)

	// Return success response
	response := map[string]interface{}{
		"message":       "Failed enrichments deleted successfully",
		"deleted_count": rowsAffected,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePendingSources permanently deletes sources with pending scrape status
func (h *AdminHandler) DeletePendingSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.logger.Info("Admin initiated delete of pending sources")

	ctx := r.Context()

	// Get count before deletion
	var pendingCount int64

	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sources WHERE scrape_status = 'pending'").Scan(&pendingCount)
	if err != nil {
		h.logger.Error("Failed to count pending sources", "error", err)
		http.Error(w, "Failed to count pending sources", http.StatusInternalServerError)
		return
	}

	if pendingCount == 0 {
		response := map[string]interface{}{
			"message":       "No pending sources to delete",
			"deleted_count": 0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		h.logger.Error("Failed to begin transaction", "error", err)
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete sources with pending scrape status
	result, err := tx.ExecContext(ctx, `
		DELETE FROM sources
		WHERE scrape_status = 'pending'
	`)
	if err != nil {
		h.logger.Error("Failed to delete pending sources", "error", err)
		http.Error(w, "Failed to delete pending sources", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		h.logger.Error("Failed to get rows affected", "error", err)
		http.Error(w, "Failed to get rows affected", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.logger.Error("Failed to commit transaction", "error", err)
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Successfully deleted pending sources",
		"deleted_count", rowsAffected,
	)

	// Return success response
	response := map[string]interface{}{
		"message":       "Pending sources deleted successfully",
		"deleted_count": rowsAffected,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListCloudflareDebugFiles lists all Cloudflare debug HTML files in /tmp
func (h *AdminHandler) ListCloudflareDebugFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	files, err := filepath.Glob("/tmp/cloudflare-blocked-*.html")
	if err != nil {
		h.logger.Error("Failed to list Cloudflare debug files", "error", err)
		http.Error(w, "Failed to list files", http.StatusInternalServerError)
		return
	}

	type FileInfo struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Size int64  `json:"size"`
	}

	fileInfos := make([]FileInfo, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, FileInfo{
			Name: filepath.Base(file),
			Path: file,
			Size: info.Size(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": fileInfos,
		"count": len(fileInfos),
	})
}

// DownloadCloudflareDebugFile downloads a specific Cloudflare debug HTML file
func (h *AdminHandler) DownloadCloudflareDebugFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	// URL format: /api/admin/cloudflare-debug-files/cloudflare-blocked-1234567890.html
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}
	filename := parts[len(parts)-1]

	// Validate filename to prevent directory traversal
	if !strings.HasPrefix(filename, "cloudflare-blocked-") || !strings.HasSuffix(filename, ".html") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("/tmp", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		h.logger.Error("Failed to open file", "file", filePath, "error", err)
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set headers for download
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Copy file to response
	if _, err := io.Copy(w, file); err != nil {
		h.logger.Error("Failed to send file", "file", filePath, "error", err)
		return
	}

	h.logger.Info("Cloudflare debug file downloaded", "file", filename)
}

// GetRecentEnrichments returns recent sources with their enrichment status and event IDs
func (h *AdminHandler) GetRecentEnrichments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	limit := 50 // Default limit

	// Query recent enrichments with their event IDs
	query := `
		SELECT
			s.id,
			s.type,
			s.url,
			s.title,
			s.published_at,
			s.enrichment_status,
			s.enrichment_error,
			s.enriched_at,
			s.event_id,
			e.title as event_title,
			e.status as event_status
		FROM sources s
		LEFT JOIN events e ON s.event_id = e.id
		WHERE s.enrichment_status != 'pending'
		ORDER BY s.enriched_at DESC NULLS LAST, s.created_at DESC
		LIMIT $1
	`

	rows, err := h.db.QueryContext(ctx, query, limit)
	if err != nil {
		h.logger.Error("Failed to query recent enrichments", "error", err)
		http.Error(w, "Failed to query recent enrichments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type EnrichmentInfo struct {
		SourceID         string  `json:"source_id"`
		SourceType       string  `json:"source_type"`
		SourceURL        string  `json:"source_url"`
		SourceTitle      string  `json:"source_title"`
		PublishedAt      string  `json:"published_at"`
		EnrichmentStatus string  `json:"enrichment_status"`
		EnrichmentError  *string `json:"enrichment_error,omitempty"`
		EnrichedAt       *string `json:"enriched_at,omitempty"`
		EventID          *string `json:"event_id,omitempty"`
		EventTitle       *string `json:"event_title,omitempty"`
		EventStatus      *string `json:"event_status,omitempty"`
	}

	enrichments := []EnrichmentInfo{}

	for rows.Next() {
		var info EnrichmentInfo
		var enrichmentError, enrichedAt, eventID, eventTitle, eventStatus sql.NullString

		err := rows.Scan(
			&info.SourceID,
			&info.SourceType,
			&info.SourceURL,
			&info.SourceTitle,
			&info.PublishedAt,
			&info.EnrichmentStatus,
			&enrichmentError,
			&enrichedAt,
			&eventID,
			&eventTitle,
			&eventStatus,
		)
		if err != nil {
			h.logger.Error("Failed to scan enrichment row", "error", err)
			continue
		}

		if enrichmentError.Valid {
			info.EnrichmentError = &enrichmentError.String
		}
		if enrichedAt.Valid {
			info.EnrichedAt = &enrichedAt.String
		}
		if eventID.Valid {
			info.EventID = &eventID.String
		}
		if eventTitle.Valid {
			info.EventTitle = &eventTitle.String
		}
		if eventStatus.Valid {
			info.EventStatus = &eventStatus.String
		}

		enrichments = append(enrichments, info)
	}

	if err := rows.Err(); err != nil {
		h.logger.Error("Error iterating enrichment rows", "error", err)
		http.Error(w, "Error reading enrichment data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enrichments": enrichments,
		"count":       len(enrichments),
	})
}
