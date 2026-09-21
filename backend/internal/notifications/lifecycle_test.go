package notifications

import (
	"testing"

	"github.com/google/uuid"
)

func TestDecodeOutingJob(t *testing.T) {
	id := uuid.New()
	job, err := decodeOutingJob([]byte(`{"journey_id":"` + id.String() + `"}`))
	if err != nil {
		t.Fatalf("decodeOutingJob() error = %v", err)
	}
	if job.JourneyID != id {
		t.Fatalf("journey ID = %s", job.JourneyID)
	}
	if _, err := decodeOutingJob([]byte(`{}`)); err == nil {
		t.Fatal("expected missing journey ID to fail")
	}
}
