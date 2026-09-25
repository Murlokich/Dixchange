package entity

import "github.com/google/uuid"

type BalanceRecord struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	SymbolID int
}

// Balance is the state of a user balance of one symbol.
type Balance struct {
	SymbolID int
	// Main is the amount available to the user, in symbol minor units.
	Main int64
	// Reserved is the amount locked by open orders, in symbol minor units.
	Reserved int64
}
