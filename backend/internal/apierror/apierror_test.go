package apierror

import (
	"errors"
	"net/http"
	"testing"
)

func TestPublicKeepsAPIError(t *testing.T) {
	want := Wrap(errors.New("database unavailable"), http.StatusServiceUnavailable, "unavailable", "try again later")
	got := Public(want)

	if got != want {
		t.Fatalf("Public() = %#v, want %#v", got, want)
	}
	if !errors.Is(got, want.Cause) {
		t.Fatal("wrapped cause was not preserved")
	}
}

func TestPublicHidesUnknownError(t *testing.T) {
	got := Public(errors.New("secret internal detail"))

	if got.Status != http.StatusInternalServerError || got.Code != "internal_error" {
		t.Fatalf("Public() = %#v", got)
	}
}
