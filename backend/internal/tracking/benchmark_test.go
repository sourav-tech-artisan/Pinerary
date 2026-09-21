package tracking

import (
	"testing"
	"time"
)

func BenchmarkCleanTenThousandPoints(b *testing.B) {
	start := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	samples := make([]Sample, 10_000)
	for index := range samples {
		samples[index] = Sample{
			DatabaseID: int64(index + 1), CapturedAt: start.Add(time.Duration(index) * 5 * time.Second),
			Latitude: 26.9 + float64(index)*0.00001, Longitude: 75.8 + float64(index)*0.00001,
			AccuracyM: 5, Accepted: true,
		}
	}

	b.ResetTimer()
	for range b.N {
		Clean(samples)
	}
}
