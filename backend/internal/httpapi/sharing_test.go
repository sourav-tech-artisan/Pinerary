package httpapi

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sourav-tech-artisan/Pinerary/backend/internal/sharing"
)

func TestSharePageTemplateEscapesContent(t *testing.T) {
	snapshot := sharing.PublicSnapshot{
		Title:     `<script>alert("x")</script>`,
		StartedAt: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Items: []sharing.PublicItem{{
			StopID: uuid.New(), DisplayName: "Nahargarh Fort", Latitude: 26.9, Longitude: 75.8,
			CapturedAt: time.Date(2026, 9, 21, 11, 0, 0, 0, time.UTC),
		}},
	}
	var output bytes.Buffer
	if err := sharePageTemplate.Execute(&output, snapshot); err != nil {
		t.Fatalf("execute share template: %v", err)
	}
	if strings.Contains(output.String(), "<script>alert") {
		t.Fatal("share page did not escape user content")
	}
	if !strings.Contains(output.String(), "Nahargarh Fort") {
		t.Fatal("share page omitted the stop")
	}
}
