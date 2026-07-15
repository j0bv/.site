package selectrucks

// SearchRequest represents the POST request payload
type SearchRequest struct {
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
	Filters  map[string]interface{} `json:"filters,omitempty"`
}

// SearchResponse represents the JSON response
type SearchResponse struct {
	Items []Truck `json:"items"`
	Total int     `json:"total"`
}

// Truck represents a single truck listing from the API
type Truck struct {
	DealerCountryAbbreviation string `json:"dealerCountryAbbreviation"`
	DealerId                  int    `json:"dealerId"`
	DealerName                string `json:"dealerName"`
	DealerPhone               string `json:"dealerPhone"`
	ImageFileName             string `json:"imageFileName"`
	Manufacturer              string `json:"manufacturer"`
	Mileage                   string `json:"mileage"`
	Model                     string `json:"model"`
	Price                     string `json:"price"`
	StockNumber               string `json:"stockNumber"`
	TruckId                   int    `json:"truckId"`
	Year                      int    `json:"year"`
}
