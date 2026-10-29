package getmarketdata

import "github.com/Murlokich/Dixchange/internal/domain/entity"

type Output struct {
	// Pairs are ordered by ID so the response is stable across calls.
	Pairs []entity.Pair
}
