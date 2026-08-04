package sql

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/pkg/errors"
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) GetPairs(ctx context.Context) ([]entity.Pair, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.GetPairs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "queries GetPairs")
	}
	return rowsToPairs(rows), nil
}

func rowsToPairs(rows []GetPairsRow) []entity.Pair {
	pairs := make([]entity.Pair, len(rows))
	for i := range rows {
		pairs[i] = entity.Pair{
			ID: int(rows[i].PairID),
			Base: entity.Symbol{
				ID:   int(rows[i].BaseCurrencyID),
				Code: rows[i].BaseCurrencyCode,
			},
			Quote: entity.Symbol{
				ID:   int(rows[i].QuoteCurrencyID),
				Code: rows[i].QuoteCurrencyCode,
			},
		}
	}
	return pairs
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
