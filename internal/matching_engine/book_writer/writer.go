package bookwriter

import (
	"context"
	"time"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
)

type BookWriter struct {
	writeTimeout time.Duration
	queue        chan orderbook.Order
}

func (w *BookWriter) InsertOrder(ctx context.Context, entityOrder entity.Order) error {
	order := orderbook.NewOrderFromEntity(entityOrder)
	ctx, cancel := context.WithTimeout(ctx, w.writeTimeout)
	defer cancel()

	select {
	case w.queue <- order:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *BookWriter) RetrieveOrder(ctx context.Context) (orderbook.Order, error) {
	select {
	case order := <-w.queue:
		return order, nil
	case <-ctx.Done():
		return orderbook.Order{}, ctx.Err()
	}
}

func NewBookWriter(cfg *config.BookWriterConfig) *BookWriter {
	return &BookWriter{
		queue:        make(chan orderbook.Order, cfg.BufferSize),
		writeTimeout: cfg.WriteTimeout,
	}
}
