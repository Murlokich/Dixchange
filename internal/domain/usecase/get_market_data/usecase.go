package getmarketdata

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/dixchange/Monolith/internal/domain/entity"
)

type MarketRepository interface {
	GetPairs() map[int]entity.Pair
}

type UseCase struct {
	marketRepo MarketRepository
}

func (u *UseCase) Execute(_ context.Context) (Output, error) {
	pairs := slices.SortedFunc(maps.Values(u.marketRepo.GetPairs()), func(a, b entity.Pair) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return Output{Pairs: pairs}, nil
}

func NewUseCase(marketRepo MarketRepository) *UseCase {
	return &UseCase{marketRepo: marketRepo}
}
