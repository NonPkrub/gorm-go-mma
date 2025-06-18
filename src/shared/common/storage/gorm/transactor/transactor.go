// Ref: https://github.com/Thiht/transactor/blob/main/sqlx/transactor.go
package transactor

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

type PostCommitHook func(ctx context.Context) error

type Transactor interface {
	WithinTransaction(ctx context.Context, txFunc func(ctxWithTx context.Context, registerPostCommitHook func(PostCommitHook)) error) error
}

type (
	gormDBGetter               func(context.Context) *gorm.DB
	nestedTransactionsStrategy func(tx *gorm.DB) (*gorm.DB, func(commit bool) error)
)

type gormTransactor struct {
	gormDBGetter
	nestedTransactionsStrategy
}

type Option func(*gormTransactor)

type (
	transactorKey struct{}
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
				_ = commit(false)
				return err
			}
			if err := commit(true); err != nil {
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

// generateSavepointName สร้างชื่อ savepoint ไม่ซ้ำ
var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateSavepointName() string {
	return fmt.Sprintf("sp_%d", seededRand.Intn(1_000_000))
}

// nestedTransactionsStrategy ที่รองรับ commit และ rollback
var NestedTransactionsSavepoints nestedTransactionsStrategy = func(tx *gorm.DB) (*gorm.DB, func(commit bool) error) {
	savepoint := generateSavepointName()

	// สร้าง SAVEPOINT
	if err := tx.Exec("SAVEPOINT " + savepoint).Error; err != nil {
		return tx, func(_ bool) error { return err }
	}

	// ฟังก์ชัน commit/rollback
	return tx, func(commit bool) error {
		if commit {
			return tx.Exec("RELEASE SAVEPOINT " + savepoint).Error
		}
		return tx.Exec("ROLLBACK TO SAVEPOINT " + savepoint).Error
	}
}
