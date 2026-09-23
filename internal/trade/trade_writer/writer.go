package tradewriter

import (
	"context"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
)

type TradeWriter struct {
	queue chan []entity.Trade
}

func (w *TradeWriter) InsertTrades(ctx context.Context, trades []entity.Trade) error {
	if len(trades) == 0 {
		return nil
	}
	select {
	case w.queue <- trades:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *TradeWriter) RetrieveTrades(ctx context.Context) ([]entity.Trade, error) {
	select {
	case trades := <-w.queue:
		return trades, nil
	case <-ctx.Done():
		return []entity.Trade{}, ctx.Err()
	}
}

func NewTradeWriter(cfg *config.TradeWriterConfig) *TradeWriter {
	return &TradeWriter{
		queue: make(chan []entity.Trade, cfg.BufferSize),
	}
}
