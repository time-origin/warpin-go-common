package database

import (
	"context"
	"gorm.io/gorm"
)

// WithTx executes a function within a database transaction.
// The `fn` function receives a transaction-bound *gorm.DB object.
// If `fn` returns an error, the transaction is automatically rolled back. Otherwise, it's committed.
func WithTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}
