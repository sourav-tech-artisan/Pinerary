package placesearch

import "context"

const ProviderGeoapify = "geoapify"

type BoundingBox struct {
	South float64 `json:"south"`
	West  float64 `json:"west"`
	North float64 `json:"north"`
	East  float64 `json:"east"`
}

type Suggestion struct {
	ResultID    string       `json:"result_id"`
	Name        string       `json:"name"`
	Address     string       `json:"address"`
	Latitude    float64      `json:"latitude"`
	Longitude   float64      `json:"longitude"`
	Category    string       `json:"category,omitempty"`
	Type        string       `json:"type,omitempty"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}

type SuggestInput struct {
	Query          string
	Limit          int32
	AcceptLanguage string
}

type Provider interface {
	Name() string
	Suggest(context.Context, SuggestInput) ([]Suggestion, error)
}
