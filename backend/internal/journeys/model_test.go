package journeys

import "testing"

func TestValidateNew(t *testing.T) {
	tests := []struct {
		name    string
		kind    Kind
		label   string
		wantErr bool
	}{
		{name: "trip", kind: KindTrip, label: "Goa", wantErr: false},
		{name: "outing", kind: KindOuting, label: "Evening ride", wantErr: false},
		{name: "unknown kind", kind: "holiday", label: "Goa", wantErr: true},
		{name: "blank label", kind: KindTrip, label: "  ", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateNew(test.kind, test.label); (got != nil) != test.wantErr {
				t.Fatalf("ValidateNew() error = %v, wantErr %v", got, test.wantErr)
			}
		})
	}
}

func TestJourneyTransitions(t *testing.T) {
	outing := Journey{Kind: KindOuting, Status: StatusActive}
	if !outing.CanComplete() || !outing.CanConvertToTrip() {
		t.Fatal("active outing should be completable and convertible")
	}

	ended := Journey{Kind: KindOuting, Status: StatusEnded}
	if ended.CanComplete() || ended.CanConvertToTrip() {
		t.Fatal("ended outing must be immutable")
	}
}
