package pkg

import (
	"context"
	goerrors "errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	DBTX
}

type TXManager struct {
	db Beginner
}

type txKey struct{}

func (tm *TXManager) WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := tm.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "TXManager begin tx")
	}
	ctx = context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("recovered from panic: %v", p)
		}

		if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				err = goerrors.Join(err, errRollback)
			}

			return
		}

		if errCommit := tx.Commit(ctx); errCommit != nil {
			err = goerrors.Join(err, errCommit)
		}
	}()

	if err = fn(ctx); err != nil {
		return errors.Wrap(err, "TXManager WithTX fn")
	}

	return nil
}

type TXGetter func(ctx context.Context, db Beginner) DBTX

func GetDB(ctx context.Context, db Beginner) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db
}
