package journeys

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindTrip   Kind = "trip"
	KindOuting Kind = "outing"
)

type Status string

const (
	StatusActive  Status = "active"
	StatusEnded   Status = "ended"
	StatusExpired Status = "expired"
)

type Journey struct {
	ID        uuid.UUID
	OwnerID   uuid.UUID
	Kind      Kind
	Label     string
	Status    Status
	StartedAt time.Time
	EndedAt   *time.Time
	Version   int32
}

var (
	ErrActiveJourneyExists = errors.New("an active journey already exists")
	ErrConflict            = errors.New("journey version conflict")
	ErrInvalidKind         = errors.New("journey kind must be trip or outing")
	ErrInvalidLabel        = errors.New("journey label must contain 1 to 160 characters")
	ErrInvalidTransition   = errors.New("journey transition is not allowed")
	ErrNotFound            = errors.New("journey not found")
)

func ValidateNew(kind Kind, label string) error {
	if kind != KindTrip && kind != KindOuting {
		return ErrInvalidKind
	}
	if length := len([]rune(strings.TrimSpace(label))); length < 1 || length > 160 {
		return ErrInvalidLabel
	}
	return nil
}

func (j Journey) CanComplete() bool {
	return j.Status == StatusActive
}

func (j Journey) CanConvertToTrip() bool {
	return j.Kind == KindOuting && j.Status == StatusActive
}
