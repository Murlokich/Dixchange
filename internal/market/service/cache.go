package service

import (
	"context"
	"slices"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/pkg/errors"
)

const (
	usdCode        = "USD"
	kostyaCoinCode = "KCN"
)

type Repository interface {
	GetPairs(ctx context.Context) ([]entity.Pair, error)
}

type DBCache struct {
	market         map[int]entity.Pair
	symbols        map[int]entity.Symbol
	defaultSymbols []entity.Symbol

	r Repository
}

func NewDBCache(ctx context.Context, r Repository) (*DBCache, error) {
	c := DBCache{r: r}
	if err := c.Load(ctx); err != nil {
		return nil, errors.Wrap(err, "db cache load")
	}
	return &c, nil
}

func (c *DBCache) Load(ctx context.Context) error {
	pairs, err := c.r.GetPairs(ctx)
	if err != nil {
		return errors.Wrap(err, "repository GetPairs")
	}
	c.market = make(map[int]entity.Pair)
	c.symbols = make(map[int]entity.Symbol)
	// assemble market pairs map and symbols based on market
	for i := range pairs {
		c.market[pairs[i].ID] = pairs[i]
		c.symbols[pairs[i].Base.ID] = pairs[i].Base
		c.symbols[pairs[i].Quote.ID] = pairs[i].Quote
	}

	defaultCodes := []string{
		usdCode,
		kostyaCoinCode,
	}

	// extract default symbols out of all
	for _, symbol := range c.symbols {
		if slices.Contains(defaultCodes, symbol.Code) {
			c.defaultSymbols = append(c.defaultSymbols, symbol)
		}
	}

	return nil
}

func (c *DBCache) GetPairs() map[int]entity.Pair {
	return c.market
}

func (c *DBCache) GetDefaultSymbols() []entity.Symbol {
	return c.defaultSymbols
}

func (c *DBCache) GetPair(id int) (entity.Pair, bool) {
	p, ok := c.market[id]
	return p, ok
}

func (c *DBCache) GetSymbol(id int) (entity.Symbol, bool) {
	s, ok := c.symbols[id]
	return s, ok
}
