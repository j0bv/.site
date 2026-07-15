package arrowtruck

// SearchRequest represents the POST request payload
type SearchRequest struct {
	PageIndex    int                    `json:"pageIndex"`
	PageSize    int                    `json:"pageSize"`
	SortCriteria []SortCriterion       `json:"sortCriteria,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// SortCriterion represents a sort criterion
type SortCriterion struct {
	Key       string `json:"key"`
	Ascending bool   `json:"ascending"`
}

// SearchResponse represents the JSON response
type SearchResponse struct {
	AppliedSortCriteria []SortCriterion `json:"appliedSortCriteria,omitempty"`
	AvailableSortKeys   []string        `json:"availableSortKeys,omitempty"`
	PageIndex            int             `json:"pageIndex"`
	PageSize             int             `json:"pageSize"`
	TotalPages           int             `json:"totalPages"`
	TotalItemCount       int             `json:"totalItemCount"`
	Products             []Product        `json:"products"`
}

// Product represents a single truck listing from the API
type Product struct {
	Type              string        `json:"$type,omitempty"`
	ProductCategory   string        `json:"productCategory,omitempty"`
	StockNumber       string        `json:"stockNumber"`
	Status            string        `json:"status,omitempty"`
	Year              int           `json:"year"`
	LocationCity      string        `json:"locationCity,omitempty"`
	LocationState     string        `json:"locationState,omitempty"`
	TruckType         string        `json:"truckType,omitempty"`
	TruckTypeOption   string        `json:"truckTypeOption,omitempty"`
	Mileage           MileageValue  `json:"mileage,omitempty"`
	EcmMileage        MileageValue  `json:"ecmMileage,omitempty"`
	Engine            Engine        `json:"engine,omitempty"`
	Axle              Axle          `json:"axle,omitempty"`
	Sleeper           Sleeper       `json:"sleeper,omitempty"`
	Transmission      Transmission  `json:"transmission,omitempty"`
	Make              string        `json:"make"`
	MakeOption        string        `json:"makeOption,omitempty"`
	Model             string        `json:"model"`
	ModelOption       string        `json:"modelOption,omitempty"`
	FleetCode         string        `json:"fleetCode,omitempty"`
	Suspension        string        `json:"suspension,omitempty"`
	Wheelbase         WheelbaseValue `json:"wheelbase,omitempty"`
	SteerWheelType    string        `json:"steerWheelType,omitempty"`
	RearWheelType     string        `json:"rearWheelType,omitempty"`
	FrontAxleRating   AxleRating    `json:"frontAxleRating,omitempty"`
	RearAxleRating    AxleRating    `json:"rearAxleRating,omitempty"`
	TireSize          string        `json:"tireSize,omitempty"`
	Fairings          string        `json:"fairings,omitempty"`
	HasFcam           bool          `json:"hasFcam,omitempty"`
	HasApu            bool          `json:"hasApu,omitempty"`
	HasJakeBrake      bool          `json:"hasJakeBrake,omitempty"`
	FifthWheel        string        `json:"fifthWheel,omitempty"`
	Brakes            string        `json:"brakes,omitempty"`
	FuelTankCapacity  FuelTankValue `json:"fuelTankCapacity,omitempty"`
	FuelTankCount     int           `json:"fuelTankCount,omitempty"`
	Media             []MediaProvider `json:"media,omitempty"`
	CanReserve        bool          `json:"canReserve,omitempty"`
	HasOffers         bool          `json:"hasOffers,omitempty"`
	IsAsIs            bool          `json:"isAsIs,omitempty"`
	IsJustArrived     bool          `json:"isJustArrived,omitempty"`
	IsLateLow         bool          `json:"isLateLow,omitempty"`
	IsBudget          bool          `json:"isBudget,omitempty"`
	Price             PriceValue    `json:"price,omitempty"`
	OldPrice          PriceValue    `json:"oldPrice,omitempty"`
	LowestPrice       PriceValue    `json:"lowestPrice,omitempty"`
	PriceChanged      bool          `json:"priceChanged,omitempty"`
	Headline          string        `json:"headline,omitempty"`
	Description       string        `json:"description,omitempty"`
	BranchID          string        `json:"branchID,omitempty"`
	Country           string        `json:"country,omitempty"`
	Industries        []string      `json:"industries,omitempty"`
	IsFeatured        bool          `json:"isFeatured,omitempty"`
}

// PriceValue represents a price with currency
type PriceValue struct {
	Value        float64 `json:"value"`
	CurrencyCode string  `json:"currencyCode,omitempty"`
}

// MileageValue represents mileage with unit
type MileageValue struct {
	Value int    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// WheelbaseValue represents wheelbase with unit
type WheelbaseValue struct {
	Value int    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// AxleRating represents axle rating with unit
type AxleRating struct {
	Value int    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// FuelTankValue represents fuel tank capacity with unit
type FuelTankValue struct {
	Value int    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// Engine represents engine specifications
type Engine struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	Horsepower   int    `json:"horsepower,omitempty"`
}

// Axle represents axle specifications
type Axle struct {
	Name string `json:"name,omitempty"`
	Ratio int   `json:"ratio,omitempty"`
}

// Sleeper represents sleeper specifications
type Sleeper struct {
	Type      string        `json:"type,omitempty"`
	Size      SleeperSize   `json:"size,omitempty"`
	BunkCount int           `json:"bunkCount,omitempty"`
}

// SleeperSize represents sleeper size with unit
type SleeperSize struct {
	Value int    `json:"value"`
	Unit  string `json:"unit,omitempty"`
}

// Transmission represents transmission specifications
type Transmission struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"name,omitempty"`
	Speed string `json:"speed,omitempty"`
}

// MediaProvider represents a media provider (Arrow, Glo3D, etc.)
type MediaProvider struct {
	ProviderName string      `json:"providerName"`
	Media        MediaContent `json:"media"`
}

// MediaContent represents media content from a provider
type MediaContent struct {
	Photos []Photo                `json:"photos,omitempty"`
	Videos []interface{}          `json:"videos,omitempty"`
	Docs  []interface{}          `json:"docs,omitempty"`
	VIN   string                 `json:"vin,omitempty"` // For Glo3D provider
	// Glo3D has additional fields but we only need VIN for now
}

// Photo represents a photo with URLs
type Photo struct {
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	MediaURL     string `json:"mediaUrl,omitempty"`
}
