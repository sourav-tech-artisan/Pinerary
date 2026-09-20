package tracking

import (
	"testing"
	"time"
)

func TestCleanFiltersJumpAndSplitsOnGap(t *testing.T) {
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	samples := []Sample{
		{DatabaseID: 1, CapturedAt: start, Latitude: 26.9, Longitude: 75.8, AccuracyM: 5, Accepted: true},
		{DatabaseID: 2, CapturedAt: start.Add(time.Minute), Latitude: 26.91, Longitude: 75.81, AccuracyM: 5, Accepted: true},
		{DatabaseID: 3, CapturedAt: start.Add(61 * time.Second), Latitude: 28.0, Longitude: 77.0, AccuracyM: 5, Accepted: true},
		{DatabaseID: 4, CapturedAt: start.Add(10 * time.Minute), Latitude: 26.92, Longitude: 75.82, AccuracyM: 5, Accepted: true},
		{DatabaseID: 5, CapturedAt: start.Add(11 * time.Minute), Latitude: 26.93, Longitude: 75.83, AccuracyM: 5, Accepted: true},
	}

	segments, decisions := Clean(samples)
	if len(segments) != 2 {
		t.Fatalf("len(segments) = %d, want 2", len(segments))
	}
	if len(decisions) != 1 || decisions[0].DatabaseID != 3 || decisions[0].Reason != "impossible_jump" {
		t.Fatalf("decisions = %#v", decisions)
	}
}

func TestHaversineMeters(t *testing.T) {
	distance := haversineMeters(0, 0, 0, 1)
	if distance < 111000 || distance > 112000 {
		t.Fatalf("distance = %.2f", distance)
	}
}
