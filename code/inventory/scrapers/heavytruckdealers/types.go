package heavytruckdealers

// FacetWPRequest represents the POST request payload
type FacetWPRequest struct {
	Action string      `json:"action"`
	Data   RequestData `json:"data"`
}

// RequestData represents the data field in the request
type RequestData struct {
	Paged        int                    `json:"paged"`
	Facets       map[string]interface{} `json:"facets"`
	FrozenFacets map[string]interface{} `json:"frozen_facets"`
	HTTPParams   HTTPParams             `json:"http_params"`
}

// HTTPParams represents the http_params field
type HTTPParams struct {
	Get []string `json:"get"`
	URI string   `json:"uri"`
}

// FacetWPResponse represents the JSON response
type FacetWPResponse struct {
	Facets   FacetsResponse `json:"facets"`
	Template string         `json:"template"` // HTML listings
}

// FacetsResponse represents the facets field in the response
type FacetsResponse struct {
	Pagination string `json:"pagination"` // HTML pagination
}
