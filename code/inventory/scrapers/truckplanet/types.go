package truckplanet

// SearchResponse represents the JSON response from the search endpoint
type SearchResponse struct {
	Hitprm    string `json:"hitprm"`
	PageTitle string `json:"pageTitle"`
	Divs      []Div  `json:"divs"`
}

// Div represents a single div element in the JSON response
type Div struct {
	ID    string `json:"id"`
	Value string `json:"value"` // HTML content
}

