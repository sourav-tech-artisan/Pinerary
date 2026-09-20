package httpapi

import "testing"

func TestValidTransportMode(t *testing.T) {
	tests := map[string]bool{
		"motorcycle": true,
		"car":        true,
		"walking":    true,
		"bicycle":    false,
		"":           false,
	}

	for mode, want := range tests {
		if got := validTransportMode(mode); got != want {
			t.Errorf("validTransportMode(%q) = %v, want %v", mode, got, want)
		}
	}
}
