package types

// Region represents a geographic datacenter location.
type Region struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// FlavorProduct is an individual hardware configuration inside a FlavorGroup.
type FlavorProduct struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	RegionName     string  `json:"region_name"`
	RegionID       string  `json:"region_id,omitempty"`
	Price          float64 `json:"price"`
	CPU            int     `json:"cpu"`
	Disk           int     `json:"disk"`                // GB root disk
	Ephemeral      int     `json:"ephemeral,omitempty"` // GB temporary disk
	RAM            int     `json:"ram"`                 // GB
	GPU            string  `json:"gpu"`
	GPUCount       int     `json:"gpu_count"`
	Bandwidth      string  `json:"bandwidth,omitempty"`
	StockAvailable bool    `json:"stock_available,omitempty"`
	Cycle          string  `json:"cycle,omitempty"` // e.g. "hourly"
}

// FlavorGroup groups FlavorProducts by region as returned by GET /flavors.
type FlavorGroup struct {
	Region   string          `json:"region"`
	Products []FlavorProduct `json:"products"`
}

// Image represents an OS image available for VM deployment.
type Image struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	RegionName  string `json:"region_name"`
	RegionID    string `json:"region_id,omitempty"`
}
