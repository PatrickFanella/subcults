package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/onnwee/subcults/internal/middleware"
)

// cspReport is the top-level JSON wrapper that browsers send.
type cspReport struct {
	Body cspReportBody `json:"csp-report"`
}

// cspReportBody contains the fields of a CSP violation report.
type cspReportBody struct {
	DocumentURI        string `json:"document-uri"`
	Referrer           string `json:"referrer"`
	BlockedURI         string `json:"blocked-uri"`
	ViolatedDirective  string `json:"violated-directive"`
	EffectiveDirective string `json:"effective-directive"`
	OriginalPolicy     string `json:"original-policy"`
	StatusCode         int    `json:"status-code"`
	SourceFile         string `json:"source-file"`
	LineNumber         int    `json:"line-number"`
	ColumnNumber       int    `json:"column-number"`
}

// CSPReportHandler returns an http.HandlerFunc that accepts Content-Security-Policy
// violation reports and logs them via structured logging.
func CSPReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
			WriteError(w, ctx, http.StatusMethodNotAllowed, ErrCodeBadRequest, "Method not allowed")
			return
		}

		// Limit the body to 10KB to prevent abuse
		body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024))
		if err != nil {
			ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Failed to read request body")
			return
		}

		var report cspReport
		if err := json.Unmarshal(body, &report); err != nil {
			ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CSP report format")
			return
		}

		v := report.Body
		slog.WarnContext(ctx, "csp_violation",
			"document_uri", v.DocumentURI,
			"blocked_uri", v.BlockedURI,
			"violated_directive", v.ViolatedDirective,
			"effective_directive", v.EffectiveDirective,
			"source_file", v.SourceFile,
			"line_number", v.LineNumber,
			"column_number", v.ColumnNumber,
			"status_code", v.StatusCode,
		)

		w.WriteHeader(http.StatusNoContent)
	}
}
