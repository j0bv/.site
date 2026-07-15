package trucktractortrailer

// SearchResponse represents the JSON response from the /filter endpoint
type SearchResponse struct {
	TractorTypes      []map[string]string              `json:"tractorTypes"`
	Brands            []map[string]string              `json:"brands"`
	ListingsCount     string                           `json:"listingsCount"`
	Pager             string                           `json:"pager"` // HTML pagination
	Listings          string                           `json:"listings"` // HTML listings
	PriceRange        PriceRange                       `json:"priceRange"`
	TractorTypeObjects map[string]interface{}          `json:"tractorTypeObjects"`
	BrandObjects      map[string]interface{}           `json:"brandObjects"`
}

// PriceRange represents the min/max price range in the response
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

