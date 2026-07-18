package deposit

import (
	"github.com/google/uuid"
	decimal "github.com/jackc/pgx-shopspring-decimal"
)

type Input struct {
	UserID    uuid.UUID
	USDAmount decimal.Decimal
}
