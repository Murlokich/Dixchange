package engine

import (
	"context"
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	"github.com/dixchange/Monolith/pkg"
)

type OrderBook interface {
	AddOrder(order *orderbook.Order) ([]orderbook.Trade, error)
}

type BookWriter interface {
	RetrieveOrder(ctx context.Context) (orderbook.Order, error)
}

type TradeWriter interface {
	InsertTrades(ctx context.Context, trades []entity.Trade) error
}

type MatchingEngine struct {
	book        OrderBook
	bookWriter  BookWriter
	tradeWriter TradeWriter
	logger      pkg.Logger
}

func (m *MatchingEngine) Run(ctx context.Context) {
	for {
		order, err := m.bookWriter.RetrieveOrder(ctx)
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
		entityTrades := orderbookToEntityTrades(trades)
		if err = m.tradeWriter.InsertTrades(ctx, entityTrades); err != nil {
			if errors.Is(err, context.Canceled) {
				// TODO: graceful shutdown
				return
			}
			m.logger.Errorf("tradeWriter Insert Trades: %v", err)
		}
	}
}

func NewMatchingEngine(
	book OrderBook,
	bookWriter BookWriter,
	tradeWriter TradeWriter,
	logger pkg.Logger,
) *MatchingEngine {
	return &MatchingEngine{
		book:        book,
		bookWriter:  bookWriter,
		tradeWriter: tradeWriter,
		logger:      logger,
	}
}

func orderbookToEntityTrades(trades []orderbook.Trade) []entity.Trade {
	entityTrades := make([]entity.Trade, len(trades))
	for i := range trades {
		entityTrades[i] = entity.Trade{
			AskOrderID: trades[i].AskOrderID,
			BidOrderID: trades[i].BidOrderID,
			Price:      trades[i].Price,
			Quantity:   trades[i].Quantity,
		}
	}
	return entityTrades
}
