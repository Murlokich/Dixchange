package engine

import (
	"context"
	"errors"

	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	"github.com/dixchange/Monolith/pkg"
)

type OrderBook interface {
	AddOrder(order *orderbook.Order) ([]orderbook.Trade, error)
}

type BookWriter interface {
	RetrieveOrder(ctx context.Context) (orderbook.Order, error)
}

type MatchingEngine struct {
	book   OrderBook
	writer BookWriter
	logger pkg.Logger
}

func (m *MatchingEngine) Run(ctx context.Context) {
	for {
		order, err := m.writer.RetrieveOrder(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			m.logger.Errorf("bookWriter Retrieve order: %v", err)
			continue
		}
		loggerWithOrderID := m.logger.WithField("order_id", order.ID)

		trades, err := m.book.AddOrder(&order)
		if err != nil {
			loggerWithOrderID.Errorf("bookWriter Retrieve order: %v", err)
			continue
		}
		// TODO: write trades
		m.logger.Infof("trades: %v", trades)
	}
}

func NewMatchingEngine(book OrderBook, writer BookWriter, logger pkg.Logger) *MatchingEngine {
	return &MatchingEngine{
		book:   book,
		writer: writer,
		logger: logger,
	}
}
