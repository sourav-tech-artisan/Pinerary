package nearby

import (
	"testing"

	"github.com/sourav-tech-artisan/Pinerary/backend/internal/routing"
)

func TestValidMode(t *testing.T) {
	if !validMode(routing.ModeMotorcycle) || !validMode(routing.ModeCar) || !validMode(routing.ModeWalking) {
		t.Fatal("supported route mode rejected")
	}
	if validMode("bicycle") {
		t.Fatal("unsupported route mode accepted")
	}
}
