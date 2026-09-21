package routing

import "testing"

func TestValhallaCosting(t *testing.T) {
	if got := valhallaCosting(ModeMotorcycle); got != "motorcycle" {
		t.Fatalf("motorcycle costing = %q", got)
	}
	if got := valhallaCosting(ModeCar); got != "auto" {
		t.Fatalf("car costing = %q", got)
	}
	if got := valhallaCosting(ModeWalking); got != "pedestrian" {
		t.Fatalf("walking costing = %q", got)
	}
}
