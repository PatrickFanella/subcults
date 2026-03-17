package health

import (
"testing"
)

// TestDBChecker_Creation tests that the DB checker is created correctly.
func TestDBChecker_Creation(t *testing.T) {
// Note: We cannot create a valid *sql.DB without a real connection.
// This test only verifies the constructor doesn't panic with nil.
// Integration tests should verify actual health checking behavior.

checker := NewDBChecker(nil)
if checker == nil {
t.Fatal("expected checker to be non-nil")
}

if checker.db != nil {
t.Error("expected checker db to be nil when nil is passed")
}
}
