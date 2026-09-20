package tracking

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClassifyFlagsNoisyPoints(t *testing.T) {
	service := &Service{now: func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }}
	accuracy := float32(150)
	point := Point{
		SampleID: uuid.New(), CapturedAt: service.now(), Latitude: 26.9, Longitude: 75.8, AccuracyM: &accuracy,
	}

	got, err := service.classify(point)
	if err != nil {
		t.Fatalf("classify() error = %v", err)
	}
	if got.IsAccepted || got.RejectionReason == nil || *got.RejectionReason != "poor_accuracy" {
		t.Fatalf("classify() = %#v", got)
	}
}

func TestClassifyRejectsInvalidCoordinates(t *testing.T) {
	service := &Service{now: time.Now}
	_, err := service.classify(Point{SampleID: uuid.New(), CapturedAt: time.Now(), Latitude: 91})
	if !errors.Is(err, ErrInvalidBatch) {
		t.Fatalf("classify() error = %v, want ErrInvalidBatch", err)
	}
}
