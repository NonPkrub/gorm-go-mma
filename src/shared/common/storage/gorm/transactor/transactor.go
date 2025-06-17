// Ref: https://github.com/Thiht/transactor/blob/main/sqlx/transactor.go
package transactor

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type PostCommitHook func(ctx context.Context) error

type Transactor interface {
	WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context, registerPostCommitHook func(PostCommitHook)) error) error
}

type (
	gormDBGetter               func(context.Context) *gorm.DB
	nestedTransactionsStrategy func(tx *gorm.DB) (*gorm.DB, func() error)
)

type gormTransactor struct {
	gormDBGetter
	nestedTransactionsStrategy
}

type Option func(*gormTransactor)

type (
	transactorKey struct{}
	// DBTXContext is used to get the current DB handler from the context.
	// It returns the current transaction if there is one, otherwise it will return the original DB.
)

func txToContext(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, transactorKey{}, tx)
}

func txFromContext(ctx context.Context) *gorm.DB {
	tx, _ := ctx.Value(transactorKey{}).(*gorm.DB)
	return tx
}

func New(db *gorm.DB, opts ...Option) (Transactor, func(context.Context) *gorm.DB) {
	t := &gormTransactor{
		gormDBGetter: func(ctx context.Context) *gorm.DB {
			if tx := txFromContext(ctx); tx != nil {
				return tx
			}
			return db
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	dbGetter := func(ctx context.Context) *gorm.DB {
		if tx := txFromContext(ctx); tx != nil {
			return tx
		}
		return db
	}

	return t, dbGetter
}

func WithNestedTransactionStrategy(strategy nestedTransactionsStrategy) Option {
	return func(t *gormTransactor) {
		t.nestedTransactionsStrategy = strategy
	}
}

func (t *gormTransactor) WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context, registerPostCommitHook func(PostCommitHook)) error) error {
	db := t.gormDBGetter(ctx)
	var hooks []PostCommitHook

	registerPostCommitHook := func(hook PostCommitHook) {
		hooks = append(hooks, hook)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Apply nested strategy if exists
		if t.nestedTransactionsStrategy != nil {
			nestedTx, commit := t.nestedTransactionsStrategy(tx)
			ctxWithTx := txToContext(ctx, nestedTx)

			if err := txFunc(ctxWithTx, registerPostCommitHook); err != nil {
				return err
			}
			if err := commit(); err != nil {
				return err
			}
		} else {
			ctxWithTx := txToContext(ctx, tx)
			if err := txFunc(ctxWithTx, registerPostCommitHook); err != nil {
				return err
			}
		}

		// run post-commit hooks
		go func() {
			for _, hook := range hooks {
				func(h PostCommitHook) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("post-commit hook panic: %v\n", r)
						}
					}()
					if err := h(ctx); err != nil {
						fmt.Printf("post-commit hook error: %v\n", err)
					}
				}(hook)
			}
		}()

		return nil
	})
}

func IsWithinTransaction(ctx context.Context) bool {
	return txFromContext(ctx) != nil
}

// NestedTransactionsSavepoints is a nested transactions implementation using savepoints.
// It's compatible with PostgreSQL, MySQL, MariaDB, and SQLite.
var NestedTransactionsSavepoints nestedTransactionsStrategy = func(tx *gorm.DB) (*gorm.DB, func() error) {
	savepoint := "sp1" // generate unique name if needed
	if err := tx.Exec("SAVEPOINT " + savepoint).Error; err != nil {
		return tx, func() error { return err }
	}

	return tx, func() error {
		return tx.Exec("RELEASE SAVEPOINT " + savepoint).Error
	}
}
