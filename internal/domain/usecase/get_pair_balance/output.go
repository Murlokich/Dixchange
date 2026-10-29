package getpairbalance

import "github.com/Murlokich/Dixchange/internal/domain/entity"

type Output struct {
	// Pair is returned alongside the balances so amounts can be formatted at the symbol precisions.
	Pair  entity.Pair
	Base  entity.Balance
	Quote entity.Balance
}
