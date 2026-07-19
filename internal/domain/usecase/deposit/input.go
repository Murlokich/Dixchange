package deposit

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Input struct {
	UserID    uuid.UUID
	USDAmount decimal.Decimal
}
