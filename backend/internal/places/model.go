package places

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Place struct {
	ID              uuid.UUID
	ClientRequestID uuid.UUID
	Name            string
	Notes           string
	Coordinates     Coordinates
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Stop struct {
	ID              uuid.UUID
	JourneyID       uuid.UUID
	PlaceID         uuid.UUID
	ClientRequestID uuid.UUID
	SequenceNumber  int64
	CapturedAt      time.Time
	DisplayName     string
	Note            string
	Coordinates     Coordinates
}

var (
	ErrInvalidCoordinates = errors.New("coordinates are outside valid latitude or longitude bounds")
	ErrInvalidName        = errors.New("name must contain 1 to 200 characters")
	ErrJourneyNotActive   = errors.New("journey is not active")
	ErrNotFound           = errors.New("place or stop not found")
)

func Validate(name string, coordinates Coordinates) error {
	length := len([]rune(strings.TrimSpace(name)))
	if length < 1 || length > 200 {
		return ErrInvalidName
	}
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 ||
		coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}
