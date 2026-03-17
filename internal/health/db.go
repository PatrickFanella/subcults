package health

import (
"context"
"database/sql"
)

// DBChecker implements health checking for SQL databases.
type DBChecker struct {
db *sql.DB
}

// NewDBChecker creates a new database health checker.
func NewDBChecker(db *sql.DB) *DBChecker {
return &DBChecker{
db: db,
}
}

// HealthCheck performs a health check on the database by executing a simple query.
func (d *DBChecker) HealthCheck(ctx context.Context) error {
return d.db.PingContext(ctx)
}
