package pkg

import (
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
)

// AccessTicket is a reserved slot in the matching engine's command queue. It sends at
// most one command; once sent, the slot belongs to that command and the engine frees it.
// Defer ReleaseWriteAccess right after obtaining the ticket: it gives the slot back if
// nothing was sent and does nothing otherwise.
type AccessTicket interface {
	SendPlaceOrder(commandID uuid.UUID, order entity.Order) error
	SendCancelOrder(commandID, orderID uuid.UUID) error
	ReleaseWriteAccess() error
}
