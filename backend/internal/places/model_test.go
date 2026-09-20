package places

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		coordinates Coordinates
		wantErr     bool
	}{
		{name: "Nahargarh Fort", coordinates: Coordinates{Latitude: 26.9373, Longitude: 75.8155}},
		{name: "", coordinates: Coordinates{}, wantErr: true},
		{name: "Invalid latitude", coordinates: Coordinates{Latitude: 91}, wantErr: true},
		{name: "Invalid longitude", coordinates: Coordinates{Longitude: -181}, wantErr: true},
	}

	for _, test := range tests {
		if got := Validate(test.name, test.coordinates); (got != nil) != test.wantErr {
			t.Errorf("Validate(%q, %#v) error = %v, wantErr %v", test.name, test.coordinates, got, test.wantErr)
		}
	}
}
