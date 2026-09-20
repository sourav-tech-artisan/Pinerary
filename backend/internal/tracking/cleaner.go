package tracking

import (
	"math"
	"time"
)

const (
	earthRadiusM       = 6371000.0
	maxDerivedSpeedMPS = 70.0
	segmentGap         = 5 * time.Minute
)

type Sample struct {
	DatabaseID int64
	CapturedAt time.Time
	Latitude   float64
	Longitude  float64
	AccuracyM  float64
	Accepted   bool
}

type QualityDecision struct {
	DatabaseID int64
	Accepted   bool
	Reason     string
}

type Segment struct {
	Points    []Sample
	DistanceM float64
	StartedAt time.Time
	EndedAt   time.Time
}

func Clean(samples []Sample) ([]Segment, []QualityDecision) {
	segments := make([]Segment, 0)
	decisions := make([]QualityDecision, 0)
	current := Segment{}
	var previous *Sample

	flush := func() {
		if len(current.Points) >= 2 {
			current.StartedAt = current.Points[0].CapturedAt
			current.EndedAt = current.Points[len(current.Points)-1].CapturedAt
			segments = append(segments, current)
		}
		current = Segment{}
		previous = nil
	}

	for index := range samples {
		sample := samples[index]
		if !sample.Accepted {
			continue
		}
		if previous == nil {
			current.Points = append(current.Points, sample)
			previous = &sample
			continue
		}

		delta := sample.CapturedAt.Sub(previous.CapturedAt)
		if delta > segmentGap {
			flush()
			current.Points = append(current.Points, sample)
			previous = &sample
			continue
		}
		if delta <= 0 {
			decisions = append(decisions, QualityDecision{DatabaseID: sample.DatabaseID, Reason: "non_monotonic_time"})
			continue
		}

		distance := haversineMeters(previous.Latitude, previous.Longitude, sample.Latitude, sample.Longitude)
		if distance/delta.Seconds() > maxDerivedSpeedMPS {
			decisions = append(decisions, QualityDecision{DatabaseID: sample.DatabaseID, Reason: "impossible_jump"})
			continue
		}
		jitterRadius := math.Max(5, (previous.AccuracyM+sample.AccuracyM)/4)
		if delta <= 30*time.Second && distance < jitterRadius {
			decisions = append(decisions, QualityDecision{DatabaseID: sample.DatabaseID, Reason: "stationary_jitter"})
			continue
		}

		current.DistanceM += distance
		current.Points = append(current.Points, sample)
		previous = &sample
	}
	flush()

	return segments, decisions
}

func haversineMeters(latitudeA, longitudeA, latitudeB, longitudeB float64) float64 {
	latA := latitudeA * math.Pi / 180
	latB := latitudeB * math.Pi / 180
	deltaLat := (latitudeB - latitudeA) * math.Pi / 180
	deltaLon := (longitudeB - longitudeA) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latA)*math.Cos(latB)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
