package entity

import "github.com/google/uuid"

type BalanceRecord struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	SymbolID int
}
